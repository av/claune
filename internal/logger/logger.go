package logger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/everlier/claune/internal/xdg"
)

func logFilePath() string {
	dir := xdg.StateHome()
	return filepath.Join(dir, "claune.log")
}

func logMessage(level, format string, args []interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, msg)

	path := logFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(line)
	}
}

func Error(format string, args ...interface{}) {
	logMessage("ERROR", format, args)
}

func Info(format string, args ...interface{}) {
	logMessage("INFO", format, args)
}

func ShowLogs(n int) error {
	path := logFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("No logs found at", path)
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	
	fmt.Printf("Showing last %d lines from %s:\n\n", len(lines)-start, path)
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, "ERROR:") {
			fmt.Printf("\033[31m%s\033[0m\n", line)
		} else if strings.Contains(line, "INFO:") {
			fmt.Printf("\033[32m%s\033[0m\n", line)
		} else {
			fmt.Println(line)
		}
	}
	return nil
}

func ClearLogs() error {
	path := logFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("No logs found.")
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to clear logs: %w", err)
	}
	fmt.Println("Logs cleared successfully.")
	return nil
}
