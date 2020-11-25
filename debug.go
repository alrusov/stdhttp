package stdhttp

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// debugBuildInfo --
func (h *HTTP) debugBuildInfo(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	info, ok := debug.ReadBuildInfo()
	if ok {
		SendJSON(w, http.StatusNotFound, info)
		return
	}

	Error(id, false, w, http.StatusNotImplemented, "Application is built without using modules", nil)
}

//----------------------------------------------------------------------------------------------------------------------------//

var (
	replaces = map[string]string{
		`(password\s*=\s*)(.*)(\s+)`:         `$1*$3`,
		`(secret\s*=\s*)(.*)(\s+)`:           `$1*$3`,
		`(\sdb_dsn_.*=\s*.*://.*:)(.*)(@.*)`: `$1*$3`,
	}

	replace = misc.NewReplace()
)

func init() {
	err := replace.AddMulti(replaces)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config.init: %s", err.Error())
		os.Exit(misc.ExProgrammerError)
	}
}

// debugEnv --
func (h *HTTP) debugEnv(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	s := strings.Join(os.Environ(), "\n")
	WriteContentHeader(w, ContentTypeText)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(replace.Do(s)))
}

//----------------------------------------------------------------------------------------------------------------------------//

// debugFreeOSmem --
func (h *HTTP) debugFreeOSmem(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	debug.FreeOSMemory()
	ReturnRefresh(id, w, r, http.StatusNoContent, "", nil, nil)
}

//----------------------------------------------------------------------------------------------------------------------------//

// debugGCstat --
func (h *HTTP) debugGCstat(id uint64, path string, w http.ResponseWriter, r *http.Request) {
	var stat debug.GCStats
	debug.ReadGCStats(&stat)
	SendJSON(w, http.StatusNotFound, stat)
}

//----------------------------------------------------------------------------------------------------------------------------//
