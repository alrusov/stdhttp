package stdhttp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// changeLogLevel --
func (h *HTTP) changeLogLevel(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	var err error

	queryParams := r.URL.Query()
	facility := queryParams.Get("facility")
	levelName := strings.ToUpper(queryParams.Get("level"))

	status := http.StatusNoContent

	f := log.GetFacility(facility)
	if f == nil {
		status = http.StatusBadRequest
		err = fmt.Errorf(`Unknown facility "%s"`, facility)
	} else {
		f.SetLogLevel(levelName, "")
	}

	ReturnRefresh(id, w, r, status, "", nil, err)
}

//----------------------------------------------------------------------------------------------------------------------------//
