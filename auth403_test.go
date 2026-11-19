package stdhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alrusov/auth"
	"github.com/alrusov/config"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// testAuthMethod -- authenticates the user from the X-Test-User header, groups from X-Test-Group
type testAuthMethod struct{}

func (m *testAuthMethod) Init(lCfg *config.Listener) error { return nil }
func (m *testAuthMethod) Enabled() bool                    { return true }
func (m *testAuthMethod) Score() int                       { return 10 }
func (m *testAuthMethod) WWWAuthHeader() (string, bool)    { return "Test", false }

func (m *testAuthMethod) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (*auth.Identity, bool, error) {
	// Как keycloak после обновления токена: только заголовок, ответ не отправлен
	if c := r.Header.Get("X-Test-Set-Cookie"); c != "" {
		w.Header().Add("Set-Cookie", c)
	}

	// Как keycloak без входа: способ сам отвечает редиректом
	if loc := r.Header.Get("X-Test-Redirect"); loc != "" {
		w.Header().Set("Location", loc)
		w.WriteHeader(http.StatusFound)
		return nil, false, nil
	}

	u := r.Header.Get("X-Test-User")
	if u == "" {
		return nil, true, nil
	}

	identity := &auth.Identity{Method: "test", User: u}
	if g := r.Header.Get("X-Test-Group"); g != "" {
		identity.Groups = []string{g}
	}

	return identity, false, nil
}

// TestAuthForbiddenWithIdentity -- an authenticated user without the required group must get 403
// and must not reach the handler (earlier the 403 was dropped because the identity was not nil)
func TestAuthForbiddenWithIdentity(t *testing.T) {
	if config.GetCommon() == nil {
		config.SetCommon(&config.Common{LoadAvgPeriod: config.Duration(time.Minute)})
	}

	cfg := &config.Listener{
		Auth: config.Auth{
			Endpoints: map[string]misc.BoolMap{
				"/api*": {"@admins": true},
			},
		},
	}

	reached := false
	h, err := NewListener(cfg, &testHandler{reached: &reached})
	if err != nil {
		t.Fatal(err)
	}

	err = h.AddAuthHandler(&testAuthMethod{})
	if err != nil {
		t.Fatal(err)
	}

	type testData struct {
		user    string
		group   string
		code    int
		reached bool
		wwwAuth bool
	}

	data := []testData{
		{"", "", http.StatusUnauthorized, false, true},
		{"user", "", http.StatusForbidden, false, true},
		{"user", "users", http.StatusForbidden, false, true},
		{"admin", "admins", http.StatusOK, true, false},
	}

	for i, d := range data {
		reached = false

		r := httptest.NewRequest(http.MethodGet, "/api/objects", nil)
		if d.user != "" {
			r.Header.Set("X-Test-User", d.user)
		}
		if d.group != "" {
			r.Header.Set("X-Test-Group", d.group)
		}

		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != d.code {
			t.Errorf(`[%d] %s/%s: code %d, expected %d`, i, d.user, d.group, w.Code, d.code)
		}

		if reached != d.reached {
			t.Errorf(`[%d] %s/%s: handler reached=%v, expected %v`, i, d.user, d.group, reached, d.reached)
		}

		if (w.Header().Get("WWW-Authenticate") != "") != d.wwwAuth {
			t.Errorf(`[%d] %s/%s: WWW-Authenticate %q, expected present=%v`, i, d.user, d.group, w.Header().Get("WWW-Authenticate"), d.wwwAuth)
		}
	}
}

type testHandler struct {
	reached *bool
}

func (th *testHandler) Handler(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (processed bool) {
	*th.reached = true
	w.WriteHeader(http.StatusOK)
	return true
}

//----------------------------------------------------------------------------------------------------------------------------//

// TestAuthPassed -- доступ открыт при 0 и 2xx; 1xx, 3xx, 4xx, 5xx -- нет (раньше code%100 == 2 пропускал 102, 302, 402, 502)
func TestAuthPassed(t *testing.T) {
	for code, expected := range map[int]bool{
		0: true, 200: true, 204: true, 299: true,
		102: false, 302: false, 401: false, 402: false, 403: false, 502: false,
	} {
		if authPassed(code) != expected {
			t.Errorf("authPassed(%d) = %v, expected %v", code, !expected, expected)
		}
	}
}

// TestAuthRefusalWithHeaders -- отказ не теряется, если способ входа выставил заголовки, но ответ не отправил
// (keycloak: Set-Cookie после обновления токена); если способ ответил сам (редирект), ответ не перезаписывается
func TestAuthRefusalWithHeaders(t *testing.T) {
	if config.GetCommon() == nil {
		config.SetCommon(&config.Common{LoadAvgPeriod: config.Duration(time.Minute)})
	}

	cfg := &config.Listener{
		Auth: config.Auth{
			Endpoints: map[string]misc.BoolMap{
				"/api*": {"@admins": true},
			},
		},
	}

	reached := false
	h, err := NewListener(cfg, &testHandler{reached: &reached})
	if err != nil {
		t.Fatal(err)
	}

	if err = h.AddAuthHandler(&testAuthMethod{}); err != nil {
		t.Fatal(err)
	}

	type testData struct {
		user, group, cookie, redirect string
		code                          int
		reached                       bool
	}

	data := []testData{
		{"user", "", "kc_access=new", "", http.StatusForbidden, false}, // раньше -- пустой 200
		{"", "", "kc_access=new", "", http.StatusUnauthorized, false},  // раньше -- пустой 200
		{"admin", "admins", "kc_access=new", "", http.StatusOK, true},  // доступ открыт, cookie уходит с ответом
		{"", "", "", "https://kc/login", http.StatusFound, false},      // способ ответил сам -- не перезаписываем
		{"user", "", "", "", http.StatusForbidden, false},              // без заголовков -- как раньше
	}

	for i, d := range data {
		reached = false

		r := httptest.NewRequest(http.MethodGet, "/api/objects", nil)
		for k, v := range map[string]string{"X-Test-User": d.user, "X-Test-Group": d.group, "X-Test-Set-Cookie": d.cookie, "X-Test-Redirect": d.redirect} {
			if v != "" {
				r.Header.Set(k, v)
			}
		}

		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != d.code {
			t.Errorf(`[%d] code %d, expected %d`, i, w.Code, d.code)
		}
		if reached != d.reached {
			t.Errorf(`[%d] handler reached=%v, expected %v`, i, reached, d.reached)
		}
		if d.cookie != "" && w.Header().Get("Set-Cookie") != d.cookie {
			t.Errorf(`[%d] Set-Cookie %q, expected %q`, i, w.Header().Get("Set-Cookie"), d.cookie)
		}
		if d.redirect != "" && w.Header().Get("Location") != d.redirect {
			t.Errorf(`[%d] Location %q, expected %q`, i, w.Header().Get("Location"), d.redirect)
		}
		if d.redirect != "" && w.Body.Len() != 0 {
			t.Errorf(`[%d] response of the auth method was overwritten: body %q`, i, w.Body.String())
		}
	}
}
