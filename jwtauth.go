package stdhttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/dgrijalva/jwt-go"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) jwtAuthHandler(id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
	return JWTauthHandler(h.listenerCfg, id, path, w, r)
}

// JWTauthHandler --
func JWTauthHandler(cfg *config.Listener, id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
	code, msg := func() (code int, msg string) {
		code = http.StatusForbidden
		msg = ""

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			msg = `Missing authorization header`
			return
		}

		s := strings.Split(authHeader, " ")
		if len(s) != 2 || s[0] != "Bearer" {
			msg = `Illegal authorization header`
			return
		}

		keyFunc := func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTsecret), nil
		}

		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(s[1], claims, keyFunc)

		if err != nil {
			msg = err.Error()
			return
		}

		ui, exists := claims["username"]
		if !exists {
			msg = `The "username" claim is not found in the authorization header`
			return
		}

		u, _ := ui.(string)
		_, exists = cfg.Users[u]
		if !exists {
			msg = fmt.Sprintf(`Unknown user "%v"`, ui)
			return
		}

		code = 0
		return
	}()

	if code == 0 {
		return true
	}

	log.Message(log.INFO, `[%d] Login error: %s`, id, msg)

	w.WriteHeader(code)
	w.Write([]byte("Forbidden"))

	return false
}

//----------------------------------------------------------------------------------------------------------------------------//

// jwtClaims --
type jwtClaims struct {
	User string `json:"username"`
	Exp  int64  `json:"exp"`
}

// Valid --
func (c jwtClaims) Valid() error {
	return nil
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) jwtLogin(id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
	return JWTloginHandler(h.listenerCfg, id, path, w, r)
}

// JWTloginHandler --
func JWTloginHandler(cfg *config.Listener, id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
	code, msg := func() (code int, msg string) {
		code = http.StatusForbidden
		msg = ""

		if cfg.JWTsecret == "" {
			msg = `JWT auth is disabled`
			return
		}

		queryParams := r.URL.Query()
		u := queryParams.Get("u")
		if u == "" {
			msg = `Empty username`
			return
		}
		p := queryParams.Get("p")

		password, exists := cfg.Users[u]
		if !exists || password != p {
			msg = fmt.Sprintf(`Illegal login or password for "%s"`, u)
			return
		}

		claims := jwtClaims{
			User: u,
			Exp:  time.Now().Add(time.Duration(cfg.JWTlifetime) * time.Second).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		msg, err := token.SignedString([]byte(cfg.JWTsecret))
		if err != nil {
			msg = err.Error()
			return
		}

		code = http.StatusOK
		return
	}()

	tp := ""
	if code != http.StatusOK {
		tp = " error"
	}

	log.Message(log.DEBUG, `[%d] JWT token%s: %s`, id, tp, msg)

	w.WriteHeader(code)
	w.Write([]byte(msg))

	return false
}

//----------------------------------------------------------------------------------------------------------------------------//
