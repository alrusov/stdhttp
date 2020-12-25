package stdhttp

import (
	"fmt"
	"net/http"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) basicAuthHandler(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) bool {
	return BasicAuthHandler(h.listenerCfg, id, path, w, r)
}

// BasicAuthHandler --
func BasicAuthHandler(cfg *config.Listener, id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
	u, p, ok := r.BasicAuth()
	if !ok {
		log.Message(log.DEBUG, `[%d] No authentication information in request`, id)
	} else if err := checkBasicLogin(cfg, u, p); err != nil {
		log.Message(log.INFO, `[%d] Login error: %s`, id, err.Error())
	} else {
		log.Message(log.DEBUG, `[%d] User %q logged in`, id, u)
		return true
	}

	w.Header().Set("WWW-Authenticate", "Basic realm="+path+"/")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Authentication needed"))

	return false
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
