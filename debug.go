package stdhttp

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// debugBuildInfo --
func (h *HTTP) debugBuildInfo(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	info, ok := debug.ReadBuildInfo()
	if ok {
		SendJSON(w, r, http.StatusNotFound, info)
		return
	}

	Error(id, false, w, r, http.StatusNotImplemented, "Application is built without using modules", nil)
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
func (h *HTTP) debugEnv(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	s := strings.Join(os.Environ(), "\n")
	WriteReply(w, r, http.StatusOK, ContentTypeText, nil, []byte(replace.Do(s)))
}

//----------------------------------------------------------------------------------------------------------------------------//

// debugFreeOSmem --
func (h *HTTP) debugFreeOSmem(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	debug.FreeOSMemory()
	ReturnRefresh(id, w, r, http.StatusNoContent, prefix+"/maintenance", nil, nil)
}

//----------------------------------------------------------------------------------------------------------------------------//

// debugGCstat --
func (h *HTTP) debugGCstat(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	var stat debug.GCStats
	debug.ReadGCStats(&stat)
	SendJSON(w, r, http.StatusNotFound, stat)
}

//----------------------------------------------------------------------------------------------------------------------------//

// memStat --
func (h *HTTP) debugMemStat(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	var stat runtime.MemStats
	runtime.ReadMemStats(&stat)
	SendJSON(w, r, http.StatusNotFound, stat)
}

//----------------------------------------------------------------------------------------------------------------------------//
