package stdhttp

import (
	"net/http"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

type infoBlock struct {
	Application *applicationBlock `json:"applicaion"`
	Runtime     *runtimeBlock     `json:"runtime"`
	Extra       interface{}       `json:"extra"`
	LastLog     interface{}       `json:"lastLog"`
	Endpoints   []endpointBlock   `json:"endpoints"`
}

type applicationBlock struct {
	Copyright string    `json:"copyright"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	BiildTime time.Time `json:"biildTime"`
	GoVersion string    `json:"goVersion"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
}

type idDef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type runtimeBlock struct {
	StartTime      time.Time `json:"startTime"`
	Now            time.Time `json:"now"`
	Uptime         int64     `json:"upTime"`
	PID            int       `json:"pid"`
	User           idDef     `json:"user"`
	Group          idDef     `json:"group"`
	EffectiveUser  idDef     `json:"effectiveUser"`
	EffectiveGroup idDef     `json:"effectiveGroup"`
	Host           string    `json:"host"`
	IP             []string  `json:"ip"`
	CommandLine    string    `json:"commandLine"`
	Application    string    `json:"application"`
	WorkDir        string    `json:"workDir"`
	LogLevel       string    `json:"logLevel"`
	LogFile        string    `json:"logFile"`
	AllocSys       uint64    `json:"allocSys"`
	HeapSys        uint64    `json:"heapSys"`
	HeapAlloc      uint64    `json:"heapAlloc"`
	NumCPU         int       `json:"numCPU"`
	GoMaxProcs     int       `json:"goMaxProcs"`
	NumGoroutine   int       `json:"numGoroutine"`
}

type endpointBlock struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ExtraInfoFunc --
type ExtraInfoFunc func() interface{}

var (
	infoMutex = new(sync.Mutex)
	extraFunc = ExtraInfoFunc(nil)

	info = &infoBlock{}
)

//----------------------------------------------------------------------------------------------------------------------------//

func initInfo() {
	info.Application = &applicationBlock{
		Name:      misc.AppName(),
		Version:   misc.AppVersion(),
		BiildTime: misc.BuildTimeTS(),
		Copyright: misc.Copyright(),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	info.Runtime = &runtimeBlock{
		StartTime: misc.AppStartTime(),
		PID:       os.Getpid(),
		User: idDef{
			ID: os.Getuid(),
		},
		Group: idDef{
			ID: os.Getgid(),
		},
		EffectiveUser: idDef{
			ID: os.Geteuid(),
		},
		EffectiveGroup: idDef{
			ID: os.Getegid(),
		},
		Application: misc.AppFullName(),
		WorkDir:     misc.AppWorkDir(),
		NumCPU:      runtime.NumCPU(),
		GoMaxProcs:  runtime.GOMAXPROCS(-1),
	}

	cmd := ""
	for i := 0; i < len(os.Args); i++ {
		cmd += " " + os.Args[i]
	}
	cmd = strings.TrimSpace(cmd)

	if cmd == "" {
		cmd = "?"
	}

	info.Runtime.Host, _ = os.Hostname()
	info.Runtime.CommandLine = cmd

	if u, err := user.LookupId(strconv.Itoa(info.Runtime.User.ID)); err == nil {
		info.Runtime.User.Name = u.Username
	}

	if g, err := user.LookupGroupId(strconv.Itoa(info.Runtime.Group.ID)); err == nil {
		info.Runtime.Group.Name = g.Name
	}

	if u, err := user.LookupId(strconv.Itoa(info.Runtime.EffectiveUser.ID)); err == nil {
		info.Runtime.EffectiveUser.Name = u.Username
	}

	if g, err := user.LookupGroupId(strconv.Itoa(info.Runtime.EffectiveGroup.ID)); err == nil {
		info.Runtime.EffectiveGroup.Name = g.Name
	}

	info.Endpoints = []endpointBlock{
		endpointBlock{
			Name:        "/",
			Description: "Root page",
		},
		{
			Name:        "/info",
			Description: "Get information about the application",
		},
		{
			Name:        "/ping",
			Description: "Checking if the application running",
		},
		{
			Name:        "/set-log-level",
			Description: "Temporary log level change",
		},
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// AddEndpointsInfo --
func AddEndpointsInfo(df map[string]string) {
	if info.Application == nil {
		initInfo()
	}

	for n, v := range df {
		info.Endpoints = append(info.Endpoints,
			endpointBlock{
				Name:        n,
				Description: v,
			},
		)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// SetExtraInfoFunc --
func SetExtraInfoFunc(f ExtraInfoFunc) {
	extraFunc = f
}

//----------------------------------------------------------------------------------------------------------------------------//

func showInfo(w http.ResponseWriter) {
	infoMutex.Lock()

	if info.Application == nil {
		initInfo()
	}

	if extraFunc != nil {
		info.Extra = extraFunc()
	}

	ip := []string{}
	if ipMap, err := misc.GetMyIPs(); err == nil {
		for i := range ipMap {
			ip = append(ip, i)
		}
	}
	sort.Strings(ip)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	info.Runtime.Now = misc.NowUTC()
	info.Runtime.Uptime = int64(info.Runtime.Now.Sub(info.Runtime.StartTime).Seconds())
	_, _, info.Runtime.LogLevel = log.GetCurrentLogLevel()
	info.Runtime.LogFile = log.FileName()
	info.Runtime.IP = ip
	info.Runtime.AllocSys = mem.Sys
	info.Runtime.HeapSys = mem.HeapSys
	info.Runtime.HeapAlloc = mem.HeapAlloc
	info.Runtime.NumGoroutine = runtime.NumGoroutine()

	info.LastLog = log.GetLastLog()

	SendJSON(w, http.StatusOK, info)

	infoMutex.Unlock()
}

//----------------------------------------------------------------------------------------------------------------------------//
