package ui

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	gray   = "\033[90m"
	bold   = "\033[1m"
	reset  = "\033[0m"
)

// Exported color functions for use in other packages.
func Yellow() string { return yellow }
func Green() string  { return green }
func Red() string    { return red }
func Gray() string   { return gray }
func Cyan() string   { return cyan }
func Reset() string  { return reset }
func Bold() string   { return bold }

// Header prints a section header.
func Header(format string, args ...interface{}) {
	fmt.Printf("\n%s▸%s %s\n", cyan, reset, fmt.Sprintf(format, args...))
}

// Step prints a progress step.
func Step(format string, args ...interface{}) {
	fmt.Printf("  %s·%s %s\n", gray, reset, fmt.Sprintf(format, args...))
}

// Success prints a success indicator.
func Success(format string, args ...interface{}) {
	fmt.Printf("  %s✓%s %s\n", green, reset, fmt.Sprintf(format, args...))
}

// Info prints an info message.
func Info(format string, args ...interface{}) {
	fmt.Printf("  %s:%s %s\n", gray, reset, fmt.Sprintf(format, args...))
}

// Error prints an error message.
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "  %s✗%s %s\n", red, reset, fmt.Sprintf(format, args...))
}

// Fatal prints an error and exits.
func Fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "  %s✗%s %s\n", red, reset, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Prompt asks for confirmation.
func Prompt(msg string) bool {
	fmt.Printf("  %s?%s %s [y/N] ", yellow, reset, msg)
	var resp string
	fmt.Scanln(&resp)
	return resp == "y" || resp == "Y"
}

// Spinner shows a simple spinner for long operations.
type Spinner struct {
	msg    string
	done   chan bool
	ticker *time.Ticker
}

// StartSpinner starts a spinner with a message.
func StartSpinner(msg string) *Spinner {
	s := &Spinner{
		msg:  msg,
		done: make(chan bool),
	}
	go s.run()
	return s
}

func (s *Spinner) run() {
	chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			fmt.Printf("\r  %s✓%s %s\n", green, reset, s.msg)
			return
		case <-ticker.C:
			fmt.Printf("\r  %s%s%s %s", gray, chars[i%len(chars)], reset, s.msg)
			i++
		}
	}
}

// Stop stops the spinner.
func (s *Spinner) Stop() {
	close(s.done)
	time.Sleep(10 * time.Millisecond)
}

// PackageList prints a list of packages in a compact format.
func PackageList(packages []string) {
	if len(packages) == 0 {
		return
	}
	const maxPerLine = 4
	for i := 0; i < len(packages); i += maxPerLine {
		end := i + maxPerLine
		if end > len(packages) {
			end = len(packages)
		}
		line := strings.Join(packages[i:end], ", ")
		fmt.Printf("    %s%s%s\n", gray, line, reset)
	}
}

// Stats prints cache/build stats.
func Stats(cached, built, total int) {
	fmt.Printf("\n  %s%d cached%s · %s%d built%s · %s%d total%s\n",
		green, cached, reset,
		yellow, built, reset,
		bold, total, reset,
	)
}
