package stdhttp

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"sort"

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

type dblStrArray [][2]string

// Len implements sort.Interface.
func (d dblStrArray) Len() int {
	return len(d)
}

// Less implements sort.Interface.
func (d dblStrArray) Less(i, j int) bool {
	return d[i][0] < d[j][0]
}

// Swap implements sort.Interface.
func (d dblStrArray) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) maintenance(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	cfg := config.GetCommon()

	params := struct {
		ThisPath        string
		Copyright       string
		ErrMsg          string
		Name            string
		App             string
		Version         string
		Tags            string
		CurrentLogLevel string
		LogLevelNames   []string
		LogLevels       dblStrArray
		ProfilerEnabled bool
		Extra           []template.HTML
		LightOpen       template.HTML
		LightClose      template.HTML
	}{
		ThisPath:        r.URL.Path,
		Copyright:       misc.Copyright(),
		ErrMsg:          r.URL.Query().Get("___err"),
		Name:            cfg.Name,
		App:             misc.AppName(),
		Version:         misc.AppVersion(),
		Tags:            misc.AppTags(),
		LogLevelNames:   log.GetLogLevels(),
		ProfilerEnabled: h.commonConfig.ProfilerEnabled,
	}
	_, _, params.CurrentLogLevel = log.CurrentLogLevelEx()
	params.LightOpen, params.LightClose = h.MenuHighlight()

	if h.extraRootItemFunc != nil {
		for _, h := range h.extraRootItemFunc() {
			params.Extra = append(params.Extra, template.HTML(h))
		}
	}

	ll := dblStrArray{}
	for name, level := range log.CurrentLogLevelNamesOfAll() {
		ll = append(ll, [2]string{name, level})
	}

	sort.Sort(ll)
	params.LogLevels = ll

	status := http.StatusOK

	buf := new(bytes.Buffer)

	t, err := template.New("maintenance").Parse(rootPage)
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

	_, err = io.Copy(w, buf)
	if err != nil {
		log.Message(log.DEBUG, "[%d] %s", id, err.Error())
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
