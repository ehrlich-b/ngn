package sf18big

import (
	"fmt"

	base "github.com/ehrlich-b/ngn/nnue"
)

// Components are the Stockfish BIG network's side-to-move PSQT and positional
// outputs after their separate division by OutputScale. Search score policy is
// intentionally outside this package.
type Components struct {
	PSQT       int32
	Positional int32
}

// SelectedEvaluation is the trace-free result for the material-selected BIG
// layer stack.
type SelectedEvaluation struct {
	Bucket uint8
	Components
}

// ReferenceState retains the separate base and FullThreats accumulators needed
// to measure simple sorted-set updates without introducing a search context.
type ReferenceState struct {
	BaseValues   [2][transformerLanes]int16
	ThreatValues [2][transformerLanes]int16
	BasePSQT     [2][psqtBuckets]int32
	ThreatPSQT   [2][psqtBuckets]int32
	Threats      [2]ThreatIndexList
}

// EvaluateSelected performs an exact scalar full refresh and propagates only
// the material-selected layer stack. It is a bounded N1b reference path, not a
// search evaluator.
func (m *Model) EvaluateSelected(position base.Position) (SelectedEvaluation, error) {
	state, occupied, err := m.ReferenceRefresh(position)
	if err != nil {
		return SelectedEvaluation{}, err
	}
	return m.evaluateSelectedState(&state, position.SideToMove, occupied), nil
}

// ReferenceRefresh builds the separate base and threat accumulator states for
// both perspectives using logical-order tensors.
func (m *Model) ReferenceRefresh(position base.Position) (ReferenceState, int, error) {
	var state ReferenceState
	if m == nil || !m.loaded {
		return state, 0, fmt.Errorf("%w: model was not returned by Load", ErrEvaluation)
	}
	board, kings, _, occupied, err := validatePosition(position)
	if err != nil {
		return state, 0, err
	}
	for perspective := base.White; perspective <= base.Black; perspective++ {
		p := int(perspective)
		copy(state.BaseValues[p][:], m.featureBias[:])
		for square, piece := range board {
			if !piece.set {
				continue
			}
			feature := baseFeatureIndex(piece, base.Square(square), kings[p], perspective)
			m.updateBaseFeature(&state.BaseValues[p], &state.BasePSQT[p], feature, true)
		}

		active, err := ActiveThreats(position, perspective)
		if err != nil {
			return ReferenceState{}, 0, err
		}
		state.Threats[p] = active
		for _, feature := range active.Indices[:active.Count] {
			weightOffset := int(feature) * transformerLanes
			for lane := 0; lane < transformerLanes; lane++ {
				state.ThreatValues[p][lane] = wrapAdd16(
					state.ThreatValues[p][lane],
					int16(m.threatWeights[weightOffset+lane]),
				)
			}
			psqtOffset := int(feature) * psqtBuckets
			for bucket := 0; bucket < psqtBuckets; bucket++ {
				state.ThreatPSQT[p][bucket] = wrapAdd32(
					state.ThreatPSQT[p][bucket],
					m.threatPSQTWeights[psqtOffset+bucket],
				)
			}
		}
	}
	return state, occupied, nil
}

func (m *Model) updateBaseFeature(
	values *[transformerLanes]int16,
	psqt *[psqtBuckets]int32,
	feature int,
	add bool,
) {
	weightOffset := feature * transformerLanes
	for lane := 0; lane < transformerLanes; lane++ {
		weight := m.baseWeights[weightOffset+lane]
		if add {
			values[lane] = wrapAdd16(values[lane], weight)
		} else {
			values[lane] = wrapSub16(values[lane], weight)
		}
	}
	psqtOffset := feature * psqtBuckets
	for bucket := 0; bucket < psqtBuckets; bucket++ {
		weight := m.basePSQTWeights[psqtOffset+bucket]
		if add {
			psqt[bucket] = wrapAdd32(psqt[bucket], weight)
		} else {
			psqt[bucket] = wrapSub32(psqt[bucket], weight)
		}
	}
}

