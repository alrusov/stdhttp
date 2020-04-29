package stdhttp

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// SetRootItemsFunc --
func (h *HTTP) SetRootItemsFunc(f ExtraRootItemFunc) {
	h.extraRootItemFunc = f
}

// MenuHighlight --
func (h *HTTP) MenuHighlight() (string, string) {
	return `<span style="color: red; font-weight: bold;">`, `</span>`
}

func (h *HTTP) root(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	levels := ""

	_, _, level := log.GetCurrentLogLevel()

	for _, name := range log.GetLogLevels() {
		opn, cls := "", ""
		if level == name {
			opn, cls = h.MenuHighlight()
		}
		levels += fmt.Sprintf(`&nbsp;<a href="/set-log-level?level=%s&amp;refresh=/">%s%s%s</a>`, url.QueryEscape(name), opn, html.EscapeString(name), cls)
	}

	profilerEnabled := h.commonConfig.ProfilerEnabled

	addProfilerItem := func(v bool) string {
		op := "enable"
		if !v {
			op = "disable"
		}

		opn, cls := "", ""
		if v == profilerEnabled {
			opn, cls = h.MenuHighlight()
		}

		return fmt.Sprintf(`&nbsp;<a href="/profiler-%s?refresh=/">%s%sD%s</a>`, url.QueryEscape(op), opn, html.EscapeString(strings.ToUpper(op)), cls)
	}

	profilerSwitch := addProfilerItem(true) + addProfilerItem(false)

	profiler := ""
	if profilerEnabled {
		profiler = `<li><a href="debug/pprof/" target="pprof">Show profiler</a></li>`
	}

	extra := ""
	if h.extraRootItemFunc != nil {
		extra = "<li>" + strings.Join(h.extraRootItemFunc(), "</li><li>")
	}

	cfg := config.GetCommon()

	s := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
	<head>
		<title>%s</title>
	</head>
	<body>
		<h4>%s [<em>%s %s%s</em>]</h4>
		<ul>
			<li><a href="/info" target="info">Application info in the JSON format</a></li>
			<li><a href="/config" target="config">Prepared config</a></li>
			<li>Change logging level:%s</li>
			<li>Profiler is %s</li>
			%s
			%s
		</ul>
	</body>
</html>
`,
		cfg.Name, cfg.Name, misc.AppName(), misc.AppVersion(), misc.AppTags(true), levels, profilerSwitch, profiler, extra)

	WriteContentHeader(w, ContentTypeHTML)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(s))
}

//----------------------------------------------------------------------------------------------------------------------------//
