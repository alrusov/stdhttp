package stdhttp

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// ExtraRootItemFunc --
type ExtraRootItemFunc func() []string

var (
	extraRootItemFunc ExtraRootItemFunc
)

//----------------------------------------------------------------------------------------------------------------------------//

// SetRootItemsFunc --
func SetRootItemsFunc(f ExtraRootItemFunc) {
	extraRootItemFunc = f
}

// MenuHighlight --
func MenuHighlight() (string, string) {
	return `<span style="color: red; font-weight: bold;">`, `</span>`
}

func (h *HTTP) root(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	levels := ""

	_, _, level := log.GetCurrentLogLevel()

	for _, name := range log.GetLogLevels() {
		opn, cls := "", ""
		if level == name {
			opn, cls = MenuHighlight()
		}
		levels += fmt.Sprintf(`&nbsp;<a href="/set-log-level?level=%s&amp;refresh=1">%s%s%s</a>`, url.QueryEscape(name), opn, html.EscapeString(name), cls)
	}

	prfEnabled := false
	if commonConfig != nil {
		prfEnabled = commonConfig.ProfilerEnabled
	}

	addProfilerItem := func(v bool) string {
		op := "enable"
		if !v {
			op = "disable"
		}

		opn, cls := "", ""
		if v == prfEnabled {
			opn, cls = MenuHighlight()
		}

		return fmt.Sprintf(`&nbsp;<a href="/profiler-%s?refresh=1">%s%sD%s</a>`, url.QueryEscape(op), opn, html.EscapeString(strings.ToUpper(op)), cls)
	}

	profiler := addProfilerItem(true) + addProfilerItem(false)

	extra := ""
	if extraRootItemFunc != nil {
		extra = "<li>" + strings.Join(extraRootItemFunc(), "</li><li>")
	}

	s := fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
  <head>
    <title>%s</title>
  </head>
  <body>
      <h4>%s %s</h4>
      <ul>
        <li><a href="/info" target="info">Application info in the JSON format</a></li>
		<li>Change logging level:%s</li>
		<li>Profiler is %s</li>
		%s
      </ul>
  </body>
</html>`,
		misc.AppName(), misc.AppName(), misc.AppVersion(), levels, profiler, extra)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(s))
}

//----------------------------------------------------------------------------------------------------------------------------//
