package basic

import (
	"fmt"
	"net/http"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/stdhttp/auth"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// AuthHandler --
	AuthHandler struct {
		cfg *config.Listener
	}
)

const method = "Basic"

//----------------------------------------------------------------------------------------------------------------------------//

// Init --
func (ah *AuthHandler) Init(cfg *config.Listener) error {
	ah.cfg = cfg
	return nil
}

// Enabled --
func (ah *AuthHandler) Enabled() bool {
	return ah.cfg.BasicAuthEnabled
}

// WWWAuthHeader --
func (ah *AuthHandler) WWWAuthHeader() (name string, withRealm bool) {
	return method, true
}

// Check --
func (ah *AuthHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *auth.Identity, tryNext bool) {
	if !ah.cfg.BasicAuthEnabled {
		return nil, true
	}

	u, p, ok := r.BasicAuth()
	if !ok {
		return nil, true
	}

	err := ah.checkBasicLogin(u, p)

	if err == nil {
		return &auth.Identity{
				Method: method,
				User:   u,
				Extra:  nil,
			},
			false
	}

	log.Message(log.INFO, `[%d] Basic login error: %v`, id, err)

	return nil, false
}

//----------------------------------------------------------------------------------------------------------------------------//

func (ah *AuthHandler) checkBasicLogin(u string, p string) error {
	password, exists := ah.cfg.Users[u]
	if exists && password == p {
		return nil
	}

	return fmt.Errorf(`Illegal login or password for "%s"`, u)
}

//----------------------------------------------------------------------------------------------------------------------------//
