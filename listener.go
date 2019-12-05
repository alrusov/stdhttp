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

type (
	// HTTP --
	HTTP struct {
		listenerCfg       *config.Listener
		commonConfig      *config.Common
		mutex             *sync.Mutex
		srv               *http.Server
		handler           Handler
		extraFunc         ExtraInfoFunc
		info              *infoBlock
		connectionID      uint64
		extraRootItemFunc ExtraRootItemFunc
	}

	// Handler --
	Handler interface {
		Handler(id uint64, path string, w http.ResponseWriter, r *http.Request) bool
	}

	// ExtraRootItemFunc --
	ExtraRootItemFunc func() []string
)

//----------------------------------------------------------------------------------------------------------------------------//

// NewListener --
func NewListener(listenerCfg *config.Listener, handler Handler) (*HTTP, error) {
	h := &HTTP{
		listenerCfg:  listenerCfg,
		commonConfig: config.GetCommon(),
		mutex:        new(sync.Mutex),
		handler:      handler,
		extraFunc:    ExtraInfoFunc(nil),
		info:         &infoBlock{},
		connectionID: 0,
	}

	timeout := time.Duration(listenerCfg.Timeout) * time.Second
	h.srv = &http.Server{
		Addr:              listenerCfg.Addr,
		Handler:           h,
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}

	h.initInfo()

	log.Message(log.INFO, `Listener created on "%s"`, listenerCfg.Addr)

	return h, nil
}

// Start --
func (h *HTTP) Start() error {
	var err error
	cert := strings.TrimSpace(h.listenerCfg.SSLCombinedPem)

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

	id := atomic.AddUint64(&h.connectionID, 1)
	log.Message(log.DEBUG, `[%d] New request %q from %q`, id, r.RequestURI, r.RemoteAddr)

	if !misc.AppStarted() {
		Error(id, false, w, http.StatusInternalServerError, "Server stopped", nil)
		return
	}

	processed := true
	path := NormalizeSlashes(r.URL.Path)

	defer func() {
		if !processed {
			path = url404
		} else if path == "" {
			path = "/"
		}
		h.info.Runtime.Requests.inc()
		h.updateEndpointStat(path)
		misc.LogProcessingTime("", id, "http", "", t0)
	}()

	if h.handler.Handler(id, path, w, r) {
		return
	}

	if h.profiler(id, path, w, r) {
		return
	}

	switch path {
	case "":
		h.root(id, path, w, r)

	case "/info":
		h.showInfo(id, path, w, r)

	case "/ping":
		w.Header().Add("X-Application-Version", fmt.Sprintf("%s %s", misc.AppName(), misc.AppVersion()))
		w.WriteHeader(http.StatusNoContent)

	case "/set-log-level":
		h.changeLogLevel(id, path, w, r)

	case "/profiler-enable":
		h.commonConfig.ProfilerEnabled = true
		ReturnRefresh(w, r, http.StatusNoContent)

	case "/profiler-disable":
		h.commonConfig.ProfilerEnabled = false
		ReturnRefresh(w, r, http.StatusNoContent)

	default:
		processed = false
		Error(id, false, w, http.StatusNotFound, `Invalid endpoint "`+path+`"`, nil)
	}
	return
}

//----------------------------------------------------------------------------------------------------------------------------//
