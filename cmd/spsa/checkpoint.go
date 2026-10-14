package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

const checkpointVersion = 2

// Count the underlying Int63 calls, including Intn rejection sampling. Wrapping
// Source (not Source64) preserves the original driver's Intn stream exactly.
type campaignSource struct {
	inner rand.Source
	draws uint64
}

func (s *campaignSource) Int63() int64    { s.draws++; return s.inner.Int63() }
func (s *campaignSource) Seed(seed int64) { s.inner.Seed(seed); s.draws = 0 }

func newCampaignRNG(seed int64) (*rand.Rand, *campaignSource) {
	s := &campaignSource{inner: rand.NewSource(seed)}
	return rand.New(s), s
}

type parameterRecipe struct {
	Name    string  `json:"name"`
	Default int     `json:"default"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	CEnd    float64 `json:"c_end"`
	CAmp    float64 `json:"c_amp"`
	AGain   float64 `json:"a_gain"`
}

// Bind both executables and ordered fixed UCI options. External network bytes
// are also pinned by the campaign manifest before play. The tuner contains the
// game harness, registry, update rule and RNG library.
// Opening order matters; hash the effective list after parsing/filtering.
type campaignRecipe struct {
	EngineSHA     string            `json:"engine_sha256"`
	EngineOptions []string          `json:"engine_options,omitempty"`
	TunerSHA      string            `json:"tuner_sha256"`
	OpeningsSHA   string            `json:"openings_sha256"`
	Openings      int               `json:"openings"`
	GoVersion     string            `json:"go_version"`
	GOOS          string            `json:"goos"`
	GOARCH        string            `json:"goarch"`
	GOMAXPROCS    int               `json:"gomaxprocs"`
	RNG           string            `json:"rng"`
	Seed          int64             `json:"seed"`
	Iterations    int               `json:"iterations"`
	Pairs         int               `json:"pairs"`
	Single        bool              `json:"single"`
	TimeMs        int               `json:"time_ms"`
	IncMs         int               `json:"inc_ms"`
	MaxMoves      int               `json:"max_moves"`
	REnd          float64           `json:"r_end"`
	AFrac         float64           `json:"a_frac"`
	Report        int               `json:"report"`
	LowPower      bool              `json:"low_power"`
	Params        []parameterRecipe `json:"params"`
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func makeRecipe(enginePath string, openings [][]string, params []*pvar,
	iters, pairs int, single bool, timeMs, incMs int, rEnd, aFrac float64,
	report int, seed int64, lowPower bool, engineOptions ...string) (campaignRecipe, error) {
	r := campaignRecipe{Openings: len(openings), GoVersion: runtime.Version(),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GOMAXPROCS: runtime.GOMAXPROCS(0),
		RNG: "math/rand.NewSource/Int63-count-v1", Seed: seed, Iterations: iters,
		Pairs: pairs, Single: single, TimeMs: timeMs, IncMs: incMs, MaxMoves: 200,
		REnd: rEnd, AFrac: aFrac, Report: report, LowPower: lowPower}
	r.EngineOptions = append([]string(nil), engineOptions...)
	var err error
	if r.EngineSHA, err = fileSHA(enginePath); err != nil {
		return r, fmt.Errorf("hash engine: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return r, err
	}
	if r.TunerSHA, err = fileSHA(executable); err != nil {
		return r, fmt.Errorf("hash tuner: %w", err)
	}
	b, err := json.Marshal(openings)
	if err != nil {
		return r, err
	}
	h := sha256.Sum256(b)
	r.OpeningsSHA = hex.EncodeToString(h[:])
	for _, p := range params {
		r.Params = append(r.Params, parameterRecipe{p.name, defaultOf(p.name), p.min, p.max, p.cEnd, p.cAmp, p.aGain})
	}
	return r, nil
}

type checkpoint struct {
	Version  int                `json:"version"`
	Iter     int                `json:"iter"`
	Time     string             `json:"time"`
	PlusPct  float64            `json:"plus_pct"`
	Theta    map[string]float64 `json:"theta"`
	Rounded  map[string]int     `json:"rounded"`
	Recipe   campaignRecipe     `json:"recipe"`
	RNGDraws uint64             `json:"rng_draws"`
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// Refuse mixed campaigns, even if -resume was accidentally omitted.
func prepareHistory(path string) error {
	if path == "" {
		return fmt.Errorf("a nonempty checkpoint path is required")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() != 0 {
		return fmt.Errorf("checkpoint history %s already contains data; use -resume or a new path", path)
	}
	return nil
}

// One complete, synced JSONL row is the commit point. An interrupted partial
// write is rejected on resume instead of silently falling back to older theta.
func writeCheckpoint(path string, iter int, params []*pvar, plusPct float64, recipe campaignRecipe, draws uint64) error {
	cp := checkpoint{Version: checkpointVersion, Iter: iter, Time: time.Now().UTC().Format(time.RFC3339),
		PlusPct: plusPct, Theta: map[string]float64{}, Rounded: map[string]int{}, Recipe: recipe, RNGDraws: draws}
	for _, p := range params {
		cp.Theta[p.name] = p.theta
		cp.Rounded[p.name] = clampRound(p.theta, p.min, p.max)
	}
	b, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b = append(b, '\n')
	n, err := f.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}

// Validate the whole history before changing theta. Replaying the exact draws
// for saved iterations both restores the RNG and verifies the recorded counter.
// This restores optimizer inputs, not wall-clock game outcomes or engine state.
func loadCheckpoint(path string, params []*pvar, recipe campaignRecipe) (int, *rand.Rand, *campaignSource, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil, nil, err
	}
	defer f.Close()
	rng, source := newCampaignRNG(recipe.Seed)
	r := bufio.NewReader(f)
	var last checkpoint
	rows := 0
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF && line == "" {
			break
		}
		if err != nil {
			return 0, nil, nil, fmt.Errorf("checkpoint row %d is incomplete: %w", rows+1, err)
		}
		rows++
		if strings.TrimSpace(line) == "" {
			return 0, nil, nil, fmt.Errorf("empty checkpoint row %d", rows)
		}
		var cp checkpoint
		dec := json.NewDecoder(bytes.NewBufferString(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cp); err != nil {
			return 0, nil, nil, fmt.Errorf("checkpoint row %d: %w", rows, err)
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return 0, nil, nil, fmt.Errorf("checkpoint row %d has trailing data", rows)
		}
		if cp.Version != checkpointVersion {
			return 0, nil, nil, fmt.Errorf("checkpoint row %d is legacy or unsupported; start a new tune", rows)
		}
		if !reflect.DeepEqual(cp.Recipe, recipe) {
			return 0, nil, nil, fmt.Errorf("checkpoint recipe mismatch at row %d; reuse exact binaries, openings and flags", rows)
		}
		if last.Iter == recipe.Iterations || cp.Iter != min(last.Iter+recipe.Report, recipe.Iterations) || cp.Iter <= last.Iter {
			return 0, nil, nil, fmt.Errorf("invalid checkpoint iteration %d at row %d", cp.Iter, rows)
		}
		if _, err := time.Parse(time.RFC3339, cp.Time); err != nil {
			return 0, nil, nil, fmt.Errorf("invalid checkpoint time at row %d", rows)
		}
		if !finite(cp.PlusPct) || cp.PlusPct < 0 || cp.PlusPct > 100 || len(cp.Theta) != len(params) || len(cp.Rounded) != len(params) {
			return 0, nil, nil, fmt.Errorf("invalid checkpoint values at row %d", rows)
		}
		for _, p := range params {
			v, ok := cp.Theta[p.name]
			rounded, roundOK := cp.Rounded[p.name]
			if !ok || !roundOK || !finite(v) || v < p.min || v > p.max || rounded != clampRound(v, p.min, p.max) {
				return 0, nil, nil, fmt.Errorf("invalid theta or rounded value for %s at row %d", p.name, rows)
			}
		}
		for k := last.Iter; k < cp.Iter; k++ {
			for j := 0; j < recipe.Pairs; j++ {
				for range params {
					rng.Intn(2)
				}
				if recipe.Openings > 0 {
					rng.Intn(recipe.Openings)
				}
			}
		}
		if source.draws != cp.RNGDraws {
			return 0, nil, nil, fmt.Errorf("RNG counter mismatch at row %d", rows)
		}
		last = cp
	}
	if rows == 0 {
		return 0, nil, nil, fmt.Errorf("checkpoint history is empty")
	}
	for _, p := range params {
		p.theta = last.Theta[p.name]
	}
	return last.Iter, rng, source, nil
}
