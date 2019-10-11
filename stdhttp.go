package stdhttp

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/alrusov/bufpool"
	"github.com/alrusov/log"
)

var (
	// ContentTypes --
	contentTypes = map[string]string{
		"text": "plain/text; charset=utf-8",
		"json": "application/json; charset=utf-8",
	}
)

//----------------------------------------------------------------------------------------------------------------------------//

// ContentHeader --
func ContentHeader(code string) (string, error) {
	h, exists := contentTypes[code]
	if !exists {
		return "", fmt.Errorf(`Illegal content code "%s"`, code)
	}

	return h, nil
}

// WriteContentHeader --
func WriteContentHeader(w http.ResponseWriter, code string) error {
	h, err := ContentHeader(code)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", h)
	return nil
}

//----------------------------------------------------------------------------------------------------------------------------//

// SendJSON --
func SendJSON(w http.ResponseWriter, code int, data interface{}) {
	m, err := json.Marshal(data)
	if err != nil {
		m = []byte(err.Error())
	}

	WriteContentHeader(w, "json")
	w.WriteHeader(code)
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
	refresh := r.URL.Query().Get("refresh") != ""
	if refresh {
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
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
func WriteReply(w http.ResponseWriter, httpCode int, contentCode string, data []byte, minSizeForGzip int) (err error) {
	if minSizeForGzip >= 0 && len(data) >= minSizeForGzip {
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

var (
	reSlash = regexp.MustCompile(`/{2,}`)
)

// NormalizeSlashes --
// the very bad realization...
func NormalizeSlashes(path string) string {
	p := bytes.TrimRight([]byte(path), "/")
	p = bytes.Replace(p, []byte("://"), []byte(":\\"), 1)
	p = reSlash.ReplaceAll(p, []byte("/"))
	p = bytes.Replace(p, []byte(":\\"), []byte("://"), 1)
	return string(p)
}

//----------------------------------------------------------------------------------------------------------------------------//
