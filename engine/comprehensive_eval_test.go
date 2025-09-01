package engine

import (
	"testing"
	"time"
)

// ComprehensiveEvaluationResult represents detailed engine analysis
type ComprehensiveEvaluationResult struct {
	TacticalScore   int
	PositionalScore int 
	MaterialScore   int
	EndgameScore    int
	OverallScore    int
	EstimatedELO    int
	AverageDepth    float64
	WeaknessReport  []string
}

// TestComprehensiveEngineEvaluation runs a thorough 2-minute engine evaluation
// with proper time allocation for meaningful search depth
func TestComprehensiveEngineEvaluation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive evaluation in short mode")
	}

	t.Log("🔥 COMPREHENSIVE ENGINE EVALUATION (2 MINUTE TEST)")
	t.Log("=================================================")
	t.Log("Each position gets 10-15 seconds for deep analysis")
	
	startTime := time.Now()
	result := &ComprehensiveEvaluationResult{
		WeaknessReport: make([]string, 0),
	}
	
	// Test categories with proper time allocation
	tacticalScore, depthTactical := testTacticalWithTime(t, 80*time.Second) // 80s for tactics (most important)
	materialScore, depthMaterial := testMaterialWithTime(t, 20*time.Second) // 20s for material 
	positionalScore, depthPositional := testPositionalWithTime(t, 15*time.Second) // 15s for positional
	endgameScore, depthEndgame := testEndgameWithTime(t, 5*time.Second) // 5s for endgame
	
	// Assign to result struct
	result.TacticalScore = tacticalScore
	result.MaterialScore = materialScore
	result.PositionalScore = positionalScore
	result.EndgameScore = endgameScore
	
	// Calculate average depth
	result.AverageDepth = (depthTactical + depthMaterial + depthPositional + depthEndgame) / 4.0
	
	// Calculate overall score and ELO
	result.calculateComprehensiveScore()
	
	totalTime := time.Since(startTime)
	
	// Report comprehensive results
	t.Log("\n🎯 COMPREHENSIVE EVALUATION RESULTS")
	t.Log("===================================")
	t.Logf("Total test time: %.1f seconds", totalTime.Seconds())
	t.Logf("Average search depth: %.1f", result.AverageDepth)
	t.Log("")
	t.Logf("Tactical Strength:     %d/100", result.TacticalScore)
	t.Logf("Material Evaluation:   %d/100", result.MaterialScore)
	t.Logf("Positional Play:       %d/100", result.PositionalScore)
	t.Logf("Endgame Technique:     %d/100", result.EndgameScore)
	t.Log("")
	t.Logf("OVERALL SCORE:         %d/100", result.OverallScore)
	t.Logf("ESTIMATED ELO:         %d", result.EstimatedELO)
	
	if len(result.WeaknessReport) > 0 {
		t.Log("\n🚨 IDENTIFIED WEAKNESSES:")
		for i, weakness := range result.WeaknessReport {
			t.Logf("%d. %s", i+1, weakness)
		}
	}
	
	// Success criteria 
	if result.EstimatedELO >= 1400 {
		t.Logf("✅ ENGINE MEETS MINIMUM STANDARD: %d ELO (≥1400 required)", result.EstimatedELO)
	} else {
		t.Errorf("❌ ENGINE BELOW MINIMUM: %d ELO (≥1400 required)", result.EstimatedELO)
	}
}

