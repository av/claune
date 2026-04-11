package cli

import (
	"fmt"
	"os"
)

// ANSI Color Codes
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// Helper functions for common colored output
func PrintError(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%sError:%s %s\n", ColorRed, ColorReset, fmt.Sprintf(format, a...))
}

func PrintWarning(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%sWarning:%s %s\n", ColorYellow, ColorReset, fmt.Sprintf(format, a...))
}

func PrintSuccess(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, "%s%s%s\n", ColorGreen, fmt.Sprintf(format, a...), ColorReset)
}

func PrintInfo(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, "%s%s%s\n", ColorCyan, fmt.Sprintf(format, a...), ColorReset)
}

func Style(text string, color string) string {
	return fmt.Sprintf("%s%s%s", color, text, ColorReset)
}
