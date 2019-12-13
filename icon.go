package stdhttp

import (
	"io"
	"net/http"
	"os"
)

//----------------------------------------------------------------------------------------------------------------------------//

// changeLogLevel --
func (h *HTTP) icon(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	if h.listenerCfg.IconFile == "" {
		Error(id, false, w, http.StatusNotFound, `No favicon.ico file configured`, nil)
		return
	}

	fd, err := os.Open(h.listenerCfg.IconFile)
	if err != nil {
		Error(id, false, w, http.StatusNotFound, `favicon.ico file not found`, nil)
		return
	}
	defer fd.Close()

	fi, err := fd.Stat()
	if err != nil {
		Error(id, false, w, http.StatusInternalServerError, `favicon.ico read error`, nil)
	}

	sz := fi.Size()
	icon := make([]byte, sz)
	n, err := io.ReadFull(fd, icon)
	if err != nil || int64(n) != sz {
		Error(id, false, w, http.StatusInternalServerError, `favicon.ico read error`, nil)
		return
	}

	WriteContentHeader(w, "icon")
	w.WriteHeader(http.StatusOK)
	w.Write(icon)
}

//----------------------------------------------------------------------------------------------------------------------------//