// ApplyThreatDiff applies a non-refresh sorted-set threat transition to one
// perspective. Callers must refresh instead when diff.RequiresRefresh is true.
func (m *Model) ApplyThreatDiff(state *ReferenceState, perspective base.Color, diff ThreatDiff) error {
	if m == nil || !m.loaded || state == nil {
		return fmt.Errorf("%w: unavailable model or state", ErrEvaluation)
	}
	if perspective > base.Black {
		return fmt.Errorf("%w: perspective %d", ErrEvaluation, perspective)
	}
	if diff.RequiresRefresh {
		return fmt.Errorf("%w: threat transition requires refresh", ErrEvaluation)
	}
	p := int(perspective)
	for _, feature := range diff.Removed.Indices[:diff.Removed.Count] {
		if feature >= threatInputFeatures {
			return fmt.Errorf("%w: removed threat index %d", ErrEvaluation, feature)
		}
		m.updateThreatFeature(&state.ThreatValues[p], &state.ThreatPSQT[p], feature, false)
	}
	for _, feature := range diff.Added.Indices[:diff.Added.Count] {
		if feature >= threatInputFeatures {
			return fmt.Errorf("%w: added threat index %d", ErrEvaluation, feature)
		}
		m.updateThreatFeature(&state.ThreatValues[p], &state.ThreatPSQT[p], feature, true)
	}
	state.Threats[p] = applyThreatIndexDiff(state.Threats[p], diff)
	return nil
}

func (m *Model) updateThreatFeature(
	values *[transformerLanes]int16,
	psqt *[psqtBuckets]int32,
	feature uint32,
	add bool,
) {
	weightOffset := int(feature) * transformerLanes
	for lane := 0; lane < transformerLanes; lane++ {
		weight := int16(m.threatWeights[weightOffset+lane])
		if add {
			values[lane] = wrapAdd16(values[lane], weight)
		} else {
			values[lane] = wrapSub16(values[lane], weight)
		}
	}
	psqtOffset := int(feature) * psqtBuckets
	for bucket := 0; bucket < psqtBuckets; bucket++ {
		weight := m.threatPSQTWeights[psqtOffset+bucket]
		if add {
			psqt[bucket] = wrapAdd32(psqt[bucket], weight)
		} else {
			psqt[bucket] = wrapSub32(psqt[bucket], weight)
		}
	}
}

func applyThreatIndexDiff(before ThreatIndexList, diff ThreatDiff) ThreatIndexList {
	var after ThreatIndexList
	i, removed, added := 0, 0, 0
	for i < int(before.Count) || added < int(diff.Added.Count) {
		for removed < int(diff.Removed.Count) && i < int(before.Count) &&
			diff.Removed.Indices[removed] == before.Indices[i] {
			i++
			removed++
		}
		if i == int(before.Count) {
			for added < int(diff.Added.Count) {
				after.Indices[after.Count] = diff.Added.Indices[added]
				after.Count++
				added++
			}
			break
		}
		if added < int(diff.Added.Count) && diff.Added.Indices[added] < before.Indices[i] {
			after.Indices[after.Count] = diff.Added.Indices[added]
			after.Count++
			added++
			continue
		}
		after.Indices[after.Count] = before.Indices[i]
		after.Count++
		i++
	}
	return after
}

func (m *Model) evaluateSelectedState(state *ReferenceState, sideToMove base.Color, occupied int) SelectedEvaluation {
	bucket := (occupied - 1) / 4
	transformed, psqtRaw := transformReferenceState(state, sideToMove, bucket)
	positionalRaw := m.propagateSelectedStack(bucket, &transformed)
	return SelectedEvaluation{
		Bucket: uint8(bucket),
		Components: Components{
			PSQT:       psqtRaw / 16,
			Positional: positionalRaw / 16,
		},
	}
}

