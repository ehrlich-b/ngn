package rating

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Provenance explicitly distinguishes a known identity from unavailable evidence.
// Value is a source revision for source, and a lowercase SHA-256 for binary/model.
type Provenance struct {
	Status string `json:"status"`
	Value  string `json:"value,omitempty"`
}
type RecordPlayer struct {
	Name    string     `json:"name"`
	Version string     `json:"version"`
	Source  Provenance `json:"source"`
	Binary  Provenance `json:"binary"`
	Model   Provenance `json:"model"`
	Threads int        `json:"threads"`
}
type HistoricalRating struct {
	System string `json:"system"`
	Value  int    `json:"value"`
}
type StrengthRecord struct {
	SchemaVersion            int               `json:"schema_version"`
	Engine                   RecordPlayer      `json:"engine"`
	Opponent                 RecordPlayer      `json:"opponent"`
	OpponentHistoricalRating *HistoricalRating `json:"opponent_historical_rating,omitempty"`
	TimeControl              string            `json:"time_control"`
	Opening                  string            `json:"opening"`
	EngineColor              string            `json:"engine_color"`
	Result                   string            `json:"result"`
	Termination              string            `json:"termination"`
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ReadStrengthRecord reads and validates exactly one v1 JSON record.
func ReadStrengthRecord(r io.Reader) (StrengthRecord, error) {
	var record StrengthRecord
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&record); err != nil {
		return record, err
	}
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return record, fmt.Errorf("multiple JSON values")
		}
		return record, err
	}
	return record, record.Validate()
}

// Validate checks the v1 identity and game contract.
func (r StrengthRecord) Validate() error {
	if r.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema version %d", r.SchemaVersion)
	}
	for _, p := range []RecordPlayer{r.Engine, r.Opponent} {
		if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Version) == "" || p.Threads < 1 {
			return fmt.Errorf("player name, version and positive threads required")
		}
		for _, item := range []struct {
			name string
			p    Provenance
			hash bool
		}{{"source", p.Source, false}, {"binary", p.Binary, true}, {"model", p.Model, true}} {
			switch item.p.Status {
			case "unknown":
				if item.p.Value != "" {
					return fmt.Errorf("unknown %s must not have a value", item.name)
				}
			case "known":
				if strings.TrimSpace(item.p.Value) == "" || (item.hash && !sha256Pattern.MatchString(item.p.Value)) {
					return fmt.Errorf("invalid known %s", item.name)
				}
			default:
				return fmt.Errorf("%s status must be known or unknown", item.name)
			}
		}
	}
	if strings.EqualFold(strings.TrimSpace(r.Engine.Name), strings.TrimSpace(r.Opponent.Name)) && strings.EqualFold(strings.TrimSpace(r.Engine.Version), strings.TrimSpace(r.Opponent.Version)) {
		return fmt.Errorf("ambiguous player identity")
	}
	if r.OpponentHistoricalRating != nil && (strings.TrimSpace(r.OpponentHistoricalRating.System) == "" || r.OpponentHistoricalRating.Value <= 0) {
		return fmt.Errorf("invalid historical rating")
	}
	if strings.TrimSpace(r.TimeControl) == "" || strings.TrimSpace(r.Opening) == "" || strings.TrimSpace(r.Termination) == "" {
		return fmt.Errorf("time control, opening and termination required")
	}
	if r.EngineColor != "white" && r.EngineColor != "black" {
		return fmt.Errorf("invalid engine color")
	}
	if r.Result != "1-0" && r.Result != "0-1" && r.Result != "1/2-1/2" {
		return fmt.Errorf("invalid result")
	}
	return nil
}
