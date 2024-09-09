package stdhttp

import (
	"fmt"
	"net/http"

	"github.com/alrusov/auth"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) Embedded(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (processed bool) {
	processed = true

	switch path {
	default:
		processed = false
		return

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
}

//----------------------------------------------------------------------------------------------------------------------------//
