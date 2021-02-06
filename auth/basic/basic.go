package basic

import (
	"fmt"
	"net/http"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
	"github.com/alrusov/stdhttp/auth"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// AuthHandler --
	AuthHandler struct {
		authCfg *config.Auth
		cfg     *config.AuthMethod
		options *methodOptions
	}

	methodOptions struct {
	}
)

const (
	module = "basic"
	method = "Basic"
)

//----------------------------------------------------------------------------------------------------------------------------//

func init() {
	config.AddAuthMethod(module, &methodOptions{}, checkConfig)
}

func checkConfig(m *config.AuthMethod) (err error) {
	msgs := misc.NewMessages()

	options, ok := m.Options.(*methodOptions)
	if !ok {
		msgs.Add(`%s.checkConfig: Options is "%T", expected "%T"`, method, m.Options, options)
	}

	err = msgs.Error()
	return
}

//----------------------------------------------------------------------------------------------------------------------------//

// Init --
func (ah *AuthHandler) Init(cfg *config.Listener) (err error) {
	ah.authCfg = nil
	ah.cfg = nil
	ah.options = nil

	methodCfg, exists := cfg.Auth.Methods[module]
	if !exists || !methodCfg.Enabled || methodCfg.Options == nil {
		return nil
	}

	options, ok := methodCfg.Options.(*methodOptions)
	if !ok {
		return fmt.Errorf(`Options for module "%s" is "%T", expected "%T"`, module, methodCfg.Options, options)
	}

	ah.authCfg = &cfg.Auth
	ah.cfg = methodCfg
	ah.options = options
	return nil
}

//----------------------------------------------------------------------------------------------------------------------------//

// Enabled --
func (ah *AuthHandler) Enabled() bool {
	return ah.cfg != nil && ah.cfg.Enabled
}

//----------------------------------------------------------------------------------------------------------------------------//

// Score --
func (ah *AuthHandler) Score() int {
	return ah.cfg.Score
}

//----------------------------------------------------------------------------------------------------------------------------//

// WWWAuthHeader --
func (ah *AuthHandler) WWWAuthHeader() (name string, withRealm bool) {
	return method, true
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (ah *AuthHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *auth.Identity, tryNext bool) {
	if ah.cfg == nil || !ah.cfg.Enabled {
		return nil, true
	}

	u, p, ok := r.BasicAuth()
	if !ok {
		return nil, true
	}

	userDef, exists := ah.authCfg.Users[u]
	if !exists {
		log.Message(log.INFO, `[%d] Basic login error: user "%s" not found`, id, u)
		return nil, false
	}

	if userDef.Password != string(misc.Sha512Hash([]byte(p))) {
		log.Message(log.INFO, `[%d] Basic login error: illegal password for "%s"`, id, u)
		return nil, false
	}

	return &auth.Identity{
			Method: module,
			User:   u,
			Groups: userDef.Groups,
			Extra:  nil,
		},
		false
}

//----------------------------------------------------------------------------------------------------------------------------//
