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

	"github.com/alrusov/bufpool"
	"github.com/alrusov/loadavg"
	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

const url404 = `<<< 404 >>>`

type (
	infoBlock struct {
		Application *applicationBlock        `json:"application"`
		Runtime     *runtimeBlock            `json:"runtime"`
		Endpoints   map[string]*endpointInfo `json:"endpoints"`
		LastLog     interface{}              `json:"lastLog"`
		Extra       interface{}              `json:"extra"`
	}

	applicationBlock struct {
		Copyright   string    `json:"copyright"`
		AppName     string    `json:"appName"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Version     string    `json:"version"`
		Tags        string    `json:"tags"`
		BuildTime   time.Time `json:"buildTime"`
		GoVersion   string    `json:"goVersion"`
		OS          string    `json:"os"`
		Arch        string    `json:"arch"`
	}

	idDef struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	runtimeBlock struct {
		StartTime       time.Time         `json:"startTime"`
		Now             time.Time         `json:"now"`
		Uptime          int64             `json:"upTime"`
		PID             int               `json:"pid"`
		User            idDef             `json:"user"`
		Group           idDef             `json:"group"`
		EffectiveUser   idDef             `json:"effectiveUser"`
		EffectiveGroup  idDef             `json:"effectiveGroup"`
		Host            string            `json:"host"`
		IP              []string          `json:"ip"`
		CommandLine     string            `json:"commandLine"`
		Application     string            `json:"application"`
		WorkDir         string            `json:"workDir"`
		LogLevel        string            `json:"logLevel"`
		LogFile         string            `json:"logFile"`
		ProfilerEnabled bool              `json:"profilerEnabled"`
		AllocSys        uint64            `json:"allocSys"`
		HeapSys         uint64            `json:"heapSys"`
		HeapInuse       uint64            `json:"HeapInuse"`
		HeapObjects     uint64            `json:"HeapObjects"`
		StackSys        uint64            `json:"stackSys"`
		StackInuse      uint64            `json:"StackInuse"`
		NumCPU          int               `json:"numCPU"`
		GoMaxProcs      int               `json:"goMaxProcs"`
		NumGoroutine    int               `json:"numGoroutine"`
		LoadAvgPeriod   int               `json:"loadAvgPeriod"`
		Requests        *urlStat          `json:"requests"`
		Pools           misc.InterfaceMap `json:"pools"`
		poolsUpdate     map[string]PoolStatFunc
	}

	endpointInfo struct {
		Description string   `json:"description"`
		Stat        *urlStat `json:"stat"`
	}

	urlStat struct {
		Total   uint64 `json:"total"`
		la      *loadavg.LoadAvg
		LoadAvg float64 `json:"loadAvg"`
	}

	// PoolStatFunc --
	PoolStatFunc func() interface{}

	// ExtraInfoFunc --
	ExtraInfoFunc func() interface{}
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

		Pools: map[string]interface{}{},
		poolsUpdate: map[string]PoolStatFunc{
			"bufpool": bufpool.GetStat,
		},
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
		"/___.css":                      "General purpose css. Parameters: -",
		"/":                             "Root page. Parameters: -",
		"/debug/build-info":             "Show applications build info. Parameters: -",
		"/debug/env":                    "Show environment. Parameters: -",
		"/debug/free-os-memory":         "Try to release an unused memory to the OS. Parameters: -",
		"/debug/gc-stat":                "Garbage collector statistics. Parameters: -",
		"/debug/mem-stat":               "Memory statistics. Parameters: -",
		"/debug/pprof":                  "Profiler root. Parameters: -",
		"/favicon.ico":                  "favicon.ico. Parameters: -",
		"/maintenance":                  "Application maintenance page. Parameters: -",
		"/maintenance/config":           "Get app config (secured). Parameters: -",
		"/maintenance/exit":             "Exit application: pid=<pid>, [code=<code>]",
		"/maintenance/info":             "Get app information. Parameters: -",
		"/maintenance/profiler-disable": "Disable profiler. Parameters: -",
		"/maintenance/profiler-enable":  "Enable profiler. Parameters: -",
		"/maintenance/set-log-level":    "Temporarily change log level. Parameters: level=<level>",
		"/status":                       "Application current status. Parameters: -",
		"/status/ping":                  "Checking if the application is running. Parameters: -",
		"/tools/jwt-login":              "Get jwt token. Parameters: u=<username>, p=<password>",
		"/tools/sha":                    "Calculate sha512 hash. Parameters: p=<string>",
	})
}

//----------------------------------------------------------------------------------------------------------------------------//

// AddEndpointsInfo --
func (h *HTTP) AddEndpointsInfo(list misc.StringMap) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

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

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) newStat() *urlStat {
	return &urlStat{
		la: loadavg.Init(h.commonConfig.LoadAvgPeriod),
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

func (h *HTTP) updateEndpointStat(path string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

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
	h.mutex.Lock()
	defer h.mutex.Unlock()

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

	for name, f := range info.Runtime.poolsUpdate {
		info.Runtime.Pools[name] = f()
	}

	info.LastLog = log.GetLastLog()

	for _, ep := range info.Endpoints {
		ep.Stat.update()
	}

	SendJSON(w, http.StatusOK, info)
}

//----------------------------------------------------------------------------------------------------------------------------//

// AddPool --
func (h *HTTP) AddPool(name string, f PoolStatFunc) {
	h.info.Runtime.poolsUpdate[name] = f
}

//----------------------------------------------------------------------------------------------------------------------------//
