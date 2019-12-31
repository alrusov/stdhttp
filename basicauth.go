package stdhttp

import (
	"fmt"
	"net/http"

	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) basicAuthHandler(id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
	u, p, ok := r.BasicAuth()
	if !ok {
	} else if err := h.checkLogin(u, p); err != nil {
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

func (h *HTTP) checkLogin(u string, p string) error {
	password, exists := h.listenerCfg.Users[u]
	if exists && password == p {
		return nil
	}

	return fmt.Errorf(`Illegal login or password for %q`, u)
}

//----------------------------------------------------------------------------------------------------------------------------//
