package stdhttp

import (
	"testing"
)

//----------------------------------------------------------------------------------------------------------------------------//

func TestNormalizeSlashes(t *testing.T) {
	type samples struct {
		in  string
		out string
	}
	smp := []samples{
		{"http://localhost", "http://localhost"},
		{"http://localhost/", "http://localhost"},
		{"http://localhost/////xxx/////yyy/zzz//", "http://localhost/xxx/yyy/zzz"},
		{"http://localhost/////xxx///https://yyy/zzz//", "http://localhost/xxx/https:/yyy/zzz"},
	}

	for i, p := range smp {
		i++
		out := NormalizeSlashes(p.in)
		if out != p.out {
			t.Errorf(`Case %d failed: in "%s", out "%s", expected: "%s"`, i, p.in, out, p.out)
		}
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
