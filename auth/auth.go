package auth

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// Handlers --
	Handlers struct {
		mutex *sync.RWMutex
		cfg   *config.Listener
		list  []Handler
	}

	// Handler --
	Handler interface {
		Init(cfg *config.Listener) error
		Enabled() bool
		Score() int
		WWWAuthHeader() (name string, withRealm bool)
		Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *Identity, tryNext bool)
	}

	// Identity --
	Identity struct {
		Method string
		User   string
		Groups []string
		Extra  interface{}
	}
)

var (
	// Log --
	Log = log.NewFacility("stdhttp.auth")
)

//----------------------------------------------------------------------------------------------------------------------------//

// NewHandlers --
func NewHandlers(cfg *config.Listener) *Handlers {
	return &Handlers{
		mutex: new(sync.RWMutex),
		cfg:   cfg,
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// Add --
func (hh *Handlers) Add(ah Handler) (err error) {
	hh.mutex.Lock()
	defer hh.mutex.Unlock()

	err = ah.Init(hh.cfg)
	if err != nil {
		return
	}

	if ah.Enabled() {
		hh.add(ah)
		return
	}

	return
}

func (hh *Handlers) add(ah Handler) {
	ln := len(hh.list)

	if ln == 0 {
		hh.list = []Handler{ah}
		return
	}

	score := ah.Score()

	i := 0
	for ; i < ln; i++ {
		if hh.list[i].Score() > score {
			break
		}
	}

	if i == 0 {
		hh.list = append([]Handler{ah}, hh.list...)
		return
	}

	if i == ln {
		hh.list = append(hh.list, ah)
		return
	}

	hh.list = append(hh.list, nil)
	copy(hh.list[i+1:], hh.list[i:])
	hh.list[i] = ah
}

//----------------------------------------------------------------------------------------------------------------------------//

// Enabled --
func (hh *Handlers) Enabled() bool {
	hh.mutex.RLock()
	defer hh.mutex.RUnlock()

	return len(hh.list) > 0
}

//----------------------------------------------------------------------------------------------------------------------------//

// WriteAuthRequestHeaders --
func (hh *Handlers) WriteAuthRequestHeaders(w http.ResponseWriter, prefix string, path string) {
	hh.mutex.RLock()
	defer hh.mutex.RUnlock()

	if len(hh.list) == 0 {
		return
	}

	for _, ah := range hh.list {
		name, withRealm := ah.WWWAuthHeader()
		if name == "" {
			continue
		}

		s := name
		if withRealm {
			s = fmt.Sprintf(`%s realm="%s%s"`, name, prefix, path)
		}

		w.Header().Add("WWW-Authenticate", s)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (hh *Handlers) Check(id uint64, prefix string, path string, permissions misc.BoolMap, w http.ResponseWriter, r *http.Request) (identity *Identity, code int, msg string) {
	hh.mutex.RLock()
	defer hh.mutex.RUnlock()

	code = 0

	if len(hh.list) == 0 {
		return
	}

	tryNext := false

	for _, ah := range hh.list {
		identity, tryNext = ah.Check(id, prefix, path, w, r)

		if identity != nil && identity.checkPermissions(permissions) {
			return
		}

		if !tryNext {
			break
		}
	}

	identity = nil

	if tryNext {
		code = http.StatusUnauthorized
		msg = "Unauthorised"
		return
	}

	code = http.StatusForbidden
	msg = "Forbidden"
	return
}

//----------------------------------------------------------------------------------------------------------------------------//

func (identity *Identity) checkPermissions(permissions misc.BoolMap) bool {
	if len(permissions) == 0 {
		return false
	}

	user := identity.User

	p, exists := permissions[user]
	if exists {
		return p
	}

	if len(identity.Groups) > 0 {
		enabled := false

		for _, g := range identity.Groups {
			p, exists := permissions["@"+g]
			if exists {
				if !p {
					return false
				}
				enabled = true
			}
		}

		if enabled {
			return true
		}
	}

	p, exists = permissions["*"]
	if exists {
		return p
	}

	return false
}

//----------------------------------------------------------------------------------------------------------------------------//

// Hash --
func Hash(p []byte, salt []byte) []byte {
	return misc.Sha512Hash(append(p, salt...))
}

//----------------------------------------------------------------------------------------------------------------------------//
