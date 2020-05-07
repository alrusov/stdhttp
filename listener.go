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
		connectionID      uint64
		listenerCfg       *config.Listener
		commonConfig      *config.Common
		mutex             *sync.Mutex
		srv               *http.Server
		handler           Handler
		extraFunc         ExtraInfoFunc
		info              *infoBlock
		extraRootItemFunc ExtraRootItemFunc
		movedPathsFwd     misc.StringMap
		movedPathsRev     misc.StringMap
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
		listenerCfg:   listenerCfg,
		commonConfig:  config.GetCommon(),
		mutex:         new(sync.Mutex),
		handler:       handler,
		extraFunc:     ExtraInfoFunc(nil),
		info:          &infoBlock{},
		connectionID:  0,
		movedPathsFwd: make(misc.StringMap),
		movedPathsRev: make(misc.StringMap),
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

	realIP := r.Header.Get("X-Real-IP")
	if realIP == "" {
		realIP = r.Header.Get("X-Forwarded-For")
		if realIP == "" {
			realIP = r.RemoteAddr
		}
	}

	log.SecuredMessage(log.DEBUG, logReplaceRequest, `[%d] New request %q from %s`, id, r.RequestURI, realIP)

	if !misc.AppStarted() {
		Error(id, false, w, http.StatusInternalServerError, "Server stopped", nil)
		return
	}

	processed := true

	path := misc.NormalizeSlashes(r.URL.Path)
	if path == "" {
		path = "/"
	}

	srcPath := path

	defer func() {
		if !processed {
			path = url404
		}
		h.info.Runtime.Requests.inc()
		h.updateEndpointStat(srcPath)
		misc.LogProcessingTime("", id, "http", "", t0)
	}()

	if h.isEndpointDisabled(path) {
		Error(id, false, w, http.StatusLocked, `Endpoint "`+path+`" is disabled`, nil)
		return
	}

	if h.listenerCfg.BasicAuthEnabled {
		if !h.basicAuthHandler(id, path, w, r) {
			return
		}
	}

	path, _ = h.OldPath(srcPath)
	_, moved := h.NewPath(srcPath)

	if !moved {
		switch path {
		case "/":
			h.root(id, path, w, r)
			return

		case "/favicon.ico":
			h.icon(id, path, w, r)
			return

		case "/exit":
			h.exit(id, path, w, r)
			return

		case "/info":
			h.showInfo(id, path, w, r)
			return

		case "/config":
			h.showConfig(id, path, w, r)
			return

		case "/ping":
			tags := misc.AppTags()
			if tags != "" {
				tags = " " + tags
			}
			w.Header().Add("X-Application-Version", fmt.Sprintf("%s %s%s", misc.AppName(), misc.AppVersion(), tags))
			w.WriteHeader(http.StatusNoContent)
			return

		case "/set-log-level":
			h.changeLogLevel(id, path, w, r)
			return

		case "/profiler-enable":
			h.commonConfig.ProfilerEnabled = true
			ReturnRefresh(w, r, http.StatusNoContent)
			return

		case "/profiler-disable":
			h.commonConfig.ProfilerEnabled = false
			ReturnRefresh(w, r, http.StatusNoContent)
			return
		}

		if h.profiler(id, path, w, r) {
			return
		}
	}

	if h.handler.Handler(id, path, w, r) {
		return
	}

	if h.File(id, path, w, r) {
		return
	}

	processed = false
	Error(id, false, w, http.StatusNotFound, `Invalid endpoint "`+path+`"`, nil)

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

/*
isEndpointDisabled -- see test
*/
func (h *HTTP) isEndpointDisabled(path string) bool {
	if len(h.commonConfig.DisabledEndpoints) == 0 {
		return false
	}

	_, exists := h.commonConfig.DisabledEndpoints["*"]
	if exists {
		return true
	}

	_, exists = h.commonConfig.DisabledEndpoints[path]
	if exists {
		return true
	}

	for {
		i := strings.LastIndexByte(path, '/')
		if i < 0 {
			break
		}

		path = path[:i]
		if path == "" {
			break
		}

		_, exists = h.commonConfig.DisabledEndpoints[path+"/*"]
		if exists {
			return true
		}

	}

	return false
}

//----------------------------------------------------------------------------------------------------------------------------//

var logReplaceRequest = &misc.Replace{}

// SetLogFilterForRequest --
func SetLogFilterForRequest(f *misc.Replace) {
	logReplaceRequest = f
}

// AddLogFilterForRequest --
func AddLogFilterForRequest(exp string, replace string) error {
	return logReplaceRequest.Add(exp, replace)
}

//----------------------------------------------------------------------------------------------------------------------------//

// MovePath --
func (h *HTTP) MovePath(path string, newPath string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if path == "" {
		return
	}

	if path == newPath || newPath == "" {
		delete(h.movedPathsFwd, path)
		delete(h.movedPathsRev, path)
	} else {
		h.movedPathsFwd[path] = newPath
		h.movedPathsRev[newPath] = path
	}

	info, exists := h.info.Endpoints[path]
	if exists {
		h.info.Endpoints[newPath] = info
		delete(h.info.Endpoints, path)
	}
}

// NewPath --
func (h *HTTP) NewPath(path string) (newPath string, exists bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	newPath, exists = h.movedPathsFwd[path]
	if !exists {
		newPath = path
	}
	return
}

// OldPath --
func (h *HTTP) OldPath(newPath string) (path string, exists bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	path, exists = h.movedPathsRev[newPath]
	if !exists {
		path = newPath
	}
	return
}

//----------------------------------------------------------------------------------------------------------------------------//
