package stdhttp

import (
	"net/http"
	"strings"

	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// changeLogLevel --
func (h *HTTP) changeLogLevel(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	levelName := strings.ToUpper(queryParams.Get("level"))

	if _, err := log.SetCurrentLogLevel(levelName, ""); err != nil {
		Error(id, false, w, http.StatusBadRequest, "Illegal value provided", err)
		return
	}

	ReturnRefresh(w, r, http.StatusNoContent)
}

//----------------------------------------------------------------------------------------------------------------------------//
