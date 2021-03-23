package stdhttp

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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

//----------------------------------------------------------------------------------------------------------------------------//

type unixSocketDialer struct {
	net.Dialer
}

func (d *unixSocketDialer) Dial(_ string, path string) (net.Conn, error) {
	return d.Dialer.Dial("unix",
		strings.ReplaceAll(
			strings.Split(path, ":")[0],
			".",
			"/",
		)+
			".sock",
	)
}

//----------------------------------------------------------------------------------------------------------------------------//

// Request --
func Request(method string, uri string, timeout time.Duration, opts misc.StringMap, extraHeaders misc.StringMap, data []byte) (*bytes.Buffer, *http.Response, error) {
	params := url.Values{}

	if data == nil {
		data = make([]byte, 0)
	}

	withGzip := gzipRecommended(data)
	skipTLSverification := false
	user := ""
	password := ""

	if opts != nil {
		for k, v := range opts {
			if strings.HasPrefix(k, ".") {
				switch k {
				case RequestOptionGzip:
					withGzip = withGzip && parseBoolOption(v)
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

	if withGzip {
		b, err := misc.GzipPack(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		data = b.Bytes()
	}

	req, err := http.NewRequest(method, uri, bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	if withGzip {
		req.Header.Set("Content-Encoding", "gzip")
	}

	if extraHeaders != nil {
		for n, v := range extraHeaders {
			req.Header.Set(n, v)
		}
	}

	if user != "" || password != "" {
		req.SetBasicAuth(user, password)
	}

	req.URL.RawQuery = params.Encode()

	if timeout == 0 {
		timeout = config.ClientDefaultTimeout
	}

	var tr *http.Transport
	switch req.URL.Scheme {
	case "http", "https":
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipTLSverification,
			},
		}
	case "unix":
		req.URL.Scheme = "http"
		tr = &http.Transport{
			Dial: (&unixSocketDialer{
				net.Dialer{
					Timeout:   timeout,
					KeepAlive: timeout,
				},
			}).Dial,
		}
	default:
		return nil, nil, fmt.Errorf(`Unknown scheme "%s"`, req.URL.Scheme)
	}

	clnt := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	resp, err := clnt.Do(req)
	tr.CloseIdleConnections()

	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, resp, err
	}

	bodyBuf, _, err := ReadData(resp.Header, resp.Body)
	if err != nil {
		return nil, resp, err
	}

	if resp.StatusCode/100 != 2 {
		return bodyBuf, resp, errors.New("Status code " + strconv.Itoa(resp.StatusCode))
	}

	return bodyBuf, resp, nil
}

//----------------------------------------------------------------------------------------------------------------------------//
