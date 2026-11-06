package httpapi

import (
	"math"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// serverInfoResponse extends Firefish's server-info with the usage numbers its
// serverStats Streaming channel would carry. The upstream fields keep their
// shape; `known` tells the client whether the platform could read the value,
// so a non-Linux deployment shows "未知" instead of a confident zero.
type serverInfoResponse struct {
	Machine string `json:"machine"`
	CPU     struct {
		Model string  `json:"model"`
		Cores int     `json:"cores"`
		Usage float64 `json:"usage"`
		Known bool    `json:"known"`
	} `json:"cpu"`
	Mem struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
		Known bool  `json:"known"`
	} `json:"mem"`
	FS struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
		Known bool  `json:"known"`
	} `json:"fs"`
	Load struct {
		One     float64 `json:"one"`
		Five    float64 `json:"five"`
		Fifteen float64 `json:"fifteen"`
		Known   bool    `json:"known"`
	} `json:"load"`
}

var (
	cpuSamplerOnce sync.Once
	cpuUsageBits   atomic.Uint64
)

// startCPUSampler keeps one background reading of /proc/stat so a request does
// not have to wait for a sampling interval.
func startCPUSampler() {
	cpuSamplerOnce.Do(func() {
		go func() {
			prevTotal, prevIdle, ok := readCPUStat()
			if !ok {
				return
			}
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				total, idle, ok := readCPUStat()
				if !ok {
					continue
				}
				dTotal := float64(total - prevTotal)
				dIdle := float64(idle - prevIdle)
				prevTotal, prevIdle = total, idle
				if dTotal <= 0 {
					continue
				}
				usage := (1 - dIdle/dTotal) * 100
				usage = math.Max(0, math.Min(100, usage))
				cpuUsageBits.Store(math.Float64bits(usage))
			}
		}()
	})
}

// handleServerInfo implements POST /api/server-info.
//
// Firefish serves this publicly when the instance enables machine stats and
// otherwise reports zeros. Cinderfish requires a credential instead: the
// values (hostname, memory, disk) are real, and a personal instance has no
// reason to publish its host profile to the world.
func (s *Server) handleServerInfo(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	startCPUSampler()

	var out serverInfoResponse
	if host, err := os.Hostname(); err == nil {
		out.Machine = host
	} else {
		out.Machine = "Not specified"
	}

	out.CPU.Cores = runtime.NumCPU()
	out.CPU.Model = readCPUModel()
	if out.CPU.Model == "" {
		out.CPU.Model = "Not specified"
	}
	out.CPU.Usage = math.Float64frombits(cpuUsageBits.Load())
	_, _, cpuOK := readCPUStat()
	out.CPU.Known = cpuOK

	if total, available, ok := readMemInfo(); ok {
		out.Mem.Total = int64(total)
		out.Mem.Used = int64(total - available)
		out.Mem.Known = true
	}

	root := s.mediaRoot()
	if total, used, ok := readFSStat(root); ok {
		out.FS.Total = int64(total)
		out.FS.Used = int64(used)
		out.FS.Known = true
	}

	if one, five, fifteen, ok := readLoadAvg(); ok {
		out.Load.One, out.Load.Five, out.Load.Fifteen = one, five, fifteen
		out.Load.Known = true
	}

	writeJSON(w, http.StatusOK, out)
}

// mediaRoot is the filesystem the drive lives on, falling back to the working
// directory when the deployment keeps media elsewhere and tests leave it unset.
func (s *Server) mediaRoot() string {
	if s.cfg != nil && s.cfg.Media.LocalPath != "" && s.cfg.Media.Backend != "s3" {
		if _, err := os.Stat(s.cfg.Media.LocalPath); err == nil {
			return s.cfg.Media.LocalPath
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
