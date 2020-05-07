package stdhttp

import (
	"html/template"
	"io"
	"net/http"

	"github.com/alrusov/bufpool"

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
func (h *HTTP) MenuHighlight() (open template.HTML, close template.HTML) {
	return `<span style="color: red; font-weight: bold;">`, `</span>`
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) root(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	cfg := config.GetCommon()

	tags := misc.AppTags()
	if tags != "" {
		tags = " " + tags
	}

	page := `` +
		`<!DOCTYPE html>` +
		`<html lang="en">` +
		/*	*/ `<head>` +
		/*		*/ `<title>{{.Name}}</title>` +
		/*	*/ `</head>` +
		/*	*/ `<body>` +
		/*		*/ `<h4>{{.Name}} <em>[{{.AppName}} {{.AppVersion}}{{if .AppTags}}&nbsp;{{.AppTags}}{{end}}]</em></h4>` +
		/*		*/ `<ul>` +
		/*			*/ `<li><a href="/info" target="info">Application info in the JSON format</a></li>` +
		/*			*/ `<li><a href="/config" target="config">Prepared config</a></li>` +
		/*			*/ `<li>Change logging level:` +
		/*				*/ `{{range $_, $Level := .LogLevels}}` +
		/*					*/ `&nbsp;<a href="/set-log-level?level={{$Level}}&amp;refresh={{$.ThisPath}}">` +
		/*						*/ `{{if eq $Level $.CurrentLogLevel}}{{$.LightOpen}}{{end}}` +
		/*						*/ `{{$Level}}` +
		/*						*/ `{{if eq $Level $.CurrentLogLevel}}{{$.LightClose}}{{end}}` +
		/*					*/ `</a>` +
		/*				*/ `{{end}}` +
		/*			*/ `</li>` +
		/*			*/ `<li>Profiler is ` +
		/*				*/ `&nbsp;<a href="/profiler-enable?refresh={{$.ThisPath}}">` +
		/*					*/ `{{if .ProfilerEnabled}}{{$.LightOpen}}{{end}}` +
		/*					*/ `ENABLED` +
		/*					*/ `{{if .ProfilerEnabled}}{{$.LightClose}}{{end}}` +
		/*				*/ `</a>` +
		/*				*/ `&nbsp;<a href="/profiler-disable?refresh={{$.ThisPath}}">` +
		/*					*/ `{{if not .ProfilerEnabled}}{{$.LightOpen}}{{end}}` +
		/*					*/ `DISABLED` +
		/*					*/ `{{if not .ProfilerEnabled}}{{$.LightClose}}{{end}}` +
		/*				*/ `</a>` +
		/*			*/ `</li>` +
		/*			*/ `{{if .ProfilerEnabled}}` +
		/*				*/ `<li><a href="debug/pprof/" target="pprof">Show profiler</a></li>` +
		/*			*/ `{{end}}` +
		/*			*/ `{{range .Extra}}` +
		/*				*/ `<li>{{.}}</li>` +
		/*			*/ `{{end}}` +
		/*		*/ `</ul>` +
		/*	*/ `</body>` +
		`</html>`

	params := struct {
		ThisPath        string
		Name            string
		AppName         string
		AppVersion      string
		AppTags         string
		CurrentLogLevel string
		LogLevels       []string
		ProfilerEnabled bool
		Extra           []template.HTML
		LightOpen       template.HTML
		LightClose      template.HTML
	}{
		Name:            cfg.Name,
		AppName:         misc.AppName(),
		AppVersion:      misc.AppVersion(),
		AppTags:         misc.AppTags(),
		LogLevels:       log.GetLogLevels(),
		ProfilerEnabled: h.commonConfig.ProfilerEnabled,
	}

	params.ThisPath, _ = h.NewPath("/")
	if params.ThisPath == "" {
		params.ThisPath = "/"
	}

	_, _, params.CurrentLogLevel = log.GetCurrentLogLevel()
	params.LightOpen, params.LightClose = h.MenuHighlight()

	if h.extraRootItemFunc != nil {
		for _, h := range h.extraRootItemFunc() {
			params.Extra = append(params.Extra, template.HTML(h))
		}
	}

	status := http.StatusOK

	buf := bufpool.GetBuf()
	defer bufpool.PutBuf(buf)

	t, err := template.New("root").Parse(page)
	if err != nil {
		status = http.StatusInternalServerError
		buf.WriteString(err.Error())
		log.Message(log.ERR, `[%d] %s`, id, err.Error())
	} else {
		err = t.Execute(buf, params)
		if err != nil {
			status = http.StatusInternalServerError
			buf.WriteString(err.Error())
			log.Message(log.ERR, `[%d] %s`, id, err.Error())
		}
	}

	WriteContentHeader(w, ContentTypeHTML)
	w.WriteHeader(status)
	io.Copy(w, buf)
}

//----------------------------------------------------------------------------------------------------------------------------//
