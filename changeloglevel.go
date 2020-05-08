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
	facility := queryParams.Get("facility")
	levelName := strings.ToUpper(queryParams.Get("level"))

	f := log.GetFacility(facility)
	if f == nil {
		Error(id, false, w, http.StatusBadRequest, `"Unknown facility "`+facility+`"`, nil)
		return
	}

	if _, err := f.SetCurrentLogLevel(levelName, ""); err != nil {
		Error(id, false, w, http.StatusBadRequest, "Illegal value provided", err)
		return
	}

	ReturnRefresh(w, r, http.StatusNoContent)
}

//----------------------------------------------------------------------------------------------------------------------------//
