package stdhttp

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/alrusov/bufpool"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

const (
	// ContentTypeHTML --
	ContentTypeHTML = "html"
	// ContentTypeCSS --
	ContentTypeCSS = "css"
	// ContentTypeText --
	ContentTypeText = "text"
	// ContentTypeJSON --
	ContentTypeJSON = "json"
	// ContentTypeIcon --
	ContentTypeIcon = "icon"

	// MethodGET --
	MethodGET = "GET"
	// MethodPOST --
	MethodPOST = "POST"
	// MethodHEAD --
	MethodHEAD = "HEAD"
	// MethodPUT --
	MethodPUT = "PUT"
	// MethodDELETE --
	MethodDELETE = "DELETE"
	// MethodCONNECT --
	MethodCONNECT = "CONNECT"
	// MethodOPTIONS --
	MethodOPTIONS = "OPTIONS"
	// MethodTRACE --
	MethodTRACE = "TRACE"
	// MethodPATCH --
	MethodPATCH = "PATCH"
)

var (
	// ContentTypes --
	contentTypes = misc.StringMap{
		ContentTypeHTML: "text/html; charset=utf-8",
		ContentTypeCSS:  "text/css; charset=utf-8",
		ContentTypeText: "text/plain; charset=utf-8",
		ContentTypeJSON: "application/json; charset=utf-8",
		ContentTypeIcon: "image/x-icon",
	}
)

//----------------------------------------------------------------------------------------------------------------------------//

// ContentHeader --
func ContentHeader(contentType string) (string, error) {
	h, exists := contentTypes[contentType]
	if !exists {
		return "", fmt.Errorf(`Illegal content code "%s"`, contentType)
	}

	return h, nil
}

// WriteContentHeader --
func WriteContentHeader(w http.ResponseWriter, contentType string) error {
	h, err := ContentHeader(contentType)
	if err != nil {
		h = contentType
	}

	w.Header().Set("Content-Type", h)
	return nil
}

//----------------------------------------------------------------------------------------------------------------------------//

// SendJSON --
func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	m, err := json.Marshal(data)
	if err != nil {
		m = []byte(err.Error())
	}

	WriteContentHeader(w, ContentTypeJSON)
	w.WriteHeader(statusCode)
	w.Write(m)
}

//-----------------------------------------------------------------------------s-----------------------------------------------//

// Error --
func Error(id uint64, answerSent bool, w http.ResponseWriter, httpCode int, message string, err error) {
	if w != nil && !answerSent {
		type e struct {
			Message string `json:"error"`
		}
		msg := e{Message: message}
		SendJSON(w, httpCode, msg)
	}

	s := ""
	if err != nil {
		s = " (" + err.Error() + ")"
	}
	log.Message(log.DEBUG, `[%d] Reply: %d - "%s"%s`, id, httpCode, message, s)
}

//----------------------------------------------------------------------------------------------------------------------------//

// ReturnRefresh --
func ReturnRefresh(w http.ResponseWriter, r *http.Request, code int) {
	path := r.URL.Query().Get("refresh")
	if path != "" {
		w.Header().Set("Location", path)
		w.WriteHeader(http.StatusSeeOther)
	} else {
		w.WriteHeader(code)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// ReadData --
func ReadData(header http.Header, body io.ReadCloser) (bodyBuf *bytes.Buffer, code int, err error) {
	if body == nil {
		code = http.StatusOK
		return
	}

	if header.Get("Content-Encoding") == "gzip" {
		var b *gzip.Reader
		b, err = gzip.NewReader(body)
		if b != nil {
			defer b.Close()
		}

		if err != nil || b == nil {
			code = http.StatusBadRequest
			return
		}

		body = b
	}

	bodyBuf = bufpool.GetBuf()

	if _, err = bodyBuf.ReadFrom(body); err != nil {
		bufpool.PutBuf(bodyBuf)
		bodyBuf = nil
		code = http.StatusInternalServerError
		return
	}

	code = http.StatusOK
	return
}

//----------------------------------------------------------------------------------------------------------------------------//

// ReadRequestBody --
func ReadRequestBody(r *http.Request) (bodyBuf *bytes.Buffer, code int, err error) {
	return ReadData(r.Header, r.Body)
}

//----------------------------------------------------------------------------------------------------------------------------//

// WriteReply --
func WriteReply(w http.ResponseWriter, httpCode int, contentCode string, data []byte) (err error) {
	if gzipRecomended(data) {
		var gzbuf bytes.Buffer
		gz := gzip.NewWriter(&gzbuf)

		if _, err = gz.Write(data); err != nil {
			return err
		}
		if err = gz.Close(); err != nil {
			return err
		}
		data = gzbuf.Bytes()

		w.Header().Set("Content-Encoding", "gzip")
	}

	if contentCode != "" {
		WriteContentHeader(w, contentCode)
	}

	w.WriteHeader(httpCode)

	if len(data) > 0 {
		_, err = w.Write(data)
	}

	return err
}

//----------------------------------------------------------------------------------------------------------------------------//

// CloneURLvalues --
func CloneURLvalues(src url.Values) (dst url.Values) {
	dst = make(url.Values, len(src))

	for n, v := range src {
		v2 := make([]string, len(v))
		copy(v2, v)
		dst[n] = v2
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

var minSizeForGzip = int32(0)

// SetMinSizeForGzip --
func SetMinSizeForGzip(size int) {
	atomic.StoreInt32(&minSizeForGzip, int32(size))
}

func gzipRecomended(data []byte) bool {
	if data == nil {
		return false
	}

	ln := len(data)
	if ln == 0 {
		return false
	}

	minSize := int(atomic.LoadInt32(&minSizeForGzip))
	return minSize >= 0 && ln >= minSize
}

//----------------------------------------------------------------------------------------------------------------------------//
