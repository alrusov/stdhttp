package stdhttp

import (
	"bytes"
	"html/template"
	"net/http"
	"sort"

	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) endpoints(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	params := struct {
		Prefix string
		Name   string
		ErrMsg string
		List   dblStrArray
	}{
		Prefix: prefix,
		Name:   "Known endpoints",
		ErrMsg: r.URL.Query().Get("___err"),
		List:   make(dblStrArray, 0, len(h.info.Endpoints)),
	}

	h.mutex.Lock()
	for name, info := range h.info.Endpoints {
		params.List = append(params.List, [2]string{name, info.Description})
	}
	h.mutex.Unlock()
	sort.Sort(params.List)

	t, err := template.New("endpoints").Parse(endpointsPage)
	if err != nil {
		Error(id, false, w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	buf := new(bytes.Buffer)

	err = t.Execute(buf, params)
	if err != nil {
		Error(id, false, w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	err = WriteReply(w, r, http.StatusOK, ContentTypeHTML, nil, buf.Bytes())
	if err != nil {
		Log.Message(log.DEBUG, "[%d] %s", id, err.Error())
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
