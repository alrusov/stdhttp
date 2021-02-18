package stdhttp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// changeLogLevel --
func (h *HTTP) changeLogLevel(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	var err error

	queryParams := r.URL.Query()
	facility := queryParams.Get("facility")
	levelName := strings.ToUpper(queryParams.Get("level"))

	f := log.GetFacility(facility)
	if f == nil {
		err = fmt.Errorf(`Unknown facility "%s"`, facility)
	} else {
		_, err = f.SetLogLevel(levelName, "")
	}

	status := http.StatusNoContent
	if err != nil {
		status = http.StatusBadRequest
	}

	ReturnRefresh(id, w, r, status, prefix+"/maintenance", nil, err)
}

//----------------------------------------------------------------------------------------------------------------------------//
