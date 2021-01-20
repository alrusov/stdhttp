package jwt

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/stdhttp/auth"
)

// AuthHandler --
type AuthHandler struct {
	cfg *config.Listener
}

const method = "Bearer"

//----------------------------------------------------------------------------------------------------------------------------//

// Init --
func (ah *AuthHandler) Init(cfg *config.Listener) error {
	ah.cfg = cfg
	return nil
}

// Enabled --
func (ah *AuthHandler) Enabled() bool {
	return ah.cfg.JWTsecret != ""
}

// WWWAuthHeader --
func (ah *AuthHandler) WWWAuthHeader() (name string, withRealm bool) {
	return method, true
}

// Check --
func (ah *AuthHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *auth.Identity, tryNext bool) {
	if ah.cfg.JWTsecret == "" {
		return nil, true
	}

	u := ""

	code, msg := func() (code int, msg string) {
		code = http.StatusNoContent
		msg = ""

		s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(s) != 2 || s[0] != method {
			return
		}

		code = http.StatusForbidden

		keyFunc := func(t *jwt.Token) (interface{}, error) {
			return []byte(ah.cfg.JWTsecret), nil
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

		u, _ = ui.(string)
		_, exists = ah.cfg.Users[u]
		if !exists {
			msg = fmt.Sprintf(`Unknown user "%v"`, ui)
			return
		}

		code = http.StatusOK
		return
	}()

	if code == http.StatusOK {
		return &auth.Identity{
				Method: method,
				User:   u,
				Extra:  nil,
			},
			false
	}

	if code == http.StatusNoContent {
		return nil, true
	}

	log.Message(log.INFO, `[%d] JWT login error: %s`, id, msg)

	return nil, false
}

//----------------------------------------------------------------------------------------------------------------------------//

// claims --
type claims struct {
	User string `json:"username"`
	Exp  int64  `json:"exp"`
}

// Valid --
func (c claims) Valid() error {
	return nil
}

//----------------------------------------------------------------------------------------------------------------------------//

// GetToken --
func GetToken(cfg *config.Listener, id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
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

		claims := claims{
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
