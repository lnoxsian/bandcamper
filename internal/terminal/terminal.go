package terminal

import (
	"os"
	"runtime"
	"strings"
)

// Info holds details about the terminal environment and color capabilities.
type Info struct {
	OS             string
	IsWindows      bool
	IsTerminal     bool
	ColorSupported bool
	VTEnabled      bool
}

// Init initializes terminal-specific settings (such as Windows virtual terminal
// processing and UTF-8 console output code page) and returns an Info descriptor.
func Init() Info {
	info := Info{
		OS:         runtime.GOOS,
		IsWindows:  runtime.GOOS == "windows",
		IsTerminal: isTerminal(os.Stdout),
	}

	if info.IsWindows {
		info.VTEnabled = enableWindowsVT()
	}

	info.ColorSupported = isColorSupported(info)
	return info
}

// isColorSupported determines whether ANSI color output should be enabled
// based on OS, terminal capabilities, and environment variables.
func isColorSupported(info Info) bool {
	// Respect NO_COLOR specification (https://no-color.org)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Terminal explicitly declared as dumb
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	if info.IsWindows {
		// On Windows:
		// 1. If Virtual Terminal Processing was successfully enabled, full ANSI is supported.
		if info.VTEnabled {
			return true
		}
		// 2. Windows Terminal sets WT_SESSION
		if os.Getenv("WT_SESSION") != "" {
			return true
		}
		// 3. ConEmu with ANSI enabled
		if strings.EqualFold(os.Getenv("ConEmuANSI"), "ON") {
			return true
		}
		// 4. ANSICON driver installed
		if os.Getenv("ANSICON") != "" {
			return true
		}
		// 5. Unix-like shells on Windows (Git Bash / MSYS2 / Cygwin) setting TERM
		if term != "" && (strings.Contains(term, "xterm") || strings.Contains(term, "color") || strings.Contains(term, "vt100")) {
			return true
		}
		// 6. VS Code integrated terminal
		if os.Getenv("TERM_PROGRAM") == "vscode" {
			return true
		}

		// Legacy Windows console without VT or ANSI driver cannot render colors
		return false
	}

	// Non-Windows (Linux, macOS, BSD, etc.):
	// ANSI is standard unless stdout is redirected and not a terminal
	return true
}

// IsTerminal returns whether the provided file is a terminal/console.
func IsTerminal(f *os.File) bool {
	return isTerminal(f)
}
