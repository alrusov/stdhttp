package stdhttp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/alrusov/jsonw"
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
	ContentTypeIcon = "ico"
	// ContentTypeForm --
	ContentTypeForm = "form"
	// ContentTypeBin --
	ContentTypeBin = "bin"

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
		ContentTypeForm: "application/x-www-form-urlencoded",
		ContentTypeBin:  "application/octet-stream",
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
func SendJSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	m, err := jsonw.Marshal(data)
	if err != nil {
		m = []byte(err.Error())
	}

	WriteReply(w, r, http.StatusOK, ContentTypeJSON, nil, m)
}

//-----------------------------------------------------------------------------s-----------------------------------------------//

// Error --
func Error(id uint64, answerSent bool, w http.ResponseWriter, r *http.Request, httpCode int, message string, err error) {
	if w != nil && !answerSent {
		type e struct {
			Message string `json:"error"`
		}
		msg := e{Message: message}
		SendJSON(w, r, httpCode, msg)
	}

	s := ""
	if err != nil {
		s = " (" + err.Error() + ")"
	}
	log.Message(log.DEBUG, `[%d] Reply: %d - "%s"%s`, id, httpCode, message, s)
}

//----------------------------------------------------------------------------------------------------------------------------//

// ReturnRefresh --
func ReturnRefresh(id uint64, w http.ResponseWriter, r *http.Request, httpCode int, forceTo string, data []byte, err error) {
	path := forceTo

	if path == "." {
		path = r.Referer()
	}

	if path == "" {
		path = r.URL.Query().Get("refresh")
	}

	if path == "" {
		if err == nil {
			w.WriteHeader(httpCode)
			if data != nil {
				w.Write(data)
			}
			return
		}

		if len(data) == 0 {
			data = []byte(err.Error())
		}

		Error(id, false, w, r, httpCode, string(data), err)
		return
	}

	p, e := url.Parse(path)
	if p != nil {
		q := p.Query()
		q.Del("___err")
		p.RawQuery = q.Encode()
	}

	if e == nil {
		if err != nil {
			q := p.Query()
			q.Set("___err", err.Error())
			p.RawQuery = q.Encode()
			Error(id, true, w, r, httpCode, string(data), err)
		}

		path = p.String()
	}

	w.Header().Set("Location", path)
	w.WriteHeader(http.StatusSeeOther)
}

//----------------------------------------------------------------------------------------------------------------------------//

// ReadData --
func ReadData(header http.Header, body io.ReadCloser) (bodyBuf *bytes.Buffer, code int, err error) {
	if body == nil {
		bodyBuf = &bytes.Buffer{}
		code = http.StatusOK
		return
	}

	if header.Get("Content-Encoding") == "gzip" {
		bodyBuf, err = misc.GzipUnpack(body)
	} else {
		bodyBuf = new(bytes.Buffer)
		_, err = bodyBuf.ReadFrom(body)
	}

	if err != nil {
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
func WriteReply(w http.ResponseWriter, r *http.Request, httpCode int, contentCode string, extraHeaders misc.StringMap, data []byte) (err error) {
	if len(data) > 0 && (extraHeaders == nil || extraHeaders["Content-Encoding"] == "") && gzipRecommended(data) {
		do := false

		if r == nil {
			do = true
		} else {
			for _, s := range r.Header["Accept-Encoding"] {
				ss := strings.Split(s, ",")
				for _, v := range ss {
					switch v {
					case "*", "gzip":
						do = true
						break
					}
				}
				if do {
					break
				}
			}
		}

		if do {
			var b *bytes.Buffer
			b, err = misc.GzipPack(bytes.NewReader(data))
			if err != nil {
				return err
			}

			data = b.Bytes()
			w.Header().Set("Content-Encoding", "gzip")
		}
	}

	if contentCode != "" {
		WriteContentHeader(w, contentCode)
	}

	if extraHeaders != nil {
		for n, v := range extraHeaders {
			w.Header().Set(n, v)
		}
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

func gzipRecommended(data []byte) bool {
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
