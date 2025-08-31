package main

import (
	"fmt"
	"os"
	"log"
	"path/filepath"
	"time"
)

func testSearch() {
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	logFile, _ := os.OpenFile(filepath.Join(execDir, "test_debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	logger := log.New(logFile, "[TEST] ", log.LstdFlags|log.Lmicroseconds)
	
	logger.Println("About to launch goroutine")
	
	go func() {
		logger.Println("Goroutine started!")
		time.Sleep(100 * time.Millisecond)
		logger.Println("Goroutine completed!")
	}()
	
	logger.Println("Goroutine launched")
	time.Sleep(500 * time.Millisecond) // Give time for goroutine
	logger.Println("Test complete")
}

func main() {
	testSearch()
	fmt.Println("Done")
}