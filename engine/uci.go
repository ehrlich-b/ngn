package engine

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// UCI Engine represents the UCI interface for the chess engine
type UCIEngine struct {
	position     *Position
	searching    bool
	stopChan     chan bool
	debugMode    bool
	engineName   string
	engineAuthor string
	version      string
}

// NewUCIEngine creates a new UCI engine instance
func NewUCIEngine() *UCIEngine {
	return &UCIEngine{
		position:     newUCIStartingPosition(),
		searching:    false,
		stopChan:     make(chan bool, 1),
		debugMode:    false,
		engineName:   "ngn",
		engineAuthor: "ngn team",
		version:      "0.1.0",
	}
}

// Helper function to create starting position
func newUCIStartingPosition() *Position {
	return &Position{
		Board:         StartingBoard(),
		Tag:           WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove,
		EnPassant:     NoSquare,
		HalfMoveClock: 0,
	}
}

// Run starts the UCI engine main loop
func (uci *UCIEngine) Run(input io.Reader, output io.Writer) {
	scanner := bufio.NewScanner(input)
	
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command == "" {
			continue
		}
		
		if uci.debugMode {
			fmt.Fprintf(output, "info string received: %s\n", command)
		}
		
		uci.handleCommand(command, output)
	}
}

// handleCommand processes UCI commands
func (uci *UCIEngine) handleCommand(command string, output io.Writer) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}
	
	cmd := parts[0]
	args := parts[1:]
	
	switch cmd {
	case "uci":
		uci.handleUCI(output)
	case "debug":
		uci.handleDebug(args, output)
	case "isready":
		uci.handleIsReady(output)
	case "setoption":
		uci.handleSetOption(args, output)
	case "register":
		uci.handleRegister(args, output)
	case "ucinewgame":
		uci.handleNewGame(output)
	case "position":
		uci.handlePosition(args, output)
	case "go":
		uci.handleGo(args, output)
	case "stop":
		uci.handleStop(output)
	case "ponderhit":
		uci.handlePonderHit(output)
	case "quit":
		uci.handleQuit(output)
	default:
		if uci.debugMode {
			fmt.Fprintf(output, "info string unknown command: %s\n", cmd)
		}
	}
}

// handleUCI responds to the uci command
func (uci *UCIEngine) handleUCI(output io.Writer) {
	fmt.Fprintf(output, "id name %s %s\n", uci.engineName, uci.version)
	fmt.Fprintf(output, "id author %s\n", uci.engineAuthor)
	
	// Engine options (we can add more later)
	fmt.Fprintf(output, "option name Hash type spin default 128 min 1 max 1024\n")
	fmt.Fprintf(output, "option name Threads type spin default 1 min 1 max 64\n")
	fmt.Fprintf(output, "option name Move Overhead type spin default 100 min 0 max 5000\n")
	
	fmt.Fprintf(output, "uciok\n")
}

// handleDebug handles debug mode toggle
func (uci *UCIEngine) handleDebug(args []string, output io.Writer) {
	if len(args) > 0 {
		switch args[0] {
		case "on":
			uci.debugMode = true
		case "off":
			uci.debugMode = false
		}
	}
}

// handleIsReady responds to isready command
func (uci *UCIEngine) handleIsReady(output io.Writer) {
	fmt.Fprintf(output, "readyok\n")
}

// handleSetOption handles option setting (placeholder for now)
func (uci *UCIEngine) handleSetOption(args []string, output io.Writer) {
	// For now, just acknowledge option setting
	// TODO: Implement actual option handling
	if uci.debugMode {
		fmt.Fprintf(output, "info string option set: %s\n", strings.Join(args, " "))
	}
}

// handleRegister handles registration (not needed for open source engine)
func (uci *UCIEngine) handleRegister(args []string, output io.Writer) {
	// Open source engine, no registration needed
}

// handleNewGame handles ucinewgame command
func (uci *UCIEngine) handleNewGame(output io.Writer) {
	uci.position = newUCIStartingPosition()
	// Clear transposition table for new game
	TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)
	
	if uci.debugMode {
		fmt.Fprintf(output, "info string new game started\n")
	}
}

// handlePosition handles position setup
func (uci *UCIEngine) handlePosition(args []string, output io.Writer) {
	if len(args) == 0 {
		return
	}
	
	switch args[0] {
	case "startpos":
		uci.position = newUCIStartingPosition()
		args = args[1:] // Remove "startpos"
	case "fen":
		if len(args) < 7 { // Need at least "fen" + 6 FEN parts
			if uci.debugMode {
				fmt.Fprintf(output, "info string invalid FEN command\n")
			}
			return
		}
		
		// Join FEN parts (positions 1-6 after "fen")
		fenString := strings.Join(args[1:7], " ")
		pos, err := ParseFEN(fenString)
		if err != nil {
			if uci.debugMode {
				fmt.Fprintf(output, "info string FEN parse error: %v\n", err)
			}
			return
		}
		uci.position = pos
		args = args[7:] // Remove "fen" and FEN string
	}
	
	// Handle moves if present
	if len(args) >= 2 && args[0] == "moves" {
		moves := args[1:]
		for _, moveStr := range moves {
			move, err := ParseAlgebraicMove(moveStr, uci.position)
			if err != nil {
				if uci.debugMode {
					fmt.Fprintf(output, "info string invalid move: %s (%v)\n", moveStr, err)
				}
				return
			}
			
			// Make the move
			_, _, _, _ = uci.position.MakeMove(move)
		}
	}
}

