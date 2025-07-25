package stdhttp

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
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
	optsEx := make(url.Values, len(opts))
	for k, v := range opts {
		optsEx[k] = []string{v}
	}

	extraHeadersEx := make(http.Header, len(opts))
	for k, v := range extraHeaders {
		extraHeadersEx[k] = []string{v}
	}

	return RequestEx(method, uri, timeout, optsEx, extraHeadersEx, data)
}

//----------------------------------------------------------------------------------------------------------------------------//

// RequestEx --
func RequestEx(method string, uri string, timeout time.Duration, opts url.Values, extraHeaders http.Header, data []byte) (*bytes.Buffer, *http.Response, error) {
	if data == nil {
		data = make([]byte, 0)
	}

	withGzip := gzipRecommended(len(data))
	skipTLSverification := false
	user := ""
	password := ""

	for k, values := range opts {
		if strings.HasPrefix(k, ".") {
			v := values[0]
			switch k {
			case RequestOptionGzip:
				withGzip = withGzip && parseBoolOption(v)
				delete(opts, k)
			case RequestOptionSkipTLSVerification:
				skipTLSverification = parseBoolOption(v)
				delete(opts, k)
			case RequestOptionBasicAuthUser:
				user = v
				delete(opts, k)
			case RequestOptionBasicAuthPassword:
				password = v
				delete(opts, k)
			}
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
		req.Header.Set(HTTPheaderContentEncoding, ContentEncodingGzip)
	}

	for n, values := range extraHeaders {
		for _, v := range values {
			req.Header.Set(n, v)
		}
	}

	if _, exists := extraHeaders[HTTPheaderAcceptEncoding]; !exists {
		req.Header.Set(HTTPheaderAcceptEncoding, ContentEncodingGzip)
	}

	if user != "" || password != "" {
		req.SetBasicAuth(user, password)
	}

	req.URL.RawQuery = opts.Encode()

	if timeout == 0 {
		timeout = config.ClientDefaultTimeout.D()
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
		return nil, nil, fmt.Errorf(`unknown scheme "%s"`, req.URL.Scheme)
	}

	c := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	resp, err := c.Do(req)
	tr.CloseIdleConnections()

	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, resp, err
	}

	rd, err := BodyReader(resp.Header, resp.Body)
	if rd != nil {
		defer rd.Close()
	}
	if err != nil {
		return nil, resp, err
	}

	b, err := io.ReadAll(rd)
	if err != nil {
		return nil, resp, err
	}

	bb := bytes.NewBuffer(b)

	if resp.StatusCode/100 != 2 {
		return bb, resp, errors.New("Status code " + strconv.Itoa(resp.StatusCode))
	}

	return bb, resp, nil
}

//----------------------------------------------------------------------------------------------------------------------------//
