// ABOUTME: bd dispatch wrapper — translates bd (beads) commands to tl equivalents.
// ABOUTME: Known daily-use bd commands are forwarded to tl; unknown commands fall through to the real bd binary.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// knownTLCommands is the set of bd subcommands that tl handles directly.
// These match the daily-use patterns from CLAUDE.md and AGENTS.md.
var knownTLCommands = map[string]bool{
	"init":    true,
	"create":  true,
	"list":    true,
	"show":    true,
	"update":  true,
	"close":   true,
	"reopen":  true,
	"ready":   true,
	"claim":   true,
	"blocked": true,
	"stats":   true,
	"dep":     true,
	"import":  true,
	"export":  true,
	"sync":    true,
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 || !knownTLCommands[args[0]] {
		fallthroughBD(args)
		return
	}

	tl := findTL()
	cmd := exec.Command(tl, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func findTL() string {
	self, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(self), "tl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if path, err := exec.LookPath("tl"); err == nil {
		return path
	}
	return "tl"
}

func fallthroughBD(args []string) {
	if path, err := exec.LookPath("bd.real"); err == nil {
		_ = syscall.Exec(path, append([]string{path}, args...), os.Environ())
	}
	os.Stderr.WriteString("bd: command not handled by tl and real bd binary (bd.real) not found\n")
	os.Exit(1)
}
