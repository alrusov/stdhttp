package krb5

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/jcmturner/goidentity.v3"
	"gopkg.in/jcmturner/gokrb5.v7/gssapi"
	"gopkg.in/jcmturner/gokrb5.v7/keytab"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"

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
		kt      *keytab.Keytab
	}

	methodOptions struct {
		KeyFile string `toml:"key-file"`
	}
)

const (
	module = "krb5"
	method = spnego.HTTPHeaderAuthResponseValueKey
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

	if strings.TrimSpace(options.KeyFile) == "" {
		msgs.Add(`%s.checkConfig: key-file parameter isn't defined"`, module)
	}

	options.KeyFile, err = misc.AbsPath(options.KeyFile)
	if err != nil {
		return
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

	if options.KeyFile == "" {
		return fmt.Errorf(`Keyfile for module "%s" cannot be empty`, module)
	}

	options.KeyFile, err = misc.AbsPath(options.KeyFile)
	if err != nil {
		return fmt.Errorf(`Auth module "%s" keyfile: %s`, module, err.Error())
	}

	ah.kt, err = keytab.Load(options.KeyFile)
	if err != nil {
		ah.kt = nil
		return
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

// WWWAuthHeader --
func (ah *AuthHandler) WWWAuthHeader() (name string, withRealm bool) {
	return method, false
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (ah *AuthHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *auth.Identity, tryNext bool) {
	if ah.kt == nil {
		return nil, true
	}

	goIdentity, err := ah.negotiate(r)

	if goIdentity != nil {
		return &auth.Identity{
				Method: module,
				User:   goIdentity.UserName(),
				Extra:  goIdentity,
			},
			false
	}

	if err != nil {
		log.Message(log.INFO, `[%d] Krb5 login error: %v`, id, err)
		return nil, false
	}

	return nil, true
}

//----------------------------------------------------------------------------------------------------------------------------//

func (ah *AuthHandler) negotiate(r *http.Request) (identity goidentity.Identity, err error) {
	// Get the auth header
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != method {
		return
	}

	// Decode the header into an SPNEGO context token
	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		err = fmt.Errorf("Error in base64 decoding negotiation header: %v", err)
		return
	}

	var st spnego.SPNEGOToken
	err = st.Unmarshal(b)
	if err != nil {
		err = fmt.Errorf("Error in unmarshaling SPNEGO token: %v", err)
		return
	}

	// Set up the SPNEGO GSS-API mechanism
	serv := spnego.SPNEGOService(ah.kt)

	// Validate the context token
	authed, ctx, status := serv.AcceptSecContext(&st)
	if status.Code != gssapi.StatusComplete && status.Code != gssapi.StatusContinueNeeded {
		err = fmt.Errorf("Validation error: %v", status)
		return
	}

	if status.Code == gssapi.StatusContinueNeeded {
		err = fmt.Errorf("GSS-API continue needed")
		return
	}

	if !authed {
		err = fmt.Errorf("Kerberos authentication failed")
		return
	}

	ii := ctx.Value(spnego.CTXKeyCredentials)
	identity, ok := ii.(goidentity.Identity)
	if !ok {
		err = fmt.Errorf("Bad identity type (%T instead %T)", ii, identity)
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
