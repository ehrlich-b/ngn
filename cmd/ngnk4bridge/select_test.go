package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/ehrlich-b/ngn/nnue/ngnk4"
)

func optimizerStateFixture(tensors ...struct {
	name   string
	values []float32
}) []byte {
	var output bytes.Buffer
	var word [8]byte
	for _, tensor := range tensors {
		output.WriteString(tensor.name)
		output.WriteByte('\n')
		binary.LittleEndian.PutUint64(word[:], uint64(len(tensor.values)))
		output.Write(word[:])
		for _, value := range tensor.values {
			var encoded [4]byte
			binary.LittleEndian.PutUint32(encoded[:], math.Float32bits(value))
			output.Write(encoded[:])
		}
	}
	return output.Bytes()
}

func TestDecodeSelectionPositionUsesNormalizedSideToMoveBoard(t *testing.T) {
	var record [32]byte
	occupied := uint64(1)<<4 | uint64(1)<<8 | uint64(1)<<60
	binary.LittleEndian.PutUint64(record[:8], occupied)
	// Piece nibbles follow occupied-square order: our king, our pawn, opponent king.
	record[8] = 5 | (0 << 4)
	record[9] = 13
	storedScore := int16(-321)
	binary.LittleEndian.PutUint16(record[24:26], uint16(storedScore))
	record[26] = 1
	record[27] = 4
	record[28] = 60 ^ 56

	position, score, err := decodeSelectionPosition(record[:])
	if err != nil {
		t.Fatal(err)
	}
	if score != -321 || position.SideToMove != ngnk4.White ||
		position.Board[ngnk4.WhiteKing] != uint64(1)<<4 ||
		position.Board[ngnk4.WhitePawn] != uint64(1)<<8 ||
		position.Board[ngnk4.BlackKing] != uint64(1)<<60 {
		t.Fatalf("decoded position=%+v score=%d", position, score)
	}

	record[28]++
	if _, _, err := decodeSelectionPosition(record[:]); err == nil {
		t.Fatal("accepted mismatched cached opponent king square")
	}
}

func TestChooseSelectionCandidateUsesIntegerMSEAndEarlyStop(t *testing.T) {
	metric := func(update uint64, loss float64, eligible bool) candidateMetric {
		return candidateMetric{Update: update, IntegerMeanMSE: loss, RawMeanMSE: 1 - loss, Eligible: eligible,
			Checkpoint: selectionFileReceipt{Path: string(rune(update))}}
	}
	metrics := []candidateMetric{
		metric(0, 0.50, false),
		metric(128, 0.20, true),
		metric(256, 0.19, true),
		metric(384, 0.190_005, true),
		metric(512, 0.190_004, true),
		metric(640, 0.190_003, true),
		metric(768, 0.190_002, true),
		metric(896, 0.10, true),
		metric(1024, 0.09, true),
	}
	selected, stop, err := chooseSelectionCandidate(metrics)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Update != 256 || !stop.Triggered || stop.StopUpdate != 768 {
		t.Fatalf("selected=%d stop=%+v", selected.Update, stop)
	}
}

func TestChooseSelectionCandidateBreaksExactTieAtEarlierUpdate(t *testing.T) {
	metrics := []candidateMetric{
		{Update: 128, IntegerMeanMSE: 0.1, Eligible: true},
		{Update: 256, IntegerMeanMSE: 0.1, Eligible: true},
	}
	selected, _, err := chooseSelectionCandidate(metrics)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Update != 128 {
		t.Fatalf("selected update %d, want 128", selected.Update)
	}
}

func TestPilotS1SelectionWaitsForMinimumAndEightMisses(t *testing.T) {
	mode, err := selectionModeFor("pilot-s1")
	if err != nil {
		t.Fatal(err)
	}
	if mode.totalUpdates != 16_384 || mode.checkpointUpdates != 512 || mode.minimumUpdate != 4_096 || mode.patience != 8 {
		t.Fatalf("mode=%+v", mode)
	}
	var metrics []candidateMetric
	for update := uint64(0); update <= 16_384; update += 512 {
		loss := 0.2
		if update == 512 {
			loss = 0.1
		}
		if update > 4_608 {
			loss = 0.05 // A later win must be excluded after the declared stop.
		}
		metrics = append(metrics, candidateMetric{Update: update, IntegerMeanMSE: loss, Eligible: update > 0})
	}
	selected, stop, err := chooseSelectionCandidateForMode(metrics, mode)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Update != 512 || !stop.Triggered || stop.StopUpdate != 4_608 {
		t.Fatalf("selected=%d stop=%+v", selected.Update, stop)
	}
}

func TestProbe5MSelectionModeUsesInheritedManifestAndLongerBudget(t *testing.T) {
	mode, err := selectionModeFor("probe5m")
	if err != nil {
		t.Fatal(err)
	}
	if mode.name != "probe5m" || mode.manifestMode != "probe5m" || mode.schema != selectionSchemaProbe5M ||
		mode.totalUpdates != 32_768 || mode.checkpointUpdates != 1_024 || mode.minimumUpdate != 8_192 || mode.patience != 8 {
		t.Fatalf("mode=%+v", mode)
	}
}

func TestValidateOptimizerStateChecksLayoutAndFiniteValues(t *testing.T) {
	data := optimizerStateFixture(
		struct {
			name   string
			values []float32
		}{"l0w", []float32{1, -2}},
		struct {
			name   string
			values []float32
		}{"l0b", []float32{3}},
	)
	layout, err := validateOptimizerState(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout) != 2 || layout[0] != (optimizerTensorSpec{Name: "l0w", Count: 2}) ||
		layout[1] != (optimizerTensorSpec{Name: "l0b", Count: 1}) {
		t.Fatalf("layout=%+v", layout)
	}

	nonfinite := optimizerStateFixture(struct {
		name   string
		values []float32
	}{"l0w", []float32{float32(math.Inf(1))}})
	if _, err := validateOptimizerState(nonfinite); err == nil {
		t.Fatal("accepted nonfinite optimizer state")
	}

	duplicate := optimizerStateFixture(
		struct {
			name   string
			values []float32
		}{"l0w", []float32{1}},
		struct {
			name   string
			values []float32
		}{"l0w", []float32{2}},
	)
	if _, err := validateOptimizerState(duplicate); err == nil {
		t.Fatal("accepted duplicate optimizer tensor")
	}
}
