package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go \"<FEN>\" [depth]")
		fmt.Println("Example: go run main.go \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1\" 6")
		os.Exit(1)
	}

	fen := os.Args[1]
	depth := 6
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &depth)
	}

	// For now, use starting position (FEN parsing not implemented yet)
	if fen != "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" {
		log.Printf("Warning: Custom FEN not supported yet, using starting position")
	}
	
	pos := &engine.Position{
		Board:     engine.StartingBoard(),
		Tag:       engine.WhiteCanCastleKingSide | engine.WhiteCanCastleQueenSide | engine.BlackCanCastleKingSide | engine.BlackCanCastleQueenSide | engine.WhiteToMove,
		EnPassant: engine.NoSquare,
	}

	fmt.Printf("=== PRUNING EFFECTIVENESS ANALYSIS ===\n")
	fmt.Printf("Position: %s\n", fen)
	fmt.Printf("Search Depth: %d\n\n", depth)

	// Run search with statistics
	start := time.Now()
	info := engine.Search(pos, depth)
	elapsed := time.Since(start)

	// Display comprehensive statistics
	fmt.Printf("=== SEARCH RESULTS ===\n")
	fmt.Printf("Best Move: %s\n", info.BestMove)
	fmt.Printf("Score: %d centipawns\n", info.BestScore)
	fmt.Printf("Time: %v\n", elapsed)
	fmt.Printf("NPS: %d\n\n", uint64(float64(info.Nodes)/elapsed.Seconds()))

	fmt.Printf("=== NODE STATISTICS ===\n")
	fmt.Printf("Total Nodes:          %8d (100.0%%)\n", info.Nodes)
	fmt.Printf("Quiescence Nodes:     %8d (%5.1f%%)\n", info.QNodes, percentage(info.QNodes, info.Nodes))
	fmt.Printf("Main Search Nodes:    %8d (%5.1f%%)\n", info.Nodes-info.QNodes, percentage(info.Nodes-info.QNodes, info.Nodes))
	fmt.Printf("\n")

	fmt.Printf("=== PRUNING EFFECTIVENESS ===\n")
	fmt.Printf("Futility Prunes:      %8d\n", info.FutilityPrunes)
	fmt.Printf("Null Move Cutoffs:    %8d\n", info.NullMoveCutoffs)
	fmt.Printf("TT Hits:              %8d (%5.1f%%)\n", info.TTHits, percentage(info.TTHits, info.Nodes))
	fmt.Printf("TT Cutoffs:           %8d (%5.1f%%)\n", info.TTCutoffs, percentage(info.TTCutoffs, info.Nodes))
	fmt.Printf("Beta Cutoffs:         %8d (%5.1f%%)\n", info.BetaCutoffs, percentage(info.BetaCutoffs, info.Nodes))
	fmt.Printf("\n")

	fmt.Printf("=== SEARCH ENHANCEMENTS ===\n")
	fmt.Printf("LMR Reductions:       %8d\n", info.LMRReductions)
	fmt.Printf("Check Extensions:     %8d\n", info.CheckExtensions)
	fmt.Printf("\n")

	// Calculate effective branching factor
	ebf := calculateEBF(info.Nodes, depth)
	fmt.Printf("=== EFFICIENCY METRICS ===\n")
	fmt.Printf("Effective Branching Factor: %.2f\n", ebf)
	fmt.Printf("Nodes per Depth:            %.0f\n", float64(info.Nodes)/float64(depth))
	
	// Pruning efficiency ratio
	totalPrunes := info.FutilityPrunes + info.NullMoveCutoffs + info.TTCutoffs
	fmt.Printf("Total Pruning Benefit:      %8d (%5.1f%% of nodes)\n", totalPrunes, percentage(totalPrunes, info.Nodes))
	
	// Node type distribution
	fmt.Printf("\n=== NODE TYPE ANALYSIS ===\n")
	if info.BetaCutoffs > 0 {
		fmt.Printf("Cut-node ratio:       %5.1f%% (beta cutoffs)\n", percentage(info.BetaCutoffs, info.Nodes))
	}
	if info.TTHits > 0 {
		ttUtilization := float64(info.TTCutoffs) / float64(info.TTHits) * 100
		fmt.Printf("TT Utilization:       %5.1f%% (cutoffs/hits)\n", ttUtilization)
	}
}

func percentage(part, total uint64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(part) / float64(total) * 100.0
}

func calculateEBF(nodes uint64, depth int) float64 {
	if depth <= 1 {
		return float64(nodes)
	}
	// EBF = (total_nodes)^(1/depth)
	return math.Pow(float64(nodes), 1.0/float64(depth))
}