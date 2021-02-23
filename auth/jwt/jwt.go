package jwt

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"

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
		Secret   string `toml:"secret"`
		Lifetime int    `toml:"lifetime"`
	}
)

const (
	module = "jwt"
	method = "Bearer"
)

//----------------------------------------------------------------------------------------------------------------------------//

func init() {
	config.AddAuthMethod(module, &methodOptions{}, checkConfig)
}

func checkConfig(m *config.AuthMethod) (err error) {
	msgs := misc.NewMessages()

	options, ok := m.Options.(*methodOptions)
	if !ok {
		msgs.Add(`%s.checkConfig: Options is "%T", expected "%T"`, module, m.Options, options)
	}

	if !m.Enabled {
		return
	}

	if options.Secret == "" {
		msgs.Add(`%s.checkConfig: secret parameter isn't defined"`, module)
	}

	if options.Secret == "" {
		msgs.Add(`%s.checkConfig: secret parameter isn't defined"`, module)
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

	if options.Secret == "" {
		return fmt.Errorf(`Secret for module "%s" cannot be empty`, module)
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
	return method, false
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (ah *AuthHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *auth.Identity, tryNext bool) {
	if ah.options.Secret == "" {
		return nil, true
	}

	var userDef config.User
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
			return []byte(ah.options.Secret), nil
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
		userDef, exists = ah.authCfg.Users[u]
		if !exists {
			msg = fmt.Sprintf(`Unknown user "%v"`, ui)
			return
		}

		code = http.StatusOK
		return
	}()

	if code == http.StatusOK {
		return &auth.Identity{
				Method: module,
				User:   u,
				Groups: userDef.Groups,
				Extra:  nil,
			},
			false
	}

	if code == http.StatusNoContent {
		return nil, true
	}

	auth.Log.Message(log.INFO, `[%d] JWT login error: %s`, id, msg)

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

		methodCfg, exists := cfg.Auth.Methods[module]
		if !exists || !methodCfg.Enabled || methodCfg.Options == nil {
			msg = `JWT auth is disabled`
			return
		}

		options, ok := methodCfg.Options.(*methodOptions)
		if !ok || options.Secret == "" {
			msg = fmt.Sprintf(`Method "%s" is misconfigured`, module)
			return
		}

		queryParams := r.URL.Query()
		u := queryParams.Get("u")
		if u == "" {
			msg = `Empty username`
			return
		}
		p := queryParams.Get("p")

		userDef, exists := cfg.Auth.Users[u]
		if !exists || userDef.Password != string(auth.Hash([]byte(p), []byte(u))) {
			msg = fmt.Sprintf(`Illegal login or password for "%s"`, u)
			return
		}

		claims := claims{
			User: u,
			Exp:  time.Now().Add(time.Duration(options.Lifetime) * time.Second).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		msg, err := token.SignedString([]byte(options.Secret))
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

	auth.Log.Message(log.DEBUG, `[%d] JWT token%s: %s`, id, tp, msg)

	w.WriteHeader(code)
	w.Write([]byte(msg))

	return false
}

//----------------------------------------------------------------------------------------------------------------------------//
