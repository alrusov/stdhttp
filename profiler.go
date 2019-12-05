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

func (h *HTTP) profiler(id uint64, path string, w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, pprofPrefix) {
		return false
	}

	if !h.commonConfig.ProfilerEnabled {
		Error(id, false, w, http.StatusNotFound, `Profiler is disabled`, nil)
		return true
	}

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
