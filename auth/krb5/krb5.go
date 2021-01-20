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
	"github.com/alrusov/stdhttp/auth"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// AuthHandler --
	AuthHandler struct {
		cfg *config.Listener
		kt  *keytab.Keytab
	}
)

const method = spnego.HTTPHeaderAuthResponseValueKey

//----------------------------------------------------------------------------------------------------------------------------//

// Init --
func (ah *AuthHandler) Init(cfg *config.Listener) (err error) {
	ah.cfg = cfg
	ah.kt = nil

	if cfg.Krb5KeyFile == "" {
		return
	}

	ah.kt, err = keytab.Load(cfg.Krb5KeyFile)
	if err != nil {
		ah.kt = nil
		return
	}

	return
}

// Enabled --
func (ah *AuthHandler) Enabled() bool {
	return ah.kt != nil
}

// WWWAuthHeader --
func (ah *AuthHandler) WWWAuthHeader() (name string, withRealm bool) {
	return method, false
}

// Check --
func (ah *AuthHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *auth.Identity, tryNext bool) {
	if ah.kt == nil {
		return nil, true
	}

	goIdentity, err := ah.negotiate(r)

	if goIdentity != nil {
		return &auth.Identity{
				Method: method,
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
