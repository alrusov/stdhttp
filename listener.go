package stdhttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alrusov/auth"
	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
	"github.com/alrusov/panic"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// HTTP --
	HTTP struct {
		mutex              *sync.Mutex
		connectionID       uint64
		listenerCfg        *config.Listener
		commonConfig       *config.Common
		srv                *http.Server
		handlers           []HandlerEx
		authEndpointsKeys  misc.BoolMap
		authHandlers       *auth.Handlers
		extraFunc          ExtraInfoFunc
		statusFunc         StatusFunc
		info               *infoBlock
		extraRootItemFuncs []ExtraRootItemFunc
		removedPaths       misc.BoolMap
	}

	// Handler --
	Handler interface {
		Handler(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (processed bool)
	}

	// HandlerEx --
	HandlerEx interface {
		Handler(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (processed bool, basePath string)
	}

	// ExtraRootItemFunc --
	ExtraRootItemFunc func(prefix string) []string

	// StatusFunc --
	StatusFunc func(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request)

	// ContextKey --
	ContextKey string
)

const (
	CtxIdentity = ContextKey("identity")
)

//----------------------------------------------------------------------------------------------------------------------------//

type handlerWrapper struct {
	simple Handler
}

func (h *handlerWrapper) Handler(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (processed bool, basePath string) {
	processed = h.simple.Handler(id, prefix, path, w, r)
	return processed, path
}

//----------------------------------------------------------------------------------------------------------------------------//

// NewListener --
func NewListener(listenerCfg *config.Listener, handler Handler) (*HTTP, error) {
	return NewListenerEx(listenerCfg, &handlerWrapper{simple: handler})
}

func NewListenerEx(listenerCfg *config.Listener, handler HandlerEx) (*HTTP, error) {
	h := &HTTP{
		listenerCfg:       listenerCfg,
		commonConfig:      config.GetCommon(),
		mutex:             new(sync.Mutex),
		handlers:          []HandlerEx{handler},
		authEndpointsKeys: make(misc.BoolMap, len(listenerCfg.Auth.Endpoints)),
		authHandlers:      auth.NewHandlers(listenerCfg),
		extraFunc:         ExtraInfoFunc(nil),
		statusFunc:        StatusFunc(nil),
		info:              &infoBlock{},
		connectionID:      0,
		removedPaths:      make(misc.BoolMap),
	}

	for path := range listenerCfg.Auth.Endpoints {
		h.authEndpointsKeys[path] = true
	}

	h.srv = &http.Server{
		Addr:              listenerCfg.Addr,
		Handler:           h,
		ReadTimeout:       time.Duration(listenerCfg.Timeout),
		ReadHeaderTimeout: time.Duration(listenerCfg.Timeout),
	}

	h.initInfo()

	Log.Message(log.INFO, `Listener created on "%s"`, listenerCfg.Addr)

	return h, nil
}

//----------------------------------------------------------------------------------------------------------------------------//

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

//----------------------------------------------------------------------------------------------------------------------------//

// Stop --
func (h *HTTP) Stop() error {
	misc.StopApp(0)
	return h.Close()
}

// Close --
func (h *HTTP) Close() error {
	return h.srv.Close()
}

//----------------------------------------------------------------------------------------------------------------------------//

// Config --
func (h *HTTP) Config() *config.Listener {
	return h.listenerCfg
}

//----------------------------------------------------------------------------------------------------------------------------//

// AddHandler --
func (h *HTTP) AddHandler(handler Handler, toHead bool) {
	h.AddHandlerEx(&handlerWrapper{simple: handler}, toHead)
}

// AddHandlerEx --
func (h *HTTP) AddHandlerEx(handler HandlerEx, toHead bool) {
	if toHead {
		h.handlers = append([]HandlerEx{handler}, h.handlers...)
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

	Log.SecuredMessage(log.DEBUG, logReplaceRequest, `[%d] New %s request "%s" from %s`, id, r.Method, r.RequestURI, realIP)
	if Log.CurrentLogLevel() >= log.TRACE4 {
		body := new(bytes.Buffer)
		teeReader := io.TeeReader(r.Body, body)
		data, _, err := ReadData(r.Header, io.NopCloser(teeReader))
		if err == nil && data.Len() > 0 {
			Log.Message(log.TRACE4, `[%d] Body: %q`, id, data.Bytes())
		}
		r.Body = io.NopCloser(body)
	}

	if !misc.AppStarted() {
		Error(id, false, w, r, http.StatusInternalServerError, "Server stopped", nil)
		return
	}

	processed := true

	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	path = misc.NormalizeSlashes(path)

	prefix := ""

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
		go h.info.Runtime.Requests.inc()
		go h.updateEndpointStat(path)
		misc.LogProcessingTime(Log.Name(), "", id, "listener", "", t0)
	}()

	_, exists := isPathInList(path, h.listenerCfg.DisabledEndpoints)
	if exists {
		Error(id, false, w, r, http.StatusLocked, `Endpoint "`+path+`" is disabled`, nil)
		return
	}

	authPath, exists := isPathInList(path, h.authEndpointsKeys)
	if exists {
		identity, code, msg := h.authHandlers.Check(id, prefix, path, h.listenerCfg.Auth.Endpoints[authPath], w, r)
		if identity == nil && code != 0 {
			if len(w.Header()) == 0 {
				h.authHandlers.WriteAuthRequestHeaders(w, prefix, path)
				Error(id, false, w, r, code, msg, nil)
			}
			return
		}

		if identity != nil {
			Log.Message(log.DEBUG, `[%d] User "%s" logged in (%s)`, id, identity.User, identity.Method)
			r = AddValueToRequestContext(r, CtxIdentity, identity)
		}
	}

	if !h.IsPathReplaced(path) {
		switch path {
		case "/___.css":
			err := WriteReply(w, r, http.StatusOK, ContentTypeCSS, nil, css)
			if err != nil {
				Log.Message(log.DEBUG, "[%d] %s", id, err.Error())
			}
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

		case "/maintenance/endpoints":
			h.endpoints(id, prefix, path, w, r)
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
			Error(id, false, w, r, http.StatusNotImplemented, "Not implemented", nil)
			return

		case "/status/ping":
			tags := misc.AppTags()
			if tags != "" {
				tags = " " + tags
			}
			w.Header().Add("X-Application-Version", fmt.Sprintf("%s %s%s", misc.AppName(), misc.AppVersion(), tags))
			w.WriteHeader(http.StatusNoContent)
			return

		case "/tools/sha":
			d := auth.Hash(
				[]byte(r.URL.Query().Get("p")),
				[]byte(r.URL.Query().Get("salt")),
			)
			err := WriteReply(w, r, http.StatusOK, ContentTypeText, nil, d)
			if err != nil {
				Log.Message(log.DEBUG, "[%d] %s", id, err.Error())
			}
			return
		}

		if h.profiler(id, prefix, path, w, r) {
			return
		}
	}

	for _, handler := range h.handlers {
		var basePath string
		processed, basePath = handler.Handler(id, prefix, path, w, r)
		if processed {
			if basePath != "" {
				path = basePath
			}
			return
		}
	}

	if h.File(id, prefix, path, w, r) {
		return
	}

	processed = false
	Error(id, false, w, r, http.StatusNotFound, fmt.Sprintf(`Invalid endpoint "%s"`, path), nil)
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

// ConcatLogFilterForRequest --
func ConcatLogFilterForRequest(f *misc.Replace) {
	logReplaceRequest.Concat(*f)
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

// AddValueToRequestContext --
func AddValueToRequestContext(r *http.Request, key any, value any) (newR *http.Request) {
	ctx := context.WithValue(r.Context(), key, value)
	return r.WithContext(ctx)
}

// GetValueFromRequestContext --
func GetValueFromRequestContext(r *http.Request, key any) (value any) {
	return r.Context().Value(key)
}

func GetIdentityFromRequestContext(r *http.Request) (identity *auth.Identity, err error) {
	iface := GetValueFromRequestContext(r, CtxIdentity)
	if iface == nil {
		return
	}

	identity, ok := iface.(*auth.Identity)
	if !ok {
		err = fmt.Errorf(`value of the "%s" in context is %T, expected %T`, CtxIdentity, iface, identity)
		return
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
