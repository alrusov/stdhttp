package auth

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/alrusov/config"
)

//----------------------------------------------------------------------------------------------------------------------------//

type testHandler struct {
	name string
	cfg  *config.AuthMethod
}

func (ah *testHandler) Init(cfg *config.Listener) (err error) {
	methodCfg, exists := cfg.Auth.Methods[ah.name]
	if !exists || !methodCfg.Enabled {
		return fmt.Errorf(`Undefined or disabled method "%s"`, ah.name)
	}

	ah.cfg = methodCfg
	return nil
}

func (ah *testHandler) Enabled() bool {
	return ah.cfg != nil && ah.cfg.Enabled
}

// Score --
func (ah *testHandler) Score() int {
	return ah.cfg.Score
}

// WWWAuthHeader --
func (ah *testHandler) WWWAuthHeader() (name string, withRealm bool) {
	return ah.name, true
}

// Check --
func (ah *testHandler) Check(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) (identity *Identity, tryNext bool) {
	return nil, false
}

//----------------------------------------------------------------------------------------------------------------------------//

func TestHandlersAdd(t *testing.T) {
	check := func(testID int, scores []int) {
		cfg := &config.Listener{
			Auth: config.Auth{
				Methods: map[string]*config.AuthMethod{},
			},
		}

		hh := NewHandlers(cfg)

		for i, score := range scores {
			name := fmt.Sprintf("test_%d", i)
			mCfg := &config.AuthMethod{Enabled: true, Score: score}
			cfg.Auth.Methods[name] = mCfg
			err := hh.Add(&testHandler{name: name})
			if err != nil {
				t.Fatalf("[%d] %s", testID, err)
			}
		}

		prev := -1
		for _, h := range hh.list {
			score := h.Score()
			if score < prev {
				t.Errorf("[%d] %d < %d", testID, score, prev)
			}
			prev = score
		}
	}

	check(1, []int{10, 20, 30, 40, 50})
	check(2, []int{50, 40, 30, 20, 10})
	check(3, []int{50, 10, 30, 20, 40})
	check(4, []int{10, 10, 10, 10, 10})
	check(5, []int{30, 10, 20, 10, 40})
}

//----------------------------------------------------------------------------------------------------------------------------//
