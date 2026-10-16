package claim

import (
	"strings"
	"testing"
)

func TestProcessCPUCountersIncludeAllThreads(t *testing.T) {
	// Process comm can contain spaces and closing parentheses. Fields 14/15
	// are aggregate CPU ticks; fields 16/17 are child ticks and must be ignored.
	raw := "123 (engine ) worker) R 1 2 3 4 5 6 7 8 9 10 3100 800 9999 9999 20 19 12"
	ticks, threads, err := parseProcessStat(raw)
	if err != nil || ticks != 3900 || threads != 12 {
		t.Fatalf("wrong counters %d/%d: %v", ticks, threads, err)
	}
	for _, bad := range []string{"garbage", "123 (engine) R 1", strings.Replace(raw, "3100", "bad", 1)} {
		if _, _, err := parseProcessStat(bad); err == nil {
			t.Fatal("bad process stat admitted")
		}
	}
}

func TestV4OccupancyAndQuotaPrecondition(t *testing.T) {
	s := OccupancySample{Role: "candidate", Mask: CPUSet, BeforeTicks: 10, AfterTicks: 3910, ClockTicks: 100, WallNS: 5e9, CPUPerWall: 7.8}
	if err := validateOccupancy(s); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*OccupancySample){
		func(s *OccupancySample) { s.CPUPerWall = 6.999 },
		func(s *OccupancySample) { s.Mask = "0-15" },
		func(s *OccupancySample) { s.AfterTicks = 0 },
	} {
		bad := s
		change(&bad)
		if validateOccupancy(bad) == nil {
			t.Fatal("unproved width admitted")
		}
	}
	for _, quota := range []string{"max 100000", "1600000 100000", "2000000 100000"} {
		if err := validateQuota(quota); err != nil {
			t.Fatal(err)
		}
	}
	for _, quota := range []string{"600000 100000", "799999 100000", "1599999 100000", "max 0", ""} {
		if validateQuota(quota) == nil {
			t.Fatalf("bad quota admitted: %s", quota)
		}
	}
}
