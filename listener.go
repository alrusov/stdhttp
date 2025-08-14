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
		sync.Mutex
		connectionID       uint64
		listenerCfg        *config.Listener
		commonConfig       *config.Common
		srv                *http.Server
		handlers           []HandlerEx
		authEndpointsKeys  misc.BoolMap
		authHandlers       *auth.Handlers
		extraFunc          ExtraInfoFunc
		statusFunc         StatusFunc
		info               *InfoBlock
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
		handlers:          []HandlerEx{handler},
		authEndpointsKeys: make(misc.BoolMap, len(listenerCfg.Auth.Endpoints)),
		authHandlers:      auth.NewHandlers(listenerCfg),
		extraFunc:         ExtraInfoFunc(nil),
		statusFunc:        StatusFunc(nil),
		info:              &InfoBlock{},
		connectionID:      0,
		removedPaths:      make(misc.BoolMap),
	}

	for path := range listenerCfg.Auth.Endpoints {
		h.authEndpointsKeys[path] = true
	}

	addr := listenerCfg.Addr
	if misc.IsDebug() && listenerCfg.DebugAddr != "" {
		addr = listenerCfg.DebugAddr
	}

	h.srv = &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadTimeout:       0,
		ReadHeaderTimeout: listenerCfg.Timeout.D(),
	}

	h.initInfo()

	Log.Message(log.INFO, `Listener created on "%s"`, addr)

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
	h.Lock()
	defer h.Unlock()

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

	var err error
	r.Body, err = BodyReader(r.Header, r.Body)
	if err != nil {
		Error(id, false, w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if Log.CurrentLogLevel() >= log.TRACE3 {
		Log.Message(log.TRACE3, `[%d] Header: %v`, id, r.Header)
		if Log.CurrentLogLevel() >= log.TRACE4 {
			bb, _ := io.ReadAll(r.Body)
			Log.Message(log.TRACE4, `[%d] Body: %q`, id, bb)
			r.Body = io.NopCloser(bytes.NewBuffer(bb))
		}
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

	prefix, path := h.GetPrefix(path, r)

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
	if exists && len(h.listenerCfg.Auth.Endpoints[authPath]) != 0 {
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
		if h.Embedded(id, prefix, path, w, r) {
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

	iter := 0

	for {
		iter++

		if iter > 1 {
			_, exists = list[path+"/*"]
			if exists {
				pattern = path + "/*"
				return
			}
		}

		_, exists = list[path+"*"]
		if exists {
			pattern = path + "*"
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
	h.Lock()
	defer h.Unlock()

	h.removedPaths[path] = true
}

// CancelPathReplacing --
func (h *HTTP) CancelPathReplacing(path string) {
	h.Lock()
	defer h.Unlock()

	delete(h.removedPaths, path)
}

// IsPathReplaced --
func (h *HTTP) IsPathReplaced(path string) bool {
	h.Lock()
	defer h.Unlock()

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

func (h *HTTP) GetPrefix(path string, r *http.Request) (prefix string, newPath string) {
	proxyPrefix := misc.NormalizeSlashes(h.GetPrefixFromHeader(r) + h.listenerCfg.ProxyPrefix)

	if proxyPrefix != "" && (path == proxyPrefix || strings.HasPrefix(path, proxyPrefix+"/")) {
		prefix = proxyPrefix
		newPath = path[len(proxyPrefix):]
		return
	}

	newPath = path
	return
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) GetPrefixFromHeader(r *http.Request) (prefix string) {
	return r.Header.Get("X-Proxy-Prefix")
}

//----------------------------------------------------------------------------------------------------------------------------//
