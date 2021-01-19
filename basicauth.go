package stdhttp

import (
	"fmt"
	"net/http"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// BasicAuthHandler --
type BasicAuthHandler struct {
	cfg *config.Listener
}

//----------------------------------------------------------------------------------------------------------------------------//

// Init --
func (ah *BasicAuthHandler) Init(cfg *config.Listener) {
	ah.cfg = cfg
}

// Enabled --
func (ah *BasicAuthHandler) Enabled() bool {
	return ah.cfg.BasicAuthEnabled
}

// WWWAuthHeader --
func (ah *BasicAuthHandler) WWWAuthHeader() (name string, withRealm bool) {
	return "Basic", true
}

// Check --
func (ah *BasicAuthHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (valid bool, tryNext bool) {
	if !ah.cfg.BasicAuthEnabled {
		return false, true
	}

	u, p, ok := r.BasicAuth()
	if !ok {
		return false, true
	}

	err := ah.checkBasicLogin(u, p)

	if err == nil {
		log.Message(log.DEBUG, `[%d] User %q logged in (Basic)`, id, u)
		return true, false
	}

	log.Message(log.INFO, `[%d] Basic login error: %s`, id, err.Error())

	return false, false
}

//----------------------------------------------------------------------------------------------------------------------------//

func (ah *BasicAuthHandler) checkBasicLogin(u string, p string) error {
	password, exists := ah.cfg.Users[u]
	if exists && password == p {
		return nil
	}

	return fmt.Errorf(`Illegal login or password for "%s"`, u)
}

//----------------------------------------------------------------------------------------------------------------------------//
