package claim

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ehrlich-b/ngn/engine"
)

const occupancyFEN = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"

type CPUQuota struct{ Path, Max string }
type OccupancySample struct {
	Role, Mask, Move, Error string
	BeforeTicks, AfterTicks uint64
	ClockTicks, Threads     int
	WallNS                  int64
	CPUPerWall              float64
	Session                 SessionRecord
}
type OccupancyReceipt struct {
	AtUTC, FEN, Command string
	Quotas              []CPUQuota
	Samples             []OccupancySample
	Passed              bool
	Error               string
}

func parseProcessStat(raw string) (uint64, int, error) {
	end := strings.LastIndex(raw, ")")
	if end < 0 {
		return 0, 0, fmt.Errorf("malformed process stat")
	}
	f := strings.Fields(raw[end+1:]) // begins at field 3 (state)
	if len(f) < 18 {
		return 0, 0, fmt.Errorf("short process stat")
	}
	u, e := strconv.ParseUint(f[11], 10, 64)
	if e != nil {
		return 0, 0, e
	}
	s, e := strconv.ParseUint(f[12], 10, 64)
	if e != nil {
		return 0, 0, e
	}
	n, e := strconv.Atoi(f[17])
	return u + s, n, e
}
func processCPU(pid int) (uint64, int, string, error) {
	b, e := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if e != nil {
		return 0, 0, "", e
	}
	ticks, threads, e := parseProcessStat(string(b))
	if e != nil {
		return 0, 0, "", e
	}
	b, e = os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if e != nil {
		return 0, 0, "", e
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "Cpus_allowed_list:") {
			return ticks, threads, strings.TrimSpace(strings.TrimPrefix(line, "Cpus_allowed_list:")), nil
		}
	}
	return 0, 0, "", fmt.Errorf("missing CPU mask")
}
func validateQuota(value string) error {
	f := strings.Fields(value)
	if len(f) != 2 {
		return fmt.Errorf("bad cpu.max: %s", value)
	}
	period, e := strconv.ParseFloat(f[1], 64)
	if e != nil || period <= 0 {
		return fmt.Errorf("bad quota period")
	}
	if f[0] == "max" {
		return nil
	}
	quota, e := strconv.ParseFloat(f[0], 64)
	if e != nil || quota/period < 16 {
		return fmt.Errorf("claim scope quota below 1600%%: %s", value)
	}
	return nil
}
func scopeQuotas() ([]CPUQuota, error) {
	b, e := os.ReadFile("/proc/self/cgroup")
	if e != nil {
		return nil, e
	}
	group := ""
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "0::") {
			group = strings.TrimPrefix(line, "0::")
		}
	}
	if group == "" {
		return nil, fmt.Errorf("cgroup v2 scope unavailable")
	}
	root := "/sys/fs/cgroup"
	dir := filepath.Join(root, group)
	var out []CPUQuota
	for {
		path := filepath.Join(dir, "cpu.max")
		value, err := os.ReadFile(path)
		if err == nil {
			out = append(out, CPUQuota{path, strings.TrimSpace(string(value))})
			if e = validateQuota(string(value)); e != nil {
				return out, e
			}
		} else if !os.IsNotExist(err) {
			return out, err
		}
		if dir == root {
			break
		}
		dir = filepath.Dir(dir)
	}
	if len(out) == 0 {
		return out, fmt.Errorf("missing scope cpu.max")
	}
	return out, nil
}
func validateOccupancy(s OccupancySample) error {
	if s.WallNS <= 0 || s.ClockTicks <= 0 || s.AfterTicks < s.BeforeTicks {
		return fmt.Errorf("invalid CPU counters")
	}
	if s.Mask != CPUSet || s.CPUPerWall < 7.0 {
		return fmt.Errorf("%s: need mask %s and CPU/wall >=7, got %s / %.3f", s.Role, CPUSet, s.Mask, s.CPUPerWall)
	}
	return nil
}
func occupancySample(role string, artifact Artifact, selected map[string]string, hz int) (out OccupancySample, err error) {
	out.Role, out.ClockTicks = role, hz
	s, err := StartSession(artifact, selected) // exactly the game launcher/environment/options
	if s != nil {
		defer func() {
			if e := s.Close(); err == nil {
				err = e
			}
			out.Session = s.Snapshot()
			if err != nil {
				out.Error = err.Error()
			}
		}()
	}
	if err != nil {
		return out, err
	}
	out.BeforeTicks, _, _, err = processCPU(s.record.PID)
	if err != nil {
		return out, err
	}
	start := time.Now()
	out.Move, _, _, err = s.Search(PositionCommand(occupancyFEN, nil), "go movetime 5000", 15e9)
	out.WallNS = time.Since(start).Nanoseconds()
	if err != nil {
		return out, err
	}
	out.AfterTicks, out.Threads, out.Mask, err = processCPU(s.record.PID)
	if err != nil {
		return out, err
	}
	out.CPUPerWall = float64(out.AfterTicks-out.BeforeTicks) / float64(hz) / (float64(out.WallNS) / 1e9)
	pos, err := engine.ParseFEN(occupancyFEN)
	if err != nil {
		return out, err
	}
	if _, err = LegalMove(pos, out.Move); err != nil {
		return out, err
	}
	return out, validateOccupancy(out)
}

// ProbeOccupancy runs once before the screen in its own exclusive scope. The
// confirmation binds this same receipt; no width is inferred from UCI labels.
func ProbeOccupancy(p Plan, selected map[string]map[string]string) (out OccupancyReceipt, err error) {
	out = OccupancyReceipt{AtUTC: time.Now().UTC().Format(time.RFC3339Nano), FEN: occupancyFEN, Command: "go movetime 5000"}
	defer func() {
		if err != nil {
			out.Error = err.Error()
		}
	}()
	out.Quotas, err = scopeQuotas()
	if err != nil {
		return out, err
	}
	text, err := command("getconf", "CLK_TCK")
	if err != nil {
		return out, err
	}
	hz, err := strconv.Atoi(text)
	if err != nil || hz <= 0 {
		return out, fmt.Errorf("bad CLK_TCK: %s", text)
	}
	roles := []Anchor{{Name: "candidate", Engine: p.Candidate.Engine}}
	roles = append(roles, p.Anchors...)
	for _, role := range roles {
		sample, e := occupancySample(role.Name, role.Engine, selected[role.Name], hz)
		out.Samples = append(out.Samples, sample)
		if e != nil {
			return out, e
		}
	}
	out.Passed = true
	return out, nil
}
