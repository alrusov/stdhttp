package stdhttp

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
	"github.com/alrusov/panic"
)

//----------------------------------------------------------------------------------------------------------------------------//

// HTTP --
type HTTP struct {
	ListenerCfg *config.Listener
	Mutex       *sync.Mutex
	srv         *http.Server

	handler Handler
}

// Handler --
type Handler interface {
	Handler(id uint64, path string, w http.ResponseWriter, r *http.Request) bool
}

var (
	connectionID = uint64(0)
)

//----------------------------------------------------------------------------------------------------------------------------//

// NewListener --
func NewListener(listenerCfg *config.Listener, handler Handler) (*HTTP, error) {
	h := &HTTP{
		ListenerCfg: listenerCfg,
		Mutex:       new(sync.Mutex),
		handler:     handler,
	}

	timeout := time.Duration(listenerCfg.Timeout) * time.Second
	h.srv = &http.Server{
		Addr:              listenerCfg.Addr,
		Handler:           h,
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}

	log.Message(log.INFO, `Listener created on "%s"`, listenerCfg.Addr)

	return h, nil
}

// Start --
func (h *HTTP) Start() error {
	var err error
	cert := strings.TrimSpace(h.ListenerCfg.SSLCombinedPem)

	if cert == "" {
		err = h.srv.ListenAndServe()
	} else {
		err = h.srv.ListenAndServeTLS(cert, cert)
	}

	if !misc.AppStarted() {
		err = nil
	}
	return err
}

// Stop --
func (h *HTTP) Stop() error {
	misc.StopApp(0)
	return h.srv.Close()
}

//----------------------------------------------------------------------------------------------------------------------------//

// ServeHTTP --
func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t0 := misc.NowUTC().UnixNano()

	defer panic.SaveStackToLog()

	id := atomic.AddUint64(&connectionID, 1)

	defer misc.LogProcessingTime("", id, "http", "", t0)

	log.Message(log.DEBUG, `[%d] New request %q from %q`, id, r.RequestURI, r.RemoteAddr)
	defer log.Message(log.TRACE1, "[%d] Finished", id)

	if !misc.AppStarted() {
		Error(id, false, w, http.StatusInternalServerError, "Server stopped", nil)
		return
	}

	path := NormalizeSlashes(r.URL.Path)

	if h.handler.Handler(id, path, w, r) {
		return
	}

	switch path {
	case "":
		root(w)

	case "/info":
		showInfo(w)

	case "/ping":
		w.Header().Add("X-Application-Version", fmt.Sprintf("%s %s", misc.AppName(), misc.AppVersion()))
		w.WriteHeader(http.StatusNoContent)

	case "/set-log-level":
		ChangeLogLevel(id, w, r)

	default:
		Error(id, false, w, http.StatusNotFound, `Invalid endpoint "`+path+`"`, nil)
	}
	return
}

//----------------------------------------------------------------------------------------------------------------------------//
