package stdhttp

import (
	"testing"
)

//----------------------------------------------------------------------------------------------------------------------------//

func TestIsPathInList(t *testing.T) {
	type testData struct {
		config map[string]bool
		path   string
		inList bool
	}

	data := []testData{
		{map[string]bool{}, "", false},
		{map[string]bool{}, "/aaa/bbb/ccc", false},
		{map[string]bool{"": true}, "", true},
		{map[string]bool{"": true}, "/aaa", false},
		{map[string]bool{"*": true}, "/", true},
		{map[string]bool{"/": true}, "/", true},
		{map[string]bool{"!/": true}, "/", false},
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
		{map[string]bool{"/aaa*": true, "/aaa/bbb": true}, "/aaa", true},
		{map[string]bool{"/aaa*": true, "/aaa/bbb": true}, "/aaa/bbb", true},
		{map[string]bool{"/aaa*": true, "/aaa/bbb": true}, "/aaa/bbb/ccc", true},
		{map[string]bool{"/aaa*": true, "/aaa/bbb*": true}, "/aaa/bbb/ccc", true},
		{map[string]bool{"/aaa*": true, "/aaa/bbb/*": true}, "/aaa/bbb/ccc", true},
		{map[string]bool{"/aaa*": true, "/aaa/bbb/ccc*": true}, "/aaa/bbb/ccc", true},
		{map[string]bool{"/aaa*": true, "/aaa/bbb/ccc/*": true}, "/aaa/bbb/ccc", true},
	}

	for i, p := range data {
		i++

		_, exists := isPathInList(p.path, p.config)
		if exists != p.inList {
			t.Errorf(`[%d] failed: config "%v", path "%s", result "%v", expected "%v"`, i, p.config, p.path, exists, p.inList)
		}
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
