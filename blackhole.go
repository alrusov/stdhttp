package stdhttp

import (
	"net/http"
)

//----------------------------------------------------------------------------------------------------------------------------//

// BlackHole -- http.ResponseWriter implementation
type BlackHole struct {
}

// Header --
func (bh *BlackHole) Header() http.Header {
	return make(http.Header)
}

// Write --
func (bh *BlackHole) Write(data []byte) (int, error) {
	return len(data), nil
}

// WriteHeader --
func (bh *BlackHole) WriteHeader(statusCode int) {
}

//----------------------------------------------------------------------------------------------------------------------------//
