package rating

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRecordRoundTrip(t *testing.T) {
	for _, file := range []string{"testdata/counter.json", "testdata/blunder.json"} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ReadStrengthRecord(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			again, err := ReadStrengthRecord(bytes.NewReader(encoded))
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatalf("round trip: %v, %v", again, err)
			}
			if strings.Contains(file, "counter") && (got.Opponent.Name != "Counter" || got.Opponent.Version != "3.8" || got.Engine.Model.Status != "unknown") {
				t.Fatal(got)
			}
			if strings.Contains(file, "blunder") && (got.Opponent.Name != "Blunder" || got.Opponent.Version != "7.4.0") {
				t.Fatal(got)
			}
		})
	}
}
func TestRecordRejectsInvalid(t *testing.T) {
	data, err := os.ReadFile("testdata/counter.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := ReadStrengthRecord(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(*StrengthRecord)
	}{
		{"ambiguous", func(r *StrengthRecord) { r.Opponent = r.Engine }},
		{"result", func(r *StrengthRecord) { r.Result = "win" }},
		{"missing provenance", func(r *StrengthRecord) { r.Engine.Model = Provenance{} }},
		{"invalid hash", func(r *StrengthRecord) { r.Engine.Binary.Value = "abc" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := r
			c.mutate(&v)
			if v.Validate() == nil {
				t.Fatal("accepted invalid record")
			}
		})
	}
	if _, err := ReadStrengthRecord(bytes.NewReader(append(data, []byte(` {}`)...))); err == nil {
		t.Fatal("accepted trailing record")
	}
}

func TestRecordRejectsAmbiguousJSONKeys(t *testing.T) {
	data, err := os.ReadFile("testdata/counter.json")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"case variant result":   strings.Replace(string(data), `"result":`, `"Result":"win", "result":`, 1),
		"duplicate result":      strings.Replace(string(data), `"result":`, `"result":"win", "result":`, 1),
		"case variant opponent": strings.Replace(string(data), `"opponent":`, `"Opponent":{}, "opponent":`, 1),
		"duplicate opponent":    strings.Replace(string(data), `"opponent":`, `"opponent":{}, "opponent":`, 1),
		"nested case variant":   strings.Replace(string(data), `"threads":`, `"Threads":1, "threads":`, 1),
		"nested duplicate":      strings.Replace(string(data), `"status":`, `"status":"unknown", "status":`, 1),
		"escaped duplicate":     strings.Replace(string(data), `"result":`, `"res\u0075lt":"win", "result":`, 1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadStrengthRecord(strings.NewReader(input)); err == nil {
				t.Fatal("accepted ambiguous keys")
			}
		})
	}
}
