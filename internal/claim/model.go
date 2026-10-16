// Package claim is NGN's independently written, fixed-sample claim instrument.
// It uses only the independently produced NGN board/move layer and Go stdlib.
package claim

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
const OpeningSHA = "974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222"

// Verified by the zero-game identity receipt: UCI Maelstrom v3.3.0.
const MaelstromSHA = "66ea969530d63f5904114079f518ba618b0deb4e269eb0c95d7496960379ea53"
const ViridithasSHA = "35b32c4acc3216fd9d1084bebad90452cee2e8725af4bdd0f175cc42c4e3def3"

// Earlier executables/receipts remain frozen for historical analysis.
const Schema = "NGN-CLAIM-4"
const ClaimThreshold = 3600
const Threads = 8
const HashMB = 256
const CPUSet = "0,2,4,6,8,10,12,14"
const BerserkSHA = "7b9b5898a781cbe0757caef0b7281df5cf5029d17e67ed3df4824f0eb1c0125d"

// The public v13.00 source is HCE, not the commercial NNUE edition rated
// by this pool. Resolve the correct artifact before any live v4 receipt.
const EtherealSHA = "UNRESOLVED-COMMERCIAL-NNUE"
const AkimboSHA = "879c769035d948068512eb9de02c0f356727b6045e10b95ac813497ebd8d7eee"
const BlackMarlinSHA = "980bd51860fc1cc61fd2216d5ca25fbcef1b5488bde1bce379d022b38b750642"

type Artifact struct {
	Path   string
	SHA256 string
}
type Identity struct {
	Engine                              Artifact
	Net                                 Artifact
	SourceRoot, SourceCommit, GoVersion string
	BuildArgv                           []string
	// Manifest, data, training, export and parity are separately bound artifacts.
	Provenance map[string]Artifact
}
type Anchor struct {
	Name         string
	Label        int
	Engine       Artifact
	Dependencies []Artifact
}

func Anchors() []Anchor {
	return []Anchor{
		{Name: "Ethereal 13.00", Label: 3581, Engine: Artifact{Path: "/home/ehrli/ngn-data/claim-v4-20261016/anchors/ethereal13/ethereal-13.00-nnue", SHA256: EtherealSHA}},
		{Name: "akimbo 1.0.0", Label: 3614, Engine: Artifact{Path: "/home/ehrli/ngn-data/claim-v4-20261016/anchors/akimbo1/akimbo-1.0.0-avx2", SHA256: AkimboSHA}},
		{Name: "Black Marlin 9.0", Label: 3623, Engine: Artifact{Path: "/home/ehrli/ngn-data/claim-v4-20261016/anchors/blackmarlin9/blackmarlin-linux-x86-64-v3", SHA256: BlackMarlinSHA}},
		{Name: "Berserk 12", Label: 3741, Engine: Artifact{Path: "/home/ehrli/ngn-data/claim-v3-20261008/anchors/berserk12/berserk-12-linux-native", SHA256: BerserkSHA}},
	}
}

type Opening struct {
	Index   int
	FEN     string
	History []string
	Line    string
}
type Plan struct {
	Schema, Mode            string
	Candidate               Identity
	Anchors                 []Anchor
	Openings                Artifact
	Count, Concurrency, Cap int
	Threads, HashMB         int
	CPUSet                  string
	ClaimThreshold          int
	BaseNS, IncrementNS     int64
	Runner                  Artifact
	WindowsBridge           Artifact // NGN-written Windows affinity/priority bridge
	MoveOverhead            int
	MaxLoad                 float64
}
type Event struct {
	AtNS         int64
	Stream, Line string
}
type SessionRecord struct {
	Engine     Artifact
	Events     []Event
	Options    map[string]string
	Selected   map[string]string
	PID        int
	ExitCode   int
	LaunchArgv []string
	Scheduling string
}
type Ply struct {
	Move, STM, Role    string
	Before, After      [2]int64 // white, black, nanoseconds
	UsedNS             int64
	GoEvent, BestEvent int
}
type Game struct {
	Schema, ID, PairID, Anchor, CandidateColor string
	AttemptID, StartedUTC, EndedUTC            string
	Opening                                    Opening
	BaseNS, IncrementNS                        int64
	Cap                                        int
	Candidate, Opponent                        SessionRecord
	Plies                                      []Ply
	Result, Reason, Error                      string // result is white POV
}
type Counts struct{ W, D, L, N, Pairs, CapDraws int }
type Summary struct {
	Schema string
	Counts map[string]Counts
	Games  int
}
type Audit struct {
	Schema, ReceiptSHA256 string
	Games                 map[string]string
	Counts                map[string]Counts
	Admitted              bool
	Exclusions            int
}

func Digest(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
func Bind(path string) (Artifact, error) {
	p, e := filepath.Abs(path)
	if e != nil {
		return Artifact{}, e
	}
	h, e := Digest(p)
	return Artifact{p, h}, e
}
func Check(a Artifact) error {
	if a.Path == "" || len(a.SHA256) != 64 {
		return fmt.Errorf("unresolved artifact: %v", a)
	}
	h, e := Digest(a.Path)
	if e != nil {
		return e
	}
	if h != a.SHA256 {
		return fmt.Errorf("hash mismatch: %s: %s != %s", a.Path, h, a.SHA256)
	}
	return nil
}
func ReadJSON(path string, v any) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if e = d.Decode(v); e != nil {
		return e
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return fmt.Errorf("extra JSON in %s", path)
	}
	return nil
}

// Exclusive creation prevents overwriting/restarting a frozen attempt.
func WriteJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	b = append(b, '\n')
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0444)
	if e != nil {
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	return f.Close()
}
