package stdhttp

import (
	"testing"
)

//----------------------------------------------------------------------------------------------------------------------------//

func TestIsEndpointDisabled(t *testing.T) {
	type testData struct {
		config   map[string]bool
		path     string
		disabled bool
	}

	data := []testData{
		{map[string]bool{}, "", false},
		{map[string]bool{}, "/aaa/bbb/ccc", false},
		{map[string]bool{"": true}, "", true},
		{map[string]bool{"": true}, "/aaa", false},
		{map[string]bool{"*": true}, "/", true},
		{map[string]bool{"*": true}, "/aaa/bbb", true},
		{map[string]bool{"/aaa": true}, "/aaa", true},
		{map[string]bool{"/aaa": true}, "/aaa/bbb", false},
		{map[string]bool{"/aaa/*": true}, "/aaa", false},
		{map[string]bool{"/aaa/*": true}, "/aaa/bbb", true},
		{map[string]bool{"/aaa/*": true}, "/aaa/bbb/ccc", true},
		{map[string]bool{"/aaa/bbb": true}, "/aaa/bbb", true},
		{map[string]bool{"/aaa/bbb/*": true}, "/aaa/bbb", false},
		{map[string]bool{"/aaa/bbb/*": true}, "/aaa/bbb/ccc/ddd/eee", true},
		{map[string]bool{"/aaa": true}, "/aaabbbccc", false},
		{map[string]bool{"/aaa": true}, "/aaabbbccc/bbb", false},
		{map[string]bool{"/aaa*": true}, "/aaabbbccc", false},
		{map[string]bool{"/aaa*": true}, "/aaabbbccc", false},
		{map[string]bool{"/aaa*": true}, "/aaa", true},
		{map[string]bool{"/aaa*": true}, "/aaa/bbb", true},
		{map[string]bool{"/aaa*": true}, "/aaa/bbb/ccc", true},
		{map[string]bool{"/aaa*": true, "!/aaa/bbb": true}, "/aaa", true},
		{map[string]bool{"/aaa*": true, "!/aaa/bbb": true}, "/aaa/bbb", false},
		{map[string]bool{"/aaa*": true, "!/aaa/bbb": true}, "/aaa/bbb/ccc", true},
	}

	for i, p := range data {
		i++

		result := isPathInList(p.path, p.config)
		if result != p.disabled {
			t.Errorf(`[%d] failed: config "%v", path "%s", result "%v", expected "%v"`, i, p.config, p.path, result, p.disabled)
		}
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
