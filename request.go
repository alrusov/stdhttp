package stdhttp

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alrusov/config"
)

//----------------------------------------------------------------------------------------------------------------------------//

// Request --
// Don't forget call bufpool.PutBuf(returned_buf)
func Request(method string, uri string, timeout int, opts map[string]string, data []byte) (*bytes.Buffer, error) {

	params := url.Values{}

	if data == nil {
		data = make([]byte, 0)
	}

	if opts != nil {
		for k, v := range opts {
			params.Set(k, v)
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = params.Encode()

	if timeout == 0 {
		timeout = config.ClientDefaultTimeout
	}

	tr := &http.Transport{}

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
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, errors.New("Status code " + strconv.Itoa(resp.StatusCode))
	}

	bodyBuf, _, err := ReadData(resp.Header, resp.Body)
	if err != nil {
		return nil, err
	}

	return bodyBuf, nil
}

//----------------------------------------------------------------------------------------------------------------------------//
