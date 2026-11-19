package stdhttp

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/alrusov/config"
)

//----------------------------------------------------------------------------------------------------------------------------//

func TestSecuredHeader(t *testing.T) {
	src := http.Header{
		"Authorization":       {"Bearer eyJhbGciOi.secret-jwt"},
		"Proxy-Authorization": {"Basic c2VjcmV0"},
		"Cookie":              {"session=secret-cookie"},
		"Set-Cookie":          {"a=secret-set-cookie"},
		"X-Api-Key":           {"secret-api-key"},
		"X-Auth-Token":        {"secret-token"},
		"Referer":             {"https://h/app?id=1&token=secret-ref"},
		"Content-Type":        {"application/json"},
		"X-Kapi-Request-Id":   {"12345"},
	}

	got := SecuredHeader(src)
	text := fmt.Sprintf("%v", got)

	for _, v := range []string{"eyJ", "c2VjcmV0", "secret-cookie", "secret-set-cookie", "secret-api-key", "secret-token", "secret-ref"} {
		if strings.Contains(text, v) {
			t.Errorf("%q is not masked: %s", v, text)
		}
	}

	for name, expected := range map[string]string{
		"Authorization":     "*",
		"Content-Type":      "application/json",
		"X-Kapi-Request-Id": "12345",
		"Referer":           "https://h/app?id=1&token=*",
	} {
		if got.Get(name) != expected {
			t.Errorf("%s: got %q, expected %q", name, got.Get(name), expected)
		}
	}

	// Исходные заголовки не меняются -- запрос обрабатывается дальше с настоящими значениями
	if src.Get("Authorization") != "Bearer eyJhbGciOi.secret-jwt" {
		t.Errorf("source header modified")
	}
}

// Тело запроса в журнале TRACE4 -- через config.SecureText (тот же маскировщик, что у дампа конфига)
func TestSecuredBody(t *testing.T) {
	body := `[{"name":"line","parameters":{"password":"secret-p1","host":"h"}}]`
	got := fmt.Sprintf("%q", config.SecureText(body))

	if strings.Contains(got, "secret-p1") || !strings.Contains(got, `\"host\":\"h\"`) {
		t.Errorf("got %s", got)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
