package stdhttp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alrusov/log"
	"github.com/dgrijalva/jwt-go"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) jwtAuthHandler(id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
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
			return h.listenerCfg.JWTsecret, nil
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
		_, exists = h.listenerCfg.Users[u]
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
