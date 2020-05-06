package stdhttp

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// File --
func (h *HTTP) File(id uint64, path string, w http.ResponseWriter, r *http.Request) (processed bool) {
	processed = false

	if h.listenerCfg.Root == "" {
		return
	}

	fn, err := misc.AbsPath(h.listenerCfg.Root + "/" + path)
	if err != nil {
		return
	}

	if !strings.HasPrefix(fn, h.listenerCfg.Root) {
		processed = true
		Error(id, false, w, http.StatusBadRequest, "Bad request", fmt.Errorf("Hackers path: %s", path))
		return
	}

	_, err = os.Stat(fn)
	if err != nil {
		return
	}

	processed = true

	defer func() {
		if err != nil {
			Error(id, false, w, http.StatusInternalServerError, "Server error", err)
		}
	}()

	fd, err := os.Open(fn)
	if err != nil {
		return
	}
	defer fd.Close()

	WriteContentHeader(w, strings.TrimLeft(filepath.Ext(fn), "."))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, bufio.NewReader(fd))

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