func transformReferenceState(state *ReferenceState, sideToMove base.Color, bucket int) ([transformerLanes]uint8, int32) {
	var transformed [transformerLanes]uint8
	perspectives := [2]base.Color{sideToMove, sideToMove ^ 1}
	for half, perspective := range perspectives {
		p := int(perspective)
		for lane := 0; lane < transformerLanes/2; lane++ {
			a := clampInt16(wrapAdd16(state.BaseValues[p][lane], state.ThreatValues[p][lane]), 0, 255)
			b := clampInt16(
				wrapAdd16(state.BaseValues[p][lane+transformerLanes/2], state.ThreatValues[p][lane+transformerLanes/2]),
				0,
				255,
			)
			transformed[half*transformerLanes/2+lane] = uint8((uint32(a) * uint32(b)) / 512)
		}
	}

	stm, ntm := int(sideToMove), int(sideToMove^1)
	baseDifference := wrapSub32(state.BasePSQT[stm][bucket], state.BasePSQT[ntm][bucket])
	threatDifference := wrapSub32(state.ThreatPSQT[stm][bucket], state.ThreatPSQT[ntm][bucket])
	psqtRaw := wrapAdd32(baseDifference, threatDifference) / 2
	return transformed, psqtRaw
}

var kingBucket = [64]uint8{
	28, 29, 30, 31, 31, 30, 29, 28,
	24, 25, 26, 27, 27, 26, 25, 24,
	20, 21, 22, 23, 23, 22, 21, 20,
	16, 17, 18, 19, 19, 18, 17, 16,
	12, 13, 14, 15, 15, 14, 13, 12,
	8, 9, 10, 11, 11, 10, 9, 8,
	4, 5, 6, 7, 7, 6, 5, 4,
	0, 1, 2, 3, 3, 2, 1, 0,
}

func baseFeatureIndex(piece boardPiece, square, king base.Square, perspective base.Color) int {
	flip := base.Square(56 * perspective)
	orient := base.Square(0)
	if king&7 < 4 {
		orient = 7
	}
	plane := 10
	if piece.piece != base.King {
		plane = 2*int(piece.piece) + int(piece.color^perspective)
	}
	return int(square^orient^flip) + plane*64 + int(kingBucket[king^flip])*704
}

func (m *Model) propagateSelectedStack(bucket int, input *[transformerLanes]uint8) int32 {
	stack := &m.stacks[bucket]
	var fc0 [16]int32
	for output := range fc0 {
		fc0[output] = affineRowValue(
			input[:],
			stack.fc0Weight[output*transformerLanes:(output+1)*transformerLanes],
			stack.fc0Bias[output],
		)
	}

	var fc1Input [32]uint8
	for index := 0; index < 15; index++ {
		fc1Input[index] = squaredClippedReLU(fc0[index])
		fc1Input[index+15] = clippedReLU(fc0[index])
	}
	var clipped1 [32]uint8
	for output := range clipped1 {
		value := affineRowValue(
			fc1Input[:],
			stack.fc1Weight[output*32:(output+1)*32],
			stack.fc1Bias[output],
		)
		clipped1[output] = clippedReLU(value)
	}
	fc2 := affineRowValue(clipped1[:], stack.fc2Weight[:], stack.fc2Bias[0])
	forward := wrapMul32(fc0[15], 9600) / 8128
	return wrapAdd32(fc2, forward)
}

func affineRowValue(input []uint8, weights []int8, bias int32) int32 {
	value := bias
	for index, inputValue := range input {
		value = wrapAdd32(value, int32(inputValue)*int32(weights[index]))
	}
	return value
}

func squaredClippedReLU(value int32) uint8 {
	squared := (int64(value) * int64(value)) >> 19
	if squared > 127 {
		return 127
	}
	return uint8(squared)
}

func clippedReLU(value int32) uint8 {
	value >>= 6
	if value < 0 {
		return 0
	}
	if value > 127 {
		return 127
	}
	return uint8(value)
}

func clampInt16(value, minimum, maximum int16) int16 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func wrapAdd16(left, right int16) int16 { return int16(uint16(left) + uint16(right)) }
func wrapSub16(left, right int16) int16 { return int16(uint16(left) - uint16(right)) }
func wrapAdd32(left, right int32) int32 { return int32(uint32(left) + uint32(right)) }
func wrapSub32(left, right int32) int32 { return int32(uint32(left) - uint32(right)) }
func wrapMul32(left, right int32) int32 { return int32(uint32(left) * uint32(right)) }
