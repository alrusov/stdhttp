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
	"github.com/alrusov/stdhttp/auth"
	"github.com/alrusov/stdhttp/auth/basic"
	"github.com/alrusov/stdhttp/auth/jwt"
	"github.com/alrusov/stdhttp/auth/krb5"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// HTTP --
	HTTP struct {
		mutex             *sync.Mutex
		connectionID      uint64
		listenerCfg       *config.Listener
		commonConfig      *config.Common
		srv               *http.Server
		handlers          []Handler
		authEnpointsKeys  misc.BoolMap
		authHandlers      *auth.Handlers
		extraFunc         ExtraInfoFunc
		statusFunc        StatusFunc
		info              *infoBlock
		extraRootItemFunc ExtraRootItemFunc
		removedPaths      misc.BoolMap
	}

	// Handler --
	Handler interface {
		Handler(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) bool
	}

	// ExtraRootItemFunc --
	ExtraRootItemFunc func(prefix string) []string

	// StatusFunc --
	StatusFunc func(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request)
)

//----------------------------------------------------------------------------------------------------------------------------//

// NewListener --
func NewListener(listenerCfg *config.Listener, handler Handler) (*HTTP, error) {
	h := &HTTP{
		listenerCfg:      listenerCfg,
		commonConfig:     config.GetCommon(),
		mutex:            new(sync.Mutex),
		handlers:         []Handler{handler},
		authEnpointsKeys: make(misc.BoolMap, len(listenerCfg.Auth.Endpoints)),
		authHandlers:     auth.NewHandlers(listenerCfg),
		extraFunc:        ExtraInfoFunc(nil),
		statusFunc:       StatusFunc(nil),
		info:             &infoBlock{},
		connectionID:     0,
		removedPaths:     make(misc.BoolMap),
	}

	for path := range listenerCfg.Auth.Endpoints {
		h.authEnpointsKeys[path] = true
	}

	stdAuthHandlers := []auth.Handler{
		&basic.AuthHandler{},
		&jwt.AuthHandler{},
		&krb5.AuthHandler{},
	}

	for _, ah := range stdAuthHandlers {
		err := h.authHandlers.Add(ah)
		if err != nil {
			return nil, err
		}
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

// AddHandler --
func (h *HTTP) AddHandler(handler Handler, toHead bool) {
	if toHead {
		h.handlers = append([]Handler{handler}, h.handlers...)
		return
	}

	h.handlers = append(h.handlers, handler)
}

//----------------------------------------------------------------------------------------------------------------------------//

// AddAuthHandler --
func (h *HTTP) AddAuthHandler(ah auth.Handler) (err error) {
	return h.authHandlers.Add(ah)
}

//----------------------------------------------------------------------------------------------------------------------------//

// AddAuthEndpoint --
func (h *HTTP) AddAuthEndpoint(endpoint string, permissions misc.BoolMap) {
	h.listenerCfg.Auth.Endpoints[endpoint] = permissions
}

//----------------------------------------------------------------------------------------------------------------------------//

// SetStatusFunc --
func (h *HTTP) SetStatusFunc(f StatusFunc, paramsInfo string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.statusFunc = f

	if paramsInfo != "" {
		name := "/status"
		h.info.Endpoints[name].Description =
			fmt.Sprintf(
				"%s: %s",
				strings.SplitN(h.info.Endpoints[name].Description, ":", 2)[0],
				paramsInfo,
			)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// ServeHTTP --
func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t0 := misc.NowUnixNano()

	panicID := panic.ID()
	defer panic.SaveStackToLogEx(panicID)

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

	prefix := ""
	path := misc.NormalizeSlashes(r.URL.Path)

	if path == h.listenerCfg.ProxyPrefix || strings.HasPrefix(path, h.listenerCfg.ProxyPrefix+"/") {
		prefix = h.listenerCfg.ProxyPrefix
		path = path[len(h.listenerCfg.ProxyPrefix):]
	}

	if path == "" {
		path = "/"
	}

	defer func() {
		if !processed {
			path = url404
		}
		h.info.Runtime.Requests.inc()
		h.updateEndpointStat(path)
		misc.LogProcessingTime("", "", id, "http", "", t0)
	}()

	_, exists := isPathInList(path, h.listenerCfg.DisabledEndpoints)
	if exists {
		Error(id, false, w, http.StatusLocked, `Endpoint "`+path+`" is disabled`, nil)
		return
	}

	authPath, exists := isPathInList(path, h.authEnpointsKeys)
	if exists {
		identity, code, msg := h.authHandlers.Check(id, prefix, path, h.listenerCfg.Auth.Endpoints[authPath], w, r)
		if identity == nil && code != 0 {
			if len(w.Header()) == 0 {
				h.authHandlers.WriteAuthRequestHeaders(w, prefix, path)
				Error(id, false, w, code, msg, nil)
			}
			return
		}

		if identity != nil {
			log.Message(log.DEBUG, `[%d] User "%s" logged in (%s)`, id, identity.User, identity.Method)
		}
	}

	if !h.IsPathReplaced(path) {
		switch path {
		case "/___.css":
			WriteContentHeader(w, ContentTypeCSS)
			w.WriteHeader(http.StatusOK)
			w.Write(css)
			return

		case "/":
			w.Header().Add("Location", prefix+"/maintenance")
			w.WriteHeader(http.StatusPermanentRedirect)
			return

		case "/debug/build-info":
			h.debugBuildInfo(id, prefix, path, w, r)
			return

		case "/debug/env":
			h.debugEnv(id, prefix, path, w, r)
			return

		case "/debug/free-os-memory":
			h.debugFreeOSmem(id, prefix, path, w, r)
			return

		case "/debug/gc-stat":
			h.debugGCstat(id, prefix, path, w, r)
			return

		case "/debug/mem-stat":
			h.debugMemStat(id, prefix, path, w, r)
			return

		case "/favicon.ico":
			h.icon(id, prefix, path, w, r)
			return

		case "/maintenance":
			h.maintenance(id, prefix, path, w, r)
			return

		case "/maintenance/config":
			h.showConfig(id, prefix, path, w, r)
			return

		case "/maintenance/exit":
			h.exit(id, prefix, path, w, r)
			return

		case "/maintenance/info":
			h.showInfo(id, prefix, path, w, r)
			return

		case "/maintenance/profiler-disable":
			h.commonConfig.ProfilerEnabled = false
			ReturnRefresh(id, w, r, http.StatusNoContent, ".", nil, nil)
			return

		case "/maintenance/profiler-enable":
			h.commonConfig.ProfilerEnabled = true
			ReturnRefresh(id, w, r, http.StatusNoContent, ".", nil, nil)
			return

		case "/maintenance/set-log-level":
			h.changeLogLevel(id, prefix, path, w, r)
			return

		case "/status":
			if h.statusFunc != nil {
				h.statusFunc(id, prefix, path, w, r)
				return
			}
			Error(id, false, w, http.StatusNotImplemented, "Not implemented", nil)
			return

		case "/status/ping":
			tags := misc.AppTags()
			if tags != "" {
				tags = " " + tags
			}
			w.Header().Add("X-Application-Version", fmt.Sprintf("%s %s%s", misc.AppName(), misc.AppVersion(), tags))
			w.WriteHeader(http.StatusNoContent)
			return

		case "/tools/jwt-login":
			jwt.GetToken(h.listenerCfg, id, path, w, r)
			return

		case "/tools/sha":
			WriteContentHeader(w, ContentTypeText)
			w.WriteHeader(http.StatusOK)
			w.Write(misc.Sha512Hash([]byte(r.URL.Query().Get("p"))))
			return
		}

		if h.profiler(id, prefix, path, w, r) {
			return
		}
	}

	for _, handler := range h.handlers {
		if handler.Handler(id, prefix, path, w, r) {
			return
		}
	}

	if h.File(id, prefix, path, w, r) {
		return
	}

	processed = false
	Error(id, false, w, http.StatusNotFound, `Invalid endpoint "`+path+`"`, nil)

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

func isPathInList(path string, list misc.BoolMap) (pattern string, exists bool) {
	if len(list) == 0 {
		return
	}

	_, exists = list[path]
	if exists {
		pattern = path
		return
	}

	_, exists = list["!"+path]
	if exists {
		exists = false
		return
	}

	iter := 0

	for {
		iter++

		if iter > 1 {
			_, exists = list[path+"/*"]
			if exists {
				pattern = path + "/*"
				return
			}

			_, exists = list["!"+path+"/*"]
			if exists {
				exists = false
				return
			}
		}

		_, exists = list[path+"*"]
		if exists {
			pattern = path + "*"
			return
		}

		_, exists = list["!"+path+"*"]
		if exists {
			exists = false
			return
		}

		i := strings.LastIndexByte(path, '/')
		if i < 0 {
			break
		}

		path = path[:i]
		if path == "" {
			break
		}
	}

	_, exists = list["*"]
	if exists {
		pattern = "*"
		return
	}

	return
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

// RemoveStdPath --
func (h *HTTP) RemoveStdPath(path string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.removedPaths[path] = true
}

// CancelPathReplacing --
func (h *HTTP) CancelPathReplacing(path string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.removedPaths, path)
}

// IsPathReplaced --
func (h *HTTP) IsPathReplaced(path string) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	_, exists := h.removedPaths[path]
	return exists
}

//----------------------------------------------------------------------------------------------------------------------------//
