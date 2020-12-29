package stdhttp

import (
	"fmt"
	"net/http"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// BasicAuthHandler --
func BasicAuthHandler(cfg *config.Listener, id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (valid bool, tryNext bool) {
	if !cfg.BasicAuthEnabled {
		return false, true
	}

	u, p, ok := r.BasicAuth()
	if !ok {
		return false, true
	}

	err := checkBasicLogin(cfg, u, p)

	if err == nil {
		log.Message(log.DEBUG, `[%d] User %q logged in (Basic)`, id, u)
		return true, false
	}

	log.Message(log.INFO, `[%d] Basic login error: %s`, id, err.Error())

	return false, false
}

//----------------------------------------------------------------------------------------------------------------------------//

func checkBasicLogin(cfg *config.Listener, u string, p string) error {
	password, exists := cfg.Users[u]
	if exists && password == p {
		return nil
	}

	return fmt.Errorf(`Illegal login or password for "%s"`, u)
}

//----------------------------------------------------------------------------------------------------------------------------//

// BasicAuthRequest --
func BasicAuthRequest(w http.ResponseWriter, path string) {
	w.Header().Set("WWW-Authenticate", "Basic realm="+path+"/")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Authentication needed"))
}

//----------------------------------------------------------------------------------------------------------------------------//
