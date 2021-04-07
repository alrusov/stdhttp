package stdhttp

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// File --
func (h *HTTP) File(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (processed bool) {
	processed = false

	if h.listenerCfg.Root == "" {
		return
	}

	fn, err := misc.AbsPath(h.listenerCfg.Root + "/" + path)
	if err != nil {
		processed = true
		Error(id, false, w, r, http.StatusBadRequest, "Bad request", err)
		return
	}

	if !strings.HasPrefix(fn, h.listenerCfg.Root) {
		processed = true
		Error(id, false, w, r, http.StatusBadRequest, "Bad request", fmt.Errorf("Hackers path: %s", path))
		return
	}

	_, err = os.Stat(fn)
	if err != nil {
		// 404
		return
	}

	fd, err := os.Open(fn)
	if err != nil {
		processed = true
		Error(id, false, w, r, http.StatusInternalServerError, "Server error", err)
		return
	}
	defer fd.Close()

	WriteContentHeader(w, strings.TrimLeft(filepath.Ext(fn), "."))
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, bufio.NewReader(fd))
	if err != nil {
		Log.Message(log.DEBUG, "[%d] %s", id, err.Error())
	}

	processed = true
	return
}

//----------------------------------------------------------------------------------------------------------------------------//
