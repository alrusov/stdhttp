package auth

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/alrusov/config"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// Handlers --
	Handlers struct {
		mutex *sync.RWMutex
		list  []Handler
	}

	// Handler --
	Handler interface {
		Init(cfg *config.Listener) error
		Enabled() bool
		WWWAuthHeader() (name string, withRealm bool)
		Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *Identity, tryNext bool)
	}

	// Identity --
	Identity struct {
		Method string
		User   string
		Extra  interface{}
	}
)

//----------------------------------------------------------------------------------------------------------------------------//

// NewHandlers --
func NewHandlers() *Handlers {
	return &Handlers{
		mutex: new(sync.RWMutex),
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// Add --
func (hh *Handlers) Add(cfg *config.Listener, ah Handler) (err error) {
	hh.mutex.Lock()
	defer hh.mutex.Unlock()

	err = ah.Init(cfg)
	if err != nil {
		return
	}

	if ah.Enabled() {
		hh.list = append(hh.list, ah)
		return
	}

	return
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
func (hh *Handlers) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *Identity, code int, msg string) {
	hh.mutex.RLock()
	defer hh.mutex.RUnlock()

	code = 0

	if len(hh.list) == 0 {
		return
	}

	tryNext := false

	for _, ah := range hh.list {
		identity, tryNext = ah.Check(id, prefix, path, w, r)
		if identity != nil {
			return
		}

		if !tryNext {
			break
		}
	}

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
