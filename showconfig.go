package stdhttp

import (
	"net/http"

	"github.com/alrusov/config"
)

//----------------------------------------------------------------------------------------------------------------------------//

// showConfig --
func (h *HTTP) showConfig(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	WriteContentHeader(w, ContentTypeText)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(config.GetSecuredText()))
}

//----------------------------------------------------------------------------------------------------------------------------//
