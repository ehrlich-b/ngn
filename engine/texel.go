package engine

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
)

// Texel Tuning - Automatic evaluation parameter tuning
// Uses gradient descent to minimize prediction error on game positions

// TuningPosition represents a position with known game result
type TuningPosition struct {
	FEN    string
	Result float64 // 1.0 = white win, 0.5 = draw, 0.0 = black win
}

// TuningParam represents a tunable parameter
type TuningParam struct {
	Name     string
	Value    *int
	Min, Max int
}

// TexelTuner holds the tuning state
type TexelTuner struct {
	positions []TuningPosition
	params    []TuningParam
	K         float64 // Scaling constant
}

// NewTexelTuner creates a new tuner instance
func NewTexelTuner() *TexelTuner {
	return &TexelTuner{
		K: 1.13, // Typical scaling constant for centipawn evaluation
	}
}

// LoadPositions loads positions from an EPD file
// EPD format: FEN c9 "1-0"; or c9 "0-1"; or c9 "1/2-1/2";
func (t *TexelTuner) LoadPositions(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		pos, err := parseEPDLine(line)
		if err != nil {
			continue // Skip invalid lines
		}
		t.positions = append(t.positions, pos)
	}

	return scanner.Err()
}

// parseEPDLine parses an EPD line into a TuningPosition
func parseEPDLine(line string) (TuningPosition, error) {
	// Split on common result markers
	var fen string
	var result float64

	if strings.Contains(line, "1-0") {
		result = 1.0
		fen = strings.Split(line, " c9")[0]
		if fen == line {
			fen = strings.Split(line, " 1-0")[0]
		}
	} else if strings.Contains(line, "0-1") {
		result = 0.0
		fen = strings.Split(line, " c9")[0]
		if fen == line {
			fen = strings.Split(line, " 0-1")[0]
		}
	} else if strings.Contains(line, "1/2-1/2") {
		result = 0.5
		fen = strings.Split(line, " c9")[0]
		if fen == line {
			fen = strings.Split(line, " 1/2-1/2")[0]
		}
	} else {
		return TuningPosition{}, fmt.Errorf("no result found in line")
	}

	fen = strings.TrimSpace(fen)

	// Validate FEN by trying to parse it
	_, err := ParseFEN(fen)
	if err != nil {
		return TuningPosition{}, err
	}

	return TuningPosition{FEN: fen, Result: result}, nil
}

// AddPositionsFromPGN loads positions from a PGN file (simplified)
func (t *TexelTuner) AddPositionsFromPGN(filename string) error {
	// For simplicity, this just loads from EPD format
	// A full implementation would parse PGN and extract positions
	return t.LoadPositions(filename)
}

// Sigmoid converts centipawn evaluation to win probability
func (t *TexelTuner) Sigmoid(eval float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, -t.K*eval/400.0))
}

// MeanSquaredError calculates the mean squared error across all positions
func (t *TexelTuner) MeanSquaredError() float64 {
	if len(t.positions) == 0 {
		return 0
	}

	totalError := 0.0
	for _, pos := range t.positions {
		// Parse the position
		position, err := ParseFEN(pos.FEN)
		if err != nil {
			continue
		}

		// Get static evaluation (from white's perspective)
		eval := float64(Evaluate(&position.Board))
		if position.Turn() == Black {
			eval = -eval
		}

		// Calculate predicted win probability
		predicted := t.Sigmoid(eval)

		// Calculate squared error
		err2 := (predicted - pos.Result) * (predicted - pos.Result)
		totalError += err2
	}

	return totalError / float64(len(t.positions))
}

// FindOptimalK finds the optimal K value to minimize MSE
func (t *TexelTuner) FindOptimalK() float64 {
	bestK := 1.0
	bestMSE := math.MaxFloat64

	for k := 0.5; k <= 2.0; k += 0.01 {
		t.K = k
		mse := t.MeanSquaredError()
		if mse < bestMSE {
			bestMSE = mse
			bestK = k
		}
	}

	t.K = bestK
	return bestK
}

