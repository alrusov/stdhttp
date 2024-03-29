package stdhttp

import (
	"bytes"
	"html/template"
	"net/http"
	"sort"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// SetRootItemsFunc --
func (h *HTTP) SetRootItemsFunc(f ExtraRootItemFunc) {
	h.extraRootItemFuncs = append(h.extraRootItemFuncs, f)
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

func (h *HTTP) maintenance(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	cfg := config.GetCommon()

	params := struct {
		Prefix          string
		HeaderPrefix    string
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
		Prefix:          prefix,
		HeaderPrefix:    h.GetPrefixFromHeader(r),
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

	for _, f := range h.extraRootItemFuncs {
		for _, t := range f(prefix) {
			params.Extra = append(params.Extra, template.HTML(t))
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
		Log.Message(log.ERR, `[%d] %s`, id, err.Error())
	} else {
		err = t.Execute(buf, params)
		if err != nil {
			status = http.StatusInternalServerError
			buf.WriteString(err.Error())
			Log.Message(log.ERR, `[%d] %s`, id, err.Error())
		}
	}

	err = WriteReply(w, r, status, ContentTypeHTML, nil, buf.Bytes())
	if err != nil {
		Log.Message(log.DEBUG, "[%d] %s", id, err.Error())
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
