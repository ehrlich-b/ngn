package engine

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Global opening book instance
var globalOpeningBook *PolyglotBook

// InitOpeningBook loads an opening book from the given path
func InitOpeningBook(path string) error {
	book, err := LoadPolyglotBook(path)
	if err != nil {
		return err
	}
	globalOpeningBook = book
	return nil
}

// GetOpeningBook returns the global opening book
func GetOpeningBook() *PolyglotBook {
	return globalOpeningBook
}

// TryLoadDefaultBook attempts to load a book from common locations
func TryLoadDefaultBook() {
	// Try common book locations
	bookPaths := []string{
		"book.bin",
		"opening.bin",
		"openings.bin",
		"./book/book.bin",
		"./book/opening.bin",
	}

	for _, path := range bookPaths {
		if err := InitOpeningBook(path); err == nil {
			return
		}
	}
}

// UCI Engine represents the UCI interface for the chess engine
type UCIEngine struct {
	position         *Position
	searching        bool
	stopChan         chan bool
	debugMode        bool
	engineName       string
	engineAuthor     string
	version          string
	timeManager      *TimeManager
	movesPlayed      int
	lastScore        int
	debugLogger      *log.Logger
	rawUCILogger     *log.Logger
	allowResignation bool
}

// NewUCIEngine creates a new UCI engine instance
func NewUCIEngine() *UCIEngine {
	// Set up debug logging to file in executable directory
	execPath, err := os.Executable()
	if err != nil {
		execPath = "./ngn" // fallback
	}
	execDir := filepath.Dir(execPath)
	logFile, err := os.OpenFile(filepath.Join(execDir, "ngn_debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err != nil {
		// Fallback to stdout if can't create log file
		logger = log.New(os.Stdout, "[NGN DEBUG] ", log.LstdFlags|log.Lmicroseconds)
	} else {
		logger = log.New(logFile, "[NGN DEBUG] ", log.LstdFlags|log.Lmicroseconds)
	}

	logger.Println("=== NGN Chess Engine Starting ===")
	logger.Printf("Executable path: %s", execPath)
	logger.Printf("Log file directory: %s", execDir)

	// Also create raw UCI message logger
	uciLogFile, err := os.OpenFile(filepath.Join(execDir, "ngn_raw_uci.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var rawUCILogger *log.Logger
	if err != nil {
		rawUCILogger = logger // fallback to main logger
	} else {
		rawUCILogger = log.New(uciLogFile, "", log.LstdFlags|log.Lmicroseconds)
	}
	rawUCILogger.Println("=== RAW UCI MESSAGE LOG ===")
	rawUCILogger.Println("Format: >>> = sent to engine, <<< = sent from engine")

	// Try to load opening book
	TryLoadDefaultBook()
	if book := GetOpeningBook(); book != nil {
		logger.Printf("Opening book loaded: %d entries", book.Size())
	} else {
		logger.Println("No opening book found")
	}

	return &UCIEngine{
		position:         newUCIStartingPosition(),
		searching:        false,
		stopChan:         make(chan bool, 1),
		debugMode:        false,
		engineName:       "ngn",
		engineAuthor:     "ngn team",
		version:          "0.1.0",
		timeManager:      NewTimeManager(),
		movesPlayed:      0,
		lastScore:        0,
		debugLogger:      logger,
		rawUCILogger:     rawUCILogger,
		allowResignation: false, // Default to no resignation
	}
}

// sendUCIMessage sends a UCI message and logs it
func (uci *UCIEngine) sendUCIMessage(output io.Writer, message string) {
	uci.rawUCILogger.Printf("<<< %s", message)
	fmt.Fprintln(output, message)
}

// SetResignationPolicy sets whether the engine should resign in hopeless positions
func (uci *UCIEngine) SetResignationPolicy(allowResignation bool) {
	uci.allowResignation = allowResignation
}

// Helper function to create starting position
func newUCIStartingPosition() *Position {
	pos := &Position{
		Board:         StartingBoard(),
		Tag:           WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove,
		EnPassant:     NoSquare,
		HalfMoveClock: 0,
		Positions:     make(map[uint64]int),
	}
	// Add initial position to repetition tracking
	pos.positionsMutex.Lock()
	pos.Positions[pos.Hash()] = 1
	pos.positionsMutex.Unlock()
	return pos
}

// Run starts the UCI engine main loop
func (uci *UCIEngine) Run(input io.Reader, output io.Writer) {
	defer GetGlobalCrashHandler().SafeRecover("UCIEngine.Run", map[string]interface{}{
		"engine_name": uci.engineName,
		"debug_mode":  uci.debugMode,
	})

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command == "" {
			continue
		}

		// Log raw UCI input
		uci.rawUCILogger.Printf(">>> %s", command)

		if uci.debugMode {
			fmt.Fprintf(output, "info string received: %s\n", command)
		}

		// Wrap each command in panic recovery
		func() {
			defer GetGlobalCrashHandler().SafeRecover("UCIEngine.handleCommand", map[string]interface{}{
				"command": command,
				"time":    time.Now().Format(time.RFC3339),
			})
			uci.handleCommand(command, output)
		}()
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
	uci.sendUCIMessage(output, fmt.Sprintf("id name %s %s", uci.engineName, uci.version))
	uci.sendUCIMessage(output, fmt.Sprintf("id author %s", uci.engineAuthor))

	// Engine options (we can add more later)
	uci.sendUCIMessage(output, "option name Hash type spin default 128 min 1 max 1024")
	uci.sendUCIMessage(output, "option name Threads type spin default 1 min 1 max 64")
	uci.sendUCIMessage(output, "option name Move Overhead type spin default 100 min 0 max 5000")
	uci.sendUCIMessage(output, "option name SyzygyPath type string default <empty>")

	uci.sendUCIMessage(output, "uciok")
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
	uci.sendUCIMessage(output, "readyok")
}

// handleSetOption handles option setting
func (uci *UCIEngine) handleSetOption(args []string, output io.Writer) {
	// Parse "name <name> value <value>" format
	var optionName, optionValue string
	for i := 0; i < len(args); i++ {
		if args[i] == "name" && i+1 < len(args) {
			// Collect name until "value" keyword
			nameStart := i + 1
			nameEnd := nameStart
			for j := nameStart; j < len(args); j++ {
				if args[j] == "value" {
					break
				}
				nameEnd = j + 1
			}
			optionName = strings.Join(args[nameStart:nameEnd], " ")
		}
		if args[i] == "value" && i+1 < len(args) {
			optionValue = strings.Join(args[i+1:], " ")
			break
		}
	}

	// Handle specific options
	switch strings.ToLower(optionName) {
	case "syzygypath":
		if optionValue != "" && optionValue != "<empty>" {
			err := InitSyzygy(optionValue)
			if err != nil {
				uci.debugLogger.Printf("Failed to initialize Syzygy tablebases: %v", err)
				if uci.debugMode {
					fmt.Fprintf(output, "info string syzygy init failed: %v\n", err)
				}
			} else {
				uci.debugLogger.Printf("Syzygy tablebases loaded from: %s (max %d pieces)", optionValue, TBLargestPieceCount())
				if uci.debugMode {
					fmt.Fprintf(output, "info string syzygy loaded: %s (%d pieces)\n", optionValue, TBLargestPieceCount())
				}
			}
		}
	case "hash":
		// TODO: Implement hash table size adjustment
	case "threads":
		// TODO: Implement multi-threading
	}

	if uci.debugMode {
		fmt.Fprintf(output, "info string option set: %s = %s\n", optionName, optionValue)
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
	// Clear history table for new game
	ClearHistoryTable()
	// Clear killer moves for new game
	ClearKillerMoves()
	// Clear counter moves for new game
	ClearCounterMoves()

	// Reset game state
	uci.movesPlayed = 0
	uci.lastScore = 0
	uci.timeManager = NewTimeManager()

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
		uci.movesPlayed = 0          // Reset move counter
		SetLastMovePlayed(EmptyMove) // Reset last move for new position
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
			uci.movesPlayed++
			// Track the last move played for counter move heuristic
			SetLastMovePlayed(move)
		}
	}
}

// SearchParams holds parameters for the search
type SearchParams struct {
	Depth     int
	Nodes     uint64
	MoveTime  int // milliseconds
	WhiteTime int // milliseconds
	BlackTime int // milliseconds
	WhiteInc  int // milliseconds increment
	BlackInc  int // milliseconds increment
	MovesToGo int
	Infinite  bool
	Ponder    bool
}

// handleGo handles the go command
func (uci *UCIEngine) handleGo(args []string, output io.Writer) {
	uci.debugLogger.Printf("handleGo called with args: %v", args)
	if uci.debugMode {
		fmt.Fprintf(output, "info string handleGo called\n")
	}
	if uci.searching {
		uci.debugLogger.Println("Already searching, returning")
		if uci.debugMode {
			fmt.Fprintf(output, "info string already searching, returning\n")
		}
		return // Already searching
	}

	params := SearchParams{
		Depth:    0, // No default depth - will use time-based search
		Infinite: false,
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

	// Configure time management
	uci.debugLogger.Println("About to set up time management")
	if uci.debugMode {
		fmt.Fprintf(output, "info string setting up time management\n")
	}
	uci.timeManager.SetTimeControl(params, uci.position.Turn() == White)
	uci.debugLogger.Println("Time management set, about to update game state")
	if uci.debugMode {
		fmt.Fprintf(output, "info string updating game state\n")
	}
	uci.timeManager.UpdateGameState(uci.movesPlayed, uci.lastScore, uci.lastScore, uci.position)

	// Start search in goroutine
	uci.debugLogger.Println("About to start search goroutine")
	if uci.debugMode {
		fmt.Fprintf(output, "info string starting search goroutine\n")
	}
	go uci.search(params, output)
	uci.debugLogger.Println("Search goroutine launched")

	// Give goroutine a moment to start (important for UCI)
	time.Sleep(10 * time.Millisecond)
}

// search performs the actual search
func (uci *UCIEngine) search(params SearchParams, output io.Writer) {
	defer GetGlobalCrashHandler().SafeRecover("UCIEngine.search", map[string]interface{}{
		"params":       params,
		"position":     GenerateFEN(uci.position),
		"moves_played": uci.movesPlayed,
		"searching":    uci.searching,
	})

	uci.debugLogger.Printf("search() function started with params: %+v", params)
	if uci.debugMode {
		fmt.Fprintf(output, "info string search() function started\n")
	}
	uci.searching = true
	defer func() {
		uci.searching = false
		uci.debugLogger.Println("search() function completed")
	}()

	uci.debugLogger.Println("search initialized, setting flags")
	if uci.debugMode {
		fmt.Fprintf(output, "info string search initialized\n")
	}

	// Determine maximum search depth
	maxDepth := params.Depth
	if maxDepth <= 0 {
		// No explicit depth specified - use time-based search with high max depth
		maxDepth = MaximumDepth
	}

	// Initialize search variables
	var bestMove Move
	var bestScore int
	lastIterationScore := uci.lastScore

	if uci.debugMode {
		fmt.Fprintf(output, "info string time control: %s, allocated: %dms\n",
			uci.timeManager.GetTimeControlString(),
			uci.timeManager.GetAllocatedTime().Milliseconds())
	}

	// Debug: Always output what we're doing
	if uci.debugMode {
		fmt.Fprintf(output, "info string starting search, maxDepth=%d\n", maxDepth)
	}

	// Check opening book first before starting search
	bookMove, bookFound := EmptyMove, false
	if book := GetOpeningBook(); book != nil {
		bookMove, bookFound = book.ProbeBook(uci.position)
	}
	// Fallback to embedded book if no external book or no hit
	if !bookFound {
		bookMove, bookFound = ProbeEmbeddedBook(uci.position)
	}

	if bookFound {
		uci.debugLogger.Printf("Opening book hit! Move: %v", bookMove.ToString())
		if uci.debugMode {
			fmt.Fprintf(output, "info string opening book move: %s\n", bookMove.ToString())
		}

		bestMove = bookMove
		bestScore = 50
		uci.debugLogger.Printf("Using book move: %v with score %d", bestMove.ToString(), bestScore)

		// Output the book move result
		fmt.Fprintf(output, "info depth 1 score cp %d nodes 1 time 0 nps 0 pv %s\n",
			bestScore, bestMove.ToString())
	} else if tbMove, tbResult := ProbeRoot(uci.position); tbResult.Found && tbMove != EmptyMove {
		// Syzygy tablebase hit - perfect endgame play
		uci.debugLogger.Printf("Syzygy tablebase hit! WDL=%d DTZ=%d Move=%s", tbResult.WDL, tbResult.DTZ, tbMove.ToString())
		if uci.debugMode {
			fmt.Fprintf(output, "info string tablebase move: %s (WDL=%d)\n", tbMove.ToString(), tbResult.WDL)
		}

		bestMove = tbMove
		// Convert WDL to centipawn score
		switch tbResult.WDL {
		case WDL_Win:
			bestScore = MATE_VALUE - 100 - tbResult.DTZ // Winning
		case WDL_CursedWin:
			bestScore = 500 // Theoretically winning but 50-move draw
		case WDL_Draw:
			bestScore = 0
		case WDL_BlessedLoss:
			bestScore = -500 // Theoretically losing but 50-move draw
		case WDL_Loss:
			bestScore = -MATE_VALUE + 100 + tbResult.DTZ // Losing
		}

		uci.debugLogger.Printf("Using tablebase move: %v with score %d", bestMove.ToString(), bestScore)

		// Output the tablebase result
		var scoreStr string
		if bestScore >= MATE_IN_MAX {
			scoreStr = fmt.Sprintf("mate %d", (MATE_VALUE-bestScore+1)/2)
		} else if bestScore <= -MATE_IN_MAX {
			scoreStr = fmt.Sprintf("mate %d", -(MATE_VALUE+bestScore+1)/2)
		} else {
			scoreStr = fmt.Sprintf("cp %d", bestScore)
		}
		fmt.Fprintf(output, "info depth 1 score %s nodes 1 time 0 nps 0 tbhits 1 pv %s\n",
			scoreStr, bestMove.ToString())
	} else {
		uci.debugLogger.Printf("No opening book move found for current position")
		if uci.debugMode {
			fmt.Fprintf(output, "info string no book move found, starting search\n")
		}

		// Perform iterative deepening search with callback for UCI info output
		uci.debugLogger.Printf("Starting SearchIterativeDeepeningWithCallback, maxDepth=%d", maxDepth)

		// Define callback to send UCI info after each depth completes
		depthCallback := func(depth int, score int, move Move, nodes uint64) {
		// Check for stop command
		select {
		case <-uci.stopChan:
			uci.debugLogger.Printf("STOP COMMAND received during search")
			return
		default:
		}

		// Update best move and score
		bestMove = move
		bestScore = score
		lastIterationScore = score

		// Update time manager with score change
		uci.timeManager.UpdateGameState(uci.movesPlayed, lastIterationScore, score, uci.position)

		// Send search info
		elapsed := uci.timeManager.GetElapsedTime()

		// UCI: report mate scores as "score mate N" when applicable
		var scoreStr string
		if score == INFINITY || score == -INFINITY {
			scoreStr = "cp 0"
		} else if score >= MATE_IN_MAX {
			mate := (MATE_VALUE - score + 1) / 2
			scoreStr = fmt.Sprintf("mate %d", mate)
		} else if score <= -MATE_IN_MAX {
			mateDistance := -score - MATE_VALUE
			if mateDistance < 0 || mateDistance > 1000 {
				mateDistance = 1000
			}
			mate := -(mateDistance/2 + 1)
			scoreStr = fmt.Sprintf("mate %d", mate)
		} else {
			scoreStr = fmt.Sprintf("cp %d", score)
		}

		// Calculate NPS
		var nps int
		if elapsed.Seconds() > 0 {
			nps = int(float64(nodes) / elapsed.Seconds())
		}

		infoMsg := fmt.Sprintf("info depth %d score %s nodes %d time %d nps %d pv %s",
			depth, scoreStr, nodes, elapsed.Milliseconds(), nps, move.ToString())
		uci.sendUCIMessage(output, infoMsg)
		uci.debugLogger.Printf("Sent info line for depth %d: score=%s bestMove=%s", depth, scoreStr, move.ToString())
		}

		// Run the search - this handles progressive TT building and aspiration windows internally
		info := SearchIterativeDeepeningWithCallback(uci.position, maxDepth, uci.timeManager, depthCallback)

		uci.debugLogger.Printf("SearchIterativeDeepeningWithCallback completed: depth=%d, bestMove=%v, bestScore=%d, nodes=%d",
			info.Depth, info.BestMove, info.BestScore, info.Nodes)

		// Use final result from search
		if info.BestMove != EmptyMove {
			bestMove = info.BestMove
			bestScore = info.BestScore
		}

		uci.debugLogger.Printf("=== SEARCH COMPLETE ===")
	}

	// Update engine state after search completes
	uci.debugLogger.Printf("=== UPDATING ENGINE STATE ===")
	uci.debugLogger.Printf("bestMove=%v, bestScore=%d", bestMove, bestScore)
	uci.lastScore = bestScore
	uci.movesPlayed++
	uci.debugLogger.Printf("Updated lastScore=%d, movesPlayed=%d", uci.lastScore, uci.movesPlayed)

	// Send best move
	uci.debugLogger.Printf("*** ABOUT TO SEND BESTMOVE ***")
	if bestMove != EmptyMove {
		// CRITICAL: Validate the move is legal before sending
		legalMoves := GenerateLegalMoves(uci.position)
		isLegal := false
		for _, legalMove := range legalMoves {
			if bestMove == legalMove {
				isLegal = true
				break
			}
		}

		if !isLegal {
			uci.debugLogger.Printf("*** CRITICAL ERROR: Search returned illegal move %s, position FEN: %s ***", bestMove.ToString(), GenerateFEN(uci.position))
			uci.debugLogger.Printf("*** Legal moves in position: %v ***", legalMoves)

			// Try to find any legal move to avoid forfeiting
			if len(legalMoves) > 0 {
				bestMove = legalMoves[0]
				uci.debugLogger.Printf("*** Using first legal move instead: %s ***", bestMove.ToString())
			} else {
				// Truly no legal moves
				bestMove = EmptyMove
				uci.debugLogger.Printf("*** No legal moves available ***")
			}
		}

		if bestMove != EmptyMove {
			// Check if we should resign due to hopeless position
			if !uci.allowResignation && bestScore < -MATE_IN_MAX {
				// Position is hopeless but resignation is disabled - play the move anyway
				uci.debugLogger.Printf("*** HOPELESS POSITION (score=%d) but resignation disabled, playing move ***", bestScore)
			}

			moveStr := bestMove.ToString()
			uci.debugLogger.Printf("*** SENDING: bestmove %s ***", moveStr)
			uci.sendUCIMessage(output, fmt.Sprintf("bestmove %s", moveStr))
			uci.debugLogger.Printf("*** BESTMOVE SENT SUCCESSFULLY: %s ***", moveStr)
		} else {
			// No legal moves available
			uci.debugLogger.Printf("*** SENDING: bestmove (none) - no legal moves ***")
			uci.sendUCIMessage(output, "bestmove (none)")
			uci.debugLogger.Printf("*** BESTMOVE (none) SENT ***")
		}
	} else {
		// Check if we should try to find any legal move to avoid resignation
		if !uci.allowResignation {
			// Try to find any legal move to avoid resignation
			moves := GenerateLegalMoves(uci.position)
			if len(moves) > 0 {
				// Play the first legal move to avoid resignation
				anyMove := moves[0]
				moveStr := anyMove.ToString()
				uci.debugLogger.Printf("*** NO BEST MOVE but resignation disabled, playing any move: %s ***", moveStr)
				uci.sendUCIMessage(output, fmt.Sprintf("bestmove %s", moveStr))
				uci.debugLogger.Printf("*** BESTMOVE SENT: %s ***", moveStr)
			} else {
				// Truly no legal moves (checkmate or stalemate) - must send (none)
				uci.debugLogger.Printf("*** SENDING: bestmove (none) - truly no legal moves ***")
				uci.sendUCIMessage(output, "bestmove (none)")
				uci.debugLogger.Printf("*** BESTMOVE (none) SENT ***")
			}
		} else {
			// Resignation allowed - send (none)
			uci.debugLogger.Printf("*** SENDING: bestmove (none) ***")
			uci.sendUCIMessage(output, "bestmove (none)")
			uci.debugLogger.Printf("*** BESTMOVE (none) SENT ***")
		}
	}
	uci.debugLogger.Printf("=== EXITING search() FUNCTION ===")
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