// TuneParameters performs gradient descent on evaluation parameters
func (t *TexelTuner) TuneParameters(iterations int, learningRate float64) {
	for iter := 0; iter < iterations; iter++ {
		for i := range t.params {
			// Calculate gradient numerically
			originalValue := *t.params[i].Value

			// Try +1
			*t.params[i].Value = min(originalValue+1, t.params[i].Max)
			msePlus := t.MeanSquaredError()

			// Try -1
			*t.params[i].Value = max(originalValue-1, t.params[i].Min)
			mseMinus := t.MeanSquaredError()

			// Restore original
			*t.params[i].Value = originalValue

			// Calculate gradient
			gradient := (msePlus - mseMinus) / 2.0

			// Update parameter
			newValue := originalValue - int(learningRate*gradient*1000)
			newValue = max(t.params[i].Min, min(t.params[i].Max, newValue))
			*t.params[i].Value = newValue
		}

		if iter%10 == 0 {
			fmt.Printf("Iteration %d: MSE = %.6f\n", iter, t.MeanSquaredError())
		}
	}
}

// RegisterCurrentParams registers the current evaluation parameters for tuning
func (t *TexelTuner) RegisterCurrentParams() {
	// These would be pointers to actual evaluation constants
	// For now, we define a set of tunable parameters

	// Note: In a real implementation, these would point to mutable global variables
	// For this framework, we demonstrate the structure
}

// PrintParams prints current parameter values
func (t *TexelTuner) PrintParams() {
	fmt.Println("Current Parameter Values:")
	for _, p := range t.params {
		fmt.Printf("  %s = %d\n", p.Name, *p.Value)
	}
}

// GenerateTuningPositions generates positions for tuning from a starting position
func GenerateTuningPositions(numPositions int) []TuningPosition {
	var positions []TuningPosition

	// Generate positions by playing random games
	for i := 0; i < numPositions/50; i++ {
		pos, _ := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
		var result float64

		// Play a random game
		for moves := 0; moves < 200; moves++ {
			legalMoves := GenerateLegalMoves(pos)
			if len(legalMoves) == 0 {
				// Game over
				if pos.IsInCheck() {
					if pos.Turn() == White {
						result = 0.0 // Black wins
					} else {
						result = 1.0 // White wins
					}
				} else {
					result = 0.5 // Stalemate
				}
				break
			}

			// Make a "reasonably good" move using search
			info := SearchFixed(pos, 2, nil)
			if info.BestMove == EmptyMove {
				break
			}

			// Store position before move (skip opening)
			if moves > 10 {
				fen := GenerateFEN(pos)
				positions = append(positions, TuningPosition{
					FEN:    fen,
					Result: -1, // Placeholder, will be filled after game
				})
			}

			pos.MakeMove(info.BestMove)

			// Check for draw by repetition/50 move
			if pos.HalfMoveClock >= 100 {
				result = 0.5
				break
			}
		}

		// Update all positions from this game with the result
		startIdx := len(positions) - 189 // Approximate positions per game
		if startIdx < 0 {
			startIdx = 0
		}
		for j := startIdx; j < len(positions); j++ {
			if positions[j].Result == -1 {
				positions[j].Result = result
			}
		}
	}

	return positions
}

// QuickTune performs a quick tune using built-in positions
func (t *TexelTuner) QuickTune() {
	// Generate some test positions
	fmt.Println("Generating test positions...")
	t.positions = GenerateTuningPositions(1000)
	fmt.Printf("Generated %d positions\n", len(t.positions))

	// Find optimal K
	fmt.Println("Finding optimal K...")
	optK := t.FindOptimalK()
	fmt.Printf("Optimal K = %.3f\n", optK)

	// Show initial MSE
	fmt.Printf("Initial MSE = %.6f\n", t.MeanSquaredError())
}
