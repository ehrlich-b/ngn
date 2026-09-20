package sf18big

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	base "github.com/ehrlich-b/ngn/nnue"
)

var (
	benchmarkThreatListSink  ThreatIndexList
	benchmarkStateSink       ReferenceState
	benchmarkTransformedSink [transformerLanes]uint8
	benchmarkInt32Sink       int32
	benchmarkSelectedSink    SelectedEvaluation
)

type n1bBenchmarkFixture struct {
	model        *Model
	positions    []base.Position
	states       []ReferenceState
	buckets      []int
	transformed  [][transformerLanes]uint8
	features     []uint32
	baseFeatures []int
	transitions  []threatListPair
}

type threatListPair struct {
	before ThreatIndexList
	after  ThreatIndexList
}

func loadN1bBenchmarkFixture(b *testing.B) n1bBenchmarkFixture {
	b.Helper()
	networkPath := os.Getenv(officialFileEnvironment)
	oraclePath := os.Getenv(referenceGameOracleEnvironment)
	if networkPath == "" || oraclePath == "" {
		b.Skip("official BIG network and representative oracle are required")
	}
	networkFile, err := os.Open(networkPath)
	if err != nil {
		b.Fatal(err)
	}
	model, err := Load(networkFile)
	networkFile.Close()
	if err != nil {
		b.Fatal(err)
	}
	oracleFile, err := os.Open(oraclePath)
	if err != nil {
		b.Fatal(err)
	}
	defer oracleFile.Close()
	var fixture n1bBenchmarkFixture
	fixture.model = model
	previousCase := ""
	var previousState ReferenceState
	scanner := bufio.NewScanner(oracleFile)
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		var row referenceOracleRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			b.Fatal(err)
		}
		position := base.Position{SideToMove: row.State.SideToMove, Pieces: row.State.Pieces}
		state, occupied, err := model.ReferenceRefresh(position)
		if err != nil {
			b.Fatal(err)
		}
		bucket := (occupied - 1) / 4
		transformed, _ := transformReferenceState(&state, position.SideToMove, bucket)
		fixture.positions = append(fixture.positions, position)
		fixture.states = append(fixture.states, state)
		fixture.buckets = append(fixture.buckets, bucket)
		fixture.transformed = append(fixture.transformed, transformed)
		board, kings, _, _, err := validatePosition(position)
		if err != nil {
			b.Fatal(err)
		}
		for perspective := base.White; perspective <= base.Black; perspective++ {
			for square, piece := range board {
				if piece.set {
					fixture.baseFeatures = append(
						fixture.baseFeatures,
						baseFeatureIndex(piece, base.Square(square), kings[perspective], perspective),
					)
				}
			}
		}
		if row.Case == previousCase {
			for perspective := base.White; perspective <= base.Black; perspective++ {
				fixture.transitions = append(fixture.transitions, threatListPair{
					before: previousState.Threats[perspective],
					after:  state.Threats[perspective],
				})
			}
		}
		for perspective := base.White; perspective <= base.Black; perspective++ {
			fixture.features = append(
				fixture.features,
				state.Threats[perspective].Indices[:state.Threats[perspective].Count]...,
			)
		}
		previousCase = row.Case
		previousState = state
	}
	if err := scanner.Err(); err != nil {
		b.Fatal(err)
	}
	if len(fixture.positions) == 0 || len(fixture.features) == 0 ||
		len(fixture.baseFeatures) == 0 || len(fixture.transitions) == 0 {
		b.Fatal("empty representative benchmark fixture")
	}
	return fixture
}

func BenchmarkN1bReference(b *testing.B) {
	fixture := loadN1bBenchmarkFixture(b)

	b.Run("threat-enumeration", func(b *testing.B) {
		b.ReportAllocs()
		var result ThreatIndexList
		for i := 0; i < b.N; i++ {
			position := fixture.positions[i%len(fixture.positions)]
			var err error
			result, err = ActiveThreats(position, base.Color(i&1))
			if err != nil {
				b.Fatal(err)
			}
		}
		benchmarkThreatListSink = result
	})

	b.Run("sorted-threat-diff", func(b *testing.B) {
		b.ReportAllocs()
		var removed, added ThreatIndexList
		for i := 0; i < b.N; i++ {
			transition := fixture.transitions[i%len(fixture.transitions)]
			removed, added = diffThreatIndexLists(transition.before, transition.after)
		}
		benchmarkThreatListSink = removed
		benchmarkThreatListSink = added
	})

	b.Run("threat-row-update-1024", func(b *testing.B) {
		b.ReportAllocs()
		var values [transformerLanes]int16
		var psqt [psqtBuckets]int32
		for i := 0; i < b.N; i++ {
			fixture.model.updateThreatFeature(
				&values,
				&psqt,
				fixture.features[i%len(fixture.features)],
				i&1 == 0,
			)
		}
		benchmarkInt32Sink = int32(values[0]) + psqt[0]
	})

	b.Run("base-row-update-1024", func(b *testing.B) {
		b.ReportAllocs()
		var values [transformerLanes]int16
		var psqt [psqtBuckets]int32
		for i := 0; i < b.N; i++ {
			fixture.model.updateBaseFeature(
				&values,
				&psqt,
				fixture.baseFeatures[i%len(fixture.baseFeatures)],
				i&1 == 0,
			)
		}
		benchmarkInt32Sink = int32(values[0]) + psqt[0]
	})

	b.Run("transform-1024", func(b *testing.B) {
		b.ReportAllocs()
		var transformed [transformerLanes]uint8
		for i := 0; i < b.N; i++ {
			index := i % len(fixture.states)
			transformed, benchmarkInt32Sink = transformReferenceState(
				&fixture.states[index],
				fixture.positions[index].SideToMove,
				fixture.buckets[index],
			)
		}
		benchmarkTransformedSink = transformed
	})

	b.Run("selected-head-1024", func(b *testing.B) {
		b.ReportAllocs()
		var result int32
		for i := 0; i < b.N; i++ {
			index := i % len(fixture.transformed)
			result = fixture.model.propagateSelectedStack(
				fixture.buckets[index],
				&fixture.transformed[index],
			)
		}
		benchmarkInt32Sink = result
	})

	b.Run("full-refresh", func(b *testing.B) {
		b.ReportAllocs()
		var state ReferenceState
		for i := 0; i < b.N; i++ {
			var err error
			state, _, err = fixture.model.ReferenceRefresh(fixture.positions[i%len(fixture.positions)])
			if err != nil {
				b.Fatal(err)
			}
		}
		benchmarkStateSink = state
	})

	b.Run("selected-full-refresh", func(b *testing.B) {
		b.ReportAllocs()
		var result SelectedEvaluation
		for i := 0; i < b.N; i++ {
			var err error
			result, err = fixture.model.EvaluateSelected(fixture.positions[i%len(fixture.positions)])
			if err != nil {
				b.Fatal(err)
			}
		}
		benchmarkSelectedSink = result
	})
}
