package stdhttp

import (
	"html/template"
	"io"
	"net/http"
	"sort"

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

	tags := misc.AppTags()
	if tags != "" {
		tags = " " + tags
	}

	params := struct {
		ThisPath        string
		Copyright       string
		Name            string
		AppName         string
		AppVersion      string
		AppTags         string
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
		Name:            cfg.Name,
		AppName:         misc.AppName(),
		AppVersion:      misc.AppVersion(),
		AppTags:         misc.AppTags(),
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

	buf := bufpool.GetBuf()
	defer bufpool.PutBuf(buf)

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
	io.Copy(w, buf)
}

//----------------------------------------------------------------------------------------------------------------------------//
