package stdhttp

import (
	"net/http"

	"github.com/alrusov/config"
)

//----------------------------------------------------------------------------------------------------------------------------//

// showConfig --
func (h *HTTP) showConfig(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	d := []byte(config.GetSecuredText())
	WriteReply(w, r, http.StatusOK, ContentTypeText, nil, d)
}

//----------------------------------------------------------------------------------------------------------------------------//
