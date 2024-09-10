package stdhttp

import (
	"net/http"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alrusov/config"
	"github.com/alrusov/loadavg"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

const url404 = `/errors/404`

type (
	InfoBlock struct {
		Application *applicationBlock        `json:"application" comment:"Application info"`
		Runtime     *runtimeBlock            `json:"runtime" comment:"Runtime info"`
		Endpoints   map[string]*endpointInfo `json:"endpoints" comment:"Enpoints info"`
		LastLog     []string                 `json:"lastLog" comment:"Last lines from the log"`
		Extra       any                      `json:"extra" comment:"Application extra info"`
	}

	applicationBlock struct {
		Copyright   string    `json:"copyright" comment:"Copyright"`
		AppName     string    `json:"appName" comment:"Common name"`
		Name        string    `json:"name" comment:"Name"`
		Description string    `json:"description" comment:"Description"`
		Version     string    `json:"version" comment:"Version"`
		Tags        string    `json:"tags" comment:"Tags"`
		BuildTime   time.Time `json:"buildTime" comment:"Build time"`
		GoVersion   string    `json:"goVersion" comment:"Golang version"`
		OS          string    `json:"os" comment:"OS"`
		Arch        string    `json:"arch" comment:"Architecture"`
	}

	idDef struct {
		ID   int    `json:"id" comment:"ID"`
		Name string `json:"name" comment:"Name"`
	}

	runtimeBlock struct {
		StartTime       time.Time       `json:"startTime" comment:"Start time"`
		Now             time.Time       `json:"now" comment:"Current time"`
		Uptime          int64           `json:"upTime" comment:"Uptime"`
		PID             int             `json:"pid" comment:"Process ID"`
		User            idDef           `json:"user" comment:"User"`
		Group           idDef           `json:"group" comment:"Group"`
		EffectiveUser   idDef           `json:"effectiveUser" comment:"Effective user"`
		EffectiveGroup  idDef           `json:"effectiveGroup" comment:"Effective group"`
		Host            string          `json:"host" comment:"Host name"`
		IP              []string        `json:"ip" comment:"Host IPs"`
		CommandLine     string          `json:"commandLine" comment:"Command line"`
		Application     string          `json:"application" comment:"Application name"`
		WorkDir         string          `json:"workDir" comment:"Working directory"`
		LogLevel        string          `json:"logLevel" comment:"Default log level"`
		LogFile         string          `json:"logFile" comment:"Current log file name"`
		ProfilerEnabled bool            `json:"profilerEnabled" comment:"Is profiler enabled"`
		AllocSys        uint64          `json:"allocSys" comment:"Allocated heap"`
		HeapSys         uint64          `json:"heapSys" comment:"Available heap"`
		HeapInuse       uint64          `json:"heapInuse" comment:"Heap in use"`
		HeapObjects     uint64          `json:"heapObjects" comment:"Number of heap objects"`
		StackSys        uint64          `json:"stackSys" comment:"Allocated stack"`
		StackInuse      uint64          `json:"stackInuse" comment:"Stack inuse"`
		NumCPU          int             `json:"numCPU" comment:"Number of CPU"`
		GoMaxProcs      int             `json:"goMaxProcs" comment:"Max procs for golang"`
		NumGoroutine    int             `json:"numGoroutine" comment:"Number of goroutines"`
		LoadAvgPeriod   config.Duration `json:"loadAvgPeriod" comment:"Load average period"`
		Requests        *urlStat        `json:"requests" comment:"Requests statistic"`
	}

	endpointInfo struct {
		Description string   `json:"description" comment:"Description"`
		Stat        *urlStat `json:"stat" comment:"Statistics"`
	}

	urlStat struct {
		Total   uint64 `json:"total" comment:"Total requests"`
		la      *loadavg.LoadAvg
		LoadAvg float64 `json:"loadAvg" comment:"Load average"`
	}

	// ExtraInfoFunc --
	ExtraInfoFunc func() any
)

//----------------------------------------------------------------------------------------------------------------------------//

// SetName --
func (h *HTTP) SetName(name string, description string) {
	if name != "" {
		h.info.Application.Name = name
	}
	if description != "" {
		h.info.Application.Description = description
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) initInfo() {
	info := h.info

	info.Application = &applicationBlock{
		AppName:   misc.AppName(),
		Version:   misc.AppVersion(),
		Tags:      misc.AppTags(),
		BuildTime: misc.BuildTimeTS(),
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
		Application:   misc.AppFullName(),
		WorkDir:       misc.AppWorkDir(),
		NumCPU:        runtime.NumCPU(),
		GoMaxProcs:    runtime.GOMAXPROCS(-1),
		LoadAvgPeriod: h.commonConfig.LoadAvgPeriod,
		Requests:      h.newStat(),
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

	info.Endpoints = make(map[string]*endpointInfo)
	h.AddEndpointsInfo(misc.StringMap{
		url404:                          `Cumulatiive "Not Found" endpoint`,
		"/___.css":                      "General purpose css",
		"/":                             "Root page",
		"/debug/build-info":             "Show applications build info",
		"/debug/env":                    "Show environment",
		"/debug/free-os-memory":         "Try to release an unused memory to the OS",
		"/debug/gc-stat":                "Garbage collector statistics",
		"/debug/mem-stat":               "Memory statistics",
		"/debug/pprof":                  "Profiler root",
		"/favicon.ico":                  "favicon.ico",
		"/maintenance":                  "Application maintenance page",
		"/maintenance/config":           "Get secured app config",
		"/maintenance/endpoints":        "Known endpoints",
		"/maintenance/exit":             "Exit application (pid=<pid>, [code=<code>])",
		"/maintenance/info":             "Get app information",
		"/maintenance/profiler-disable": "Disable profiler",
		"/maintenance/profiler-enable":  "Enable profiler",
		"/maintenance/set-log-level":    "Temporarily change log level (level=<level>)",
		"/status":                       "Application current status",
		"/status/ping":                  "Checking if the application is running",
		"/tools/sha":                    "Calculate hash (p=<string>, salt=<string>)",
	})
}

//----------------------------------------------------------------------------------------------------------------------------//

// AddEndpointsInfo --
func (h *HTTP) AddEndpointsInfo(list misc.StringMap) {
	h.Lock()
	defer h.Unlock()

	h.addEndpointsInfo(list)
}

func (h *HTTP) addEndpointsInfo(list misc.StringMap) {
	for name, descr := range list {
		h.info.Endpoints[name] =
			&endpointInfo{
				Description: descr,
				Stat:        h.newStat(),
			}
	}
}

func (h *HTTP) DelEndpointsInfo(list misc.StringMap) {
	h.Lock()
	defer h.Unlock()

	h.delEndpointsInfo(list)
}

func (h *HTTP) delEndpointsInfo(list misc.StringMap) {
	for name := range list {
		delete(h.info.Endpoints, name)
	}
}

func (h *HTTP) EndpointDescription(name string) string {
	h.Lock()
	defer h.Unlock()

	e := h.info.Endpoints[name]
	if e == nil {
		return ""
	}

	return e.Description
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) newStat() *urlStat {
	return &urlStat{
		la: loadavg.Init(h.commonConfig.LoadAvgPeriod.D()),
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) updateEndpointStat(path string) {
	h.Lock()
	defer h.Unlock()

	ep, exists := h.info.Endpoints[path]
	if !exists {
		h.addEndpointsInfo(misc.StringMap{path: "<<< NO DESCRIPTION >>>"})
		ep, exists = h.info.Endpoints[path]
		if !exists {
			return
		}
	}

	ep.Stat.inc()
}

//----------------------------------------------------------------------------------------------------------------------------//

func (s *urlStat) inc() {
	atomic.AddUint64(&s.Total, 1)
	s.la.Add(1)
}

func (s *urlStat) update() {
	s.LoadAvg = s.la.Value()
}

//----------------------------------------------------------------------------------------------------------------------------//

// SetExtraInfoFunc --
func (h *HTTP) SetExtraInfoFunc(f ExtraInfoFunc) {
	h.extraFunc = f
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) showInfo(id uint64, prefix string, path string, w http.ResponseWriter, r *http.Request) {
	h.Lock()
	defer h.Unlock()

	info := h.info

	if h.extraFunc != nil {
		h.info.Extra = h.extraFunc()
	}

	ip := []string{}
	if ipMap, err := misc.GetMyIPs(); err == nil {
		for i := range ipMap {
			ip = append(ip, i)
		}
	}
	sort.Strings(ip)

	var mem runtime.MemStats
	//runtime.GC()
	runtime.ReadMemStats(&mem)

	info.Runtime.Now = misc.NowUTC()
	info.Runtime.Uptime = int64(info.Runtime.Now.Sub(info.Runtime.StartTime).Seconds())
	info.Runtime.IP = ip
	_, _, info.Runtime.LogLevel = log.CurrentLogLevelEx()
	info.Runtime.LogFile = log.FileName()
	info.Runtime.ProfilerEnabled = h.commonConfig.ProfilerEnabled
	info.Runtime.AllocSys = mem.Sys
	info.Runtime.HeapSys = mem.HeapSys
	info.Runtime.HeapInuse = mem.HeapInuse
	info.Runtime.HeapObjects = mem.HeapObjects
	info.Runtime.StackSys = mem.StackSys
	info.Runtime.StackInuse = mem.StackInuse
	info.Runtime.NumGoroutine = runtime.NumGoroutine()
	info.Runtime.Requests.update()

	info.LastLog = log.GetLastLog()

	for _, ep := range info.Endpoints {
		ep.Stat.update()
	}

	SendJSON(w, r, http.StatusOK, info)
}

//----------------------------------------------------------------------------------------------------------------------------//