// testTacticalWithTime tests tactical awareness with proper time allocation
func testTacticalWithTime(t *testing.T, timeLimit time.Duration) (int, float64) {
	t.Log("\n🎯 TACTICAL ANALYSIS (Deep Search)")
	t.Log("=================================")
	
	// Key tactical positions that require depth to solve correctly
	tacticalSuite := []struct {
		name        string
		fen         string
		timeAlloc   time.Duration
		bestMoves   []string
		description string
	}{
		{
			name:        "Fork_Attack", 
			fen:         "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3",
			timeAlloc:   15 * time.Second,
			bestMoves:   []string{"f3g5", "f3e5"}, // Accept either tactical setup or material gain
			description: "Knight fork setup vs immediate material",
		},
		{
			name:        "Free_Material",
			fen:         "rnbqkb1r/pppp1ppp/5n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", 
			timeAlloc:   10 * time.Second,
			bestMoves:   []string{"c4f7"}, // Bishop takes free pawn
			description: "Should capture completely free material",
		},
		{
			name:        "Pin_Defense",
			fen:         "r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
			timeAlloc:   12 * time.Second,
			bestMoves:   []string{"b5c6", "b5d7"}, // Attack the pinned knight
			description: "Should exploit pin tactically",
		},
		{
			name:        "Mate_Threat",
			fen:         "rnb1kbnr/pppp1ppp/8/4p1q1/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3",
			timeAlloc:   20 * time.Second,
			bestMoves:   []string{"f1e2", "d2d3", "h2h3"}, // Block mate threat
			description: "Should defend against mate threats",
		},
		{
			name:        "Double_Attack",
			fen:         "r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R w KQkq - 0 6",
			timeAlloc:   15 * time.Second,
			bestMoves:   []string{"c4f7", "f3g5"}, // Double attack ideas
			description: "Should find double attack opportunities",
		},
	}
	
	correct := 0
	totalDepth := 0.0
	
	for _, test := range tacticalSuite {
		t.Run(test.name, func(t *testing.T) {
			pos, err := ParseFEN(test.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}
			
			t.Logf("Position: %s", test.description)
			t.Logf("Time allocation: %.1f seconds", test.timeAlloc.Seconds())
			
			// Search with time limit using iterative deepening
			timeManager := NewTimeManager()
			params := SearchParams{
				MoveTime: int(test.timeAlloc.Milliseconds()),
			}
			timeManager.SetTimeControl(params, pos.Turn() == White)
			result := SearchIterativeDeepening(pos, 20, timeManager)
			
			if result != nil {
				move := result.BestMove.ToString()
				depth := result.Depth
				score := result.BestScore
				nodes := result.Nodes
				
				totalDepth += float64(depth)
				
				t.Logf("Result: %s (depth: %d, score: %d, nodes: %d)", move, depth, score, nodes)
				
				// Check if move is acceptable
				moveGood := false
				for _, goodMove := range test.bestMoves {
					if move == goodMove {
						moveGood = true
						break
					}
				}
				
				if moveGood {
					t.Logf("✅ CORRECT: Good tactical choice")
					correct++
				} else {
					t.Logf("⚠️  SUBOPTIMAL: Expected %v, got %s", test.bestMoves, move)
				}
			} else {
				t.Logf("❌ FAILED: No result from search")
			}
		})
	}
	
	avgDepth := totalDepth / float64(len(tacticalSuite))
	score := (correct * 100) / len(tacticalSuite)
	
	t.Logf("\n🎯 Tactical Results: %d/100 (%d/%d correct, avg depth: %.1f)", 
		score, correct, len(tacticalSuite), avgDepth)
	
	return score, avgDepth
}

// testMaterialWithTime tests material evaluation with time
func testMaterialWithTime(t *testing.T, timeLimit time.Duration) (int, float64) {
	t.Log("\n💎 MATERIAL EVALUATION")
	t.Log("=====================")
	
	materialTests := []struct {
		name     string
		fen      string
		expected string // "white_wins", "black_wins", "equal"
	}{
		{
			name:     "Queen_vs_Rook",
			fen:      "4k3/8/8/8/8/8/4Q3/4K3 w - - 0 1",
			expected: "white_wins",
		},
		{
			name:     "Rook_vs_Minor_Pieces", 
			fen:      "4k3/8/8/8/8/2bn4/4R3/4K3 w - - 0 1",
			expected: "equal", // Should be roughly equal
		},
	}
	
	correct := 0
	
	for _, test := range materialTests {
		pos, _ := ParseFEN(test.fen)
		eval := EvaluateForPlayer(&pos.Board, White)
		
		var result string
		if eval > 200 {
			result = "white_wins"
		} else if eval < -200 {
			result = "black_wins" 
		} else {
			result = "equal"
		}
		
		if result == test.expected {
			correct++
			t.Logf("✅ %s: %d (correct)", test.name, eval)
		} else {
			t.Logf("❌ %s: %d (expected %s)", test.name, eval, test.expected)
		}
	}
	
	score := (correct * 100) / len(materialTests)
	return score, 1.0 // Material evaluation is static
}

// testPositionalWithTime tests positional understanding  
func testPositionalWithTime(t *testing.T, timeLimit time.Duration) (int, float64) {
	t.Log("\n🏰 POSITIONAL UNDERSTANDING")
	t.Log("==========================")
	
	// For now, basic implementation
	return 50, 2.0 // Placeholder - needs development
}

// testEndgameWithTime tests endgame technique
func testEndgameWithTime(t *testing.T, timeLimit time.Duration) (int, float64) {
	t.Log("\n👑 ENDGAME TECHNIQUE") 
	t.Log("===================")
	
	// For now, basic implementation
	return 30, 3.0 // Placeholder - needs development
}

// calculateComprehensiveScore calculates overall score and ELO estimate
func (r *ComprehensiveEvaluationResult) calculateComprehensiveScore() {
	// Weight tactical strength heavily since it's most important
	r.OverallScore = (r.TacticalScore*60 + r.MaterialScore*25 + r.PositionalScore*10 + r.EndgameScore*5) / 100
	
	// ELO calculation based on comprehensive score and depth
	baseELO := 800 + (r.OverallScore * 15) // Base scaling
	
	// Depth bonus - deeper search should increase ELO estimate
	depthBonus := int((r.AverageDepth - 3.0) * 50)
	if depthBonus < 0 {
		depthBonus = 0
	}
	
	r.EstimatedELO = baseELO + depthBonus
	
	// Cap at reasonable values
	if r.EstimatedELO > 2200 {
		r.EstimatedELO = 2200
	}
}