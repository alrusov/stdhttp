package stdhttp

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
)

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) endpoints(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	buf := new(bytes.Buffer)
	buf.WriteString(`<h1>Endpoint templates</h1><table class="grd"><tr><th>URL</th><th>Description</th></tr>`)

	for name, info := range h.info.Endpoints {
		eName := html.EscapeString(name)
		buf.WriteString(
			fmt.Sprintf(
				`<tr><td><a href="%s">%s</a></td><td>%s</td></tr>`,
				eName,
				eName,
				html.EscapeString(info.Description),
			),
		)
	}

	buf.WriteString(`</table>`)

	WriteReply(w, r, http.StatusOK, ContentTypeHTML, nil, buf.Bytes())
}

//----------------------------------------------------------------------------------------------------------------------------//
