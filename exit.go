package stdhttp

import (
	"net/http"
	"os"
	"strconv"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// exit --
func (h *HTTP) exit(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	pid, err := strconv.ParseInt(queryParams.Get("pid"), 10, 64)
	if err != nil || pid != int64(os.Getpid()) {
		Error(id, false, w, http.StatusBadRequest, "Illegal pid", err)
		return
	}

	code := int64(0)
	s := queryParams.Get("code")
	if s != "" {
		code, err = strconv.ParseInt(s, 10, 16)
		if err != nil {
			Error(id, false, w, http.StatusBadRequest, "Illegal code", err)
			return
		}
	}

	misc.StopApp(int(code))
	w.WriteHeader(http.StatusNoContent)
}

//----------------------------------------------------------------------------------------------------------------------------//
