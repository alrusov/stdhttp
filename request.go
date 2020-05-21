package stdhttp

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alrusov/bufpool"
	"github.com/alrusov/config"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// options with names starting with "." are used as internal parameters and are not added to query parameters
const (
	RequestOptionGzip                = ".gzip"
	RequestOptionSkipTLSVerification = ".skip-tls-verification"
	RequestOptionBasicAuthUser       = ".user"
	RequestOptionBasicAuthPassword   = ".password"
)

func parseBoolOption(opt string) bool {
	switch strings.ToLower(opt) {
	case "t", "true", "y", "yes", "1":
		return true
	}

	return false
}

// Request --
func Request(method string, uri string, timeout int, opts misc.StringMap, data []byte) (*bytes.Buffer, error) {
	buf, _, err := RequestEx(method, uri, timeout, opts, data)
	return buf, err
}

// RequestEx --
// Don't forget call bufpool.PutBuf(returned_buf)
func RequestEx(method string, uri string, timeout int, opts misc.StringMap, data []byte) (*bytes.Buffer, *http.Response, error) {
	params := url.Values{}

	if data == nil {
		data = make([]byte, 0)
	}

	withGzip := gzipRecomended(data)
	skipTLSverification := false
	user := ""
	password := ""

	if opts != nil {
		for k, v := range opts {
			if strings.HasPrefix(k, ".") {
				switch k {
				case RequestOptionGzip:
					withGzip = parseBoolOption(v)
				case RequestOptionSkipTLSVerification:
					skipTLSverification = parseBoolOption(v)
				case RequestOptionBasicAuthUser:
					user = v
				case RequestOptionBasicAuthPassword:
					password = v
				}
				continue
			}
			params.Set(k, v)
		}
	}

	preparedData := data
	if withGzip {
		buf := bufpool.GetBuf()
		defer bufpool.PutBuf(buf)
		gz := gzip.NewWriter(buf)

		if _, err := gz.Write(data); err != nil {
			return nil, nil, err
		}
		if err := gz.Close(); err != nil {
			return nil, nil, err
		}
		preparedData = buf.Bytes()
	}

	req, err := http.NewRequest(method, uri, bytes.NewReader(preparedData))
	if err != nil {
		return nil, nil, err
	}

	if user != "" || password != "" {
		req.SetBasicAuth(user, password)
	}

	if withGzip {
		req.Header.Add("Content-Encoding", "gzip")
	}

	req.URL.RawQuery = params.Encode()

	if timeout == 0 {
		timeout = config.ClientDefaultTimeout
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSverification,
		},
	}

	clnt := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: tr,
	}

	resp, err := clnt.Do(req)
	tr.CloseIdleConnections()

	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, nil, err
	}

	bodyBuf, _, err := ReadData(resp.Header, resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode/100 != 2 {
		return bodyBuf, resp, errors.New("Status code " + strconv.Itoa(resp.StatusCode))
	}

	return bodyBuf, resp, nil
}

//----------------------------------------------------------------------------------------------------------------------------//