// SearchParams holds parameters for the search
type SearchParams struct {
	Depth      int
	Nodes      uint64
	MoveTime   int // milliseconds
	WhiteTime  int // milliseconds
	BlackTime  int // milliseconds
	WhiteInc   int // milliseconds increment
	BlackInc   int // milliseconds increment
	MovesToGo  int
	Infinite   bool
	Ponder     bool
}

// handleGo handles the go command
func (uci *UCIEngine) handleGo(args []string, output io.Writer) {
	if uci.searching {
		return // Already searching
	}
	
	params := SearchParams{
		Depth:     6, // Default depth
		Infinite:  false,
	}
	
	// Parse search parameters
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "depth":
			if i+1 < len(args) {
				if depth, err := strconv.Atoi(args[i+1]); err == nil {
					params.Depth = depth
				}
				i++
			}
		case "nodes":
			if i+1 < len(args) {
				if nodes, err := strconv.ParseUint(args[i+1], 10, 64); err == nil {
					params.Nodes = nodes
				}
				i++
			}
		case "movetime":
			if i+1 < len(args) {
				if moveTime, err := strconv.Atoi(args[i+1]); err == nil {
					params.MoveTime = moveTime
				}
				i++
			}
		case "wtime":
			if i+1 < len(args) {
				if wtime, err := strconv.Atoi(args[i+1]); err == nil {
					params.WhiteTime = wtime
				}
				i++
			}
		case "btime":
			if i+1 < len(args) {
				if btime, err := strconv.Atoi(args[i+1]); err == nil {
					params.BlackTime = btime
				}
				i++
			}
		case "winc":
			if i+1 < len(args) {
				if winc, err := strconv.Atoi(args[i+1]); err == nil {
					params.WhiteInc = winc
				}
				i++
			}
		case "binc":
			if i+1 < len(args) {
				if binc, err := strconv.Atoi(args[i+1]); err == nil {
					params.BlackInc = binc
				}
				i++
			}
		case "movestogo":
			if i+1 < len(args) {
				if mtg, err := strconv.Atoi(args[i+1]); err == nil {
					params.MovesToGo = mtg
				}
				i++
			}
		case "infinite":
			params.Infinite = true
		case "ponder":
			params.Ponder = true
		}
	}
	
	// Start search in goroutine
	go uci.search(params, output)
}

// search performs the actual search
func (uci *UCIEngine) search(params SearchParams, output io.Writer) {
	uci.searching = true
	defer func() { uci.searching = false }()
	
	startTime := time.Now()
	
	// Calculate search depth based on time if not specified
	searchDepth := params.Depth
	if params.MoveTime > 0 {
		// Simple time-to-depth heuristic
		if params.MoveTime < 1000 {
			searchDepth = 4
		} else if params.MoveTime < 5000 {
			searchDepth = 5
		} else {
			searchDepth = 6
		}
	}
	
	// Perform iterative deepening search
	var bestMove Move
	var bestScore int
	
	for depth := 1; depth <= searchDepth; depth++ {
		// Check if we should stop
		select {
		case <-uci.stopChan:
			break
		default:
		}
		
		// Check time limit
		elapsed := time.Since(startTime)
		if params.MoveTime > 0 && elapsed.Milliseconds() >= int64(params.MoveTime*3/4) {
			break
		}
		
		info := Search(uci.position, depth)
		if info.BestMove != EmptyMove {
			bestMove = info.BestMove
			bestScore = info.BestScore
			
			// Send search info
			fmt.Fprintf(output, "info depth %d score cp %d nodes %d time %d pv %s\n",
				depth, bestScore, info.Nodes, elapsed.Milliseconds(), bestMove.ToString())
		}
	}
	
	// Send best move
	if bestMove != EmptyMove {
		fmt.Fprintf(output, "bestmove %s\n", bestMove.ToString())
	} else {
		// No legal moves (checkmate or stalemate)
		fmt.Fprintf(output, "bestmove (none)\n")
	}
}

// handleStop handles the stop command
func (uci *UCIEngine) handleStop(output io.Writer) {
	if uci.searching {
		select {
		case uci.stopChan <- true:
		default:
		}
	}
}

// handlePonderHit handles ponderhit command
func (uci *UCIEngine) handlePonderHit(output io.Writer) {
	// TODO: Implement pondering support
}

// handleQuit handles the quit command
func (uci *UCIEngine) handleQuit(output io.Writer) {
	if uci.searching {
		uci.handleStop(output)
	}
	// The main loop should exit after this
}