package stdhttp

import (
	"net/http"

	"github.com/alrusov/config"
	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// showConfig --
func (h *HTTP) showConfig(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	d := []byte(config.GetSecuredText())
	err := WriteReply(w, r, http.StatusOK, ContentTypeText, nil, d)
	if err != nil {
		Log.Message(log.DEBUG, "[%d] %s", id, err.Error())
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
