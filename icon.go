package stdhttp

import (
	"bufio"
	"io"
	"net/http"
	"os"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// changeLogLevel --
func (h *HTTP) icon(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	if h.listenerCfg.IconFile == "" {
		Error(id, false, w, http.StatusNotFound, `No favicon.ico file configured`, nil)
		return
	}

	fn, err := misc.AbsPath(h.listenerCfg.IconFile)
	if err != nil {
		Error(id, false, w, http.StatusNotFound, `favicon.ico file not found`, err)
		return
	}

	fd, err := os.Open(fn)
	if err != nil {
		Error(id, false, w, http.StatusNotFound, `favicon.ico file not found`, err)
		return
	}
	defer fd.Close()

	WriteContentHeader(w, ContentTypeIcon)
	w.WriteHeader(http.StatusOK)
	io.Copy(w, bufio.NewReader(fd))
}

//----------------------------------------------------------------------------------------------------------------------------//
