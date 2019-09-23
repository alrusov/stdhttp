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

func root(w http.ResponseWriter) {
	levels := ""

	_, _, level := log.GetCurrentLogLevel()

	for _, name := range log.GetLogLevels() {
		opn, cls := "", ""
		if level == name {
			opn, cls = MenuHighlight()
		}
		levels += fmt.Sprintf(`&nbsp;<a href="/set-log-level?level=%s&amp;refresh=1">%s%s%s</a>`, url.QueryEscape(name), opn, html.EscapeString(name), cls)
	}

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
		%s
      </ul>
  </body>
</html>`,
		misc.AppName(), misc.AppName(), misc.AppVersion(), levels, extra)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(s))
}

//----------------------------------------------------------------------------------------------------------------------------//
