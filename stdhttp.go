package stdhttp

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
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

	// MethodCONNECT --
	MethodCONNECT = "CONNECT"
	// MethodTRACE --
	MethodTRACE = "TRACE"
	// MethodOPTIONS --
	MethodOPTIONS = "OPTIONS"
	// MethodHEAD --
	MethodHEAD = "HEAD"
	// MethodGET --
	MethodGET = "GET"
	// MethodPOST --
	MethodPOST = "POST"
	// MethodPUT --
	MethodPUT = "PUT"
	// MethodPATCH --
	MethodPATCH = "PATCH"
	// MethodDELETE --
	MethodDELETE = "DELETE"

	HTTPheaderHash = "X-Hash" // data hash
)

var (
	// Log --
	Log = log.NewFacility("stdhttp")

	StdMethods = []string{
		MethodCONNECT,
		MethodTRACE,
		MethodOPTIONS,
		MethodHEAD,
		MethodGET,
		MethodPOST,
		MethodPUT,
		MethodPATCH,
		MethodDELETE,
	}

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
		return "", fmt.Errorf(`illegal content code "%s"`, contentType)
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
func SendJSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	m, err := jsonw.Marshal(data)
	if err != nil {
		m = []byte(err.Error())
	}

	WriteReply(w, r, statusCode, ContentTypeJSON, nil, m)
}

//-----------------------------------------------------------------------------s-----------------------------------------------//

type ErrorResponse struct {
	Message string `json:"error"`
}

// Error --
func Error(id uint64, answerSent bool, w http.ResponseWriter, r *http.Request, httpCode int, message string, err error) {
	if w != nil && !answerSent {
		msg := ErrorResponse{Message: message}
		SendJSON(w, r, httpCode, msg)
	}

	s := ""
	if err != nil {
		s = " (" + err.Error() + ")"
	}
	Log.Message(log.DEBUG, `[%d] Reply: %d - "%s"%s`, id, httpCode, message, s)
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
	if UseGzip(r, len(data), extraHeaders) {
		var b *bytes.Buffer
		b, err = misc.GzipPack(bytes.NewReader(data))
		if err != nil {
			return err
		}

		data = b.Bytes()
		w.Header().Set("Content-Encoding", "gzip")
	}

	if contentCode != "" {
		WriteContentHeader(w, contentCode)
	}

	for n, v := range extraHeaders {
		w.Header().Set(n, v)
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

func UseGzip(r *http.Request, dataLen int, headers misc.StringMap) bool {
	if headers != nil && headers["Content-Encoding"] != "" {
		return false
	}

	if !gzipRecommended(dataLen) {
		return false
	}

	if r == nil {
		return true
	}

	for _, s := range r.Header["Accept-Encoding"] {
		ss := strings.Split(s, ",")
		for _, v := range ss {
			switch v {
			case "*", "gzip":
				return true
			}
		}
	}

	return false
}

//----------------------------------------------------------------------------------------------------------------------------//

var minSizeForGzip = int32(0)

// SetMinSizeForGzip --
func SetMinSizeForGzip(size int) {
	atomic.StoreInt32(&minSizeForGzip, int32(size))
}

func gzipRecommended(dataLen int) bool {
	minSize := int(atomic.LoadInt32(&minSizeForGzip))
	return minSize >= 0 && dataLen >= minSize
}

//----------------------------------------------------------------------------------------------------------------------------//

// JSONResultWithDataHash --
func JSONResultWithDataHash(data any, useHash bool, hash string, srcHeaders misc.StringMap) (result []byte, code int, headers misc.StringMap, err error) {
	headers = srcHeaders

	j, err := jsonw.Marshal(data)
	if err != nil {
		code = http.StatusInternalServerError
		return
	}

	if useHash {
		if headers == nil {
			headers = make(misc.StringMap, 1)
		}

		sha := sha512.Sum512(j)
		newHash := hex.EncodeToString(sha[:])
		headers[HTTPheaderHash] = newHash

		if newHash == hash {
			code = http.StatusNoContent
			return
		}
	}

	result = j
	code = http.StatusOK
	return
}

//----------------------------------------------------------------------------------------------------------------------------//
