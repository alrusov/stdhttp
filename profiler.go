package stdhttp

import (
	"net/http"
	"net/http/pprof"
	"strings"
)

const (
	pprofPrefix = "/debug/pprof"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) profiler(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(path, pprofPrefix) {
		return false
	}

	if !h.commonConfig.ProfilerEnabled {
		Error(id, false, w, r, http.StatusLocked, `Profiler is disabled`, nil)
		return true
	}

	r.URL.Path = path
	path = strings.Replace(path, pprofPrefix, "", 1)

	switch path {
	default:
		pprof.Index(w, r)
	case "/cmdline":
		pprof.Cmdline(w, r)

	case "/profile":
		pprof.Profile(w, r)

	case "/symbol":
		pprof.Symbol(w, r)

	case "/trace":
		pprof.Trace(w, r)
	}

	return true
}

//----------------------------------------------------------------------------------------------------------------------------//
