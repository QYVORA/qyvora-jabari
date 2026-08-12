package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// shellKind classifies a console line as a host-shell invocation. It returns
// "interactive" for lines that drop into a live interactive shell, "shell"
// for one-shot host commands, and "" for anything else.
//
// The console ships with a deliberate shell escape hatch, exactly like
// Metasploit's `shell` and bettercap's `!` prefix: every unhandled command
// that starts with "!" is executed on the host, bare `shell` drops into an
// interactive shell, and `device shell` hands control to the selected
// device's shell. This is the operator's own machine and they typed the
// command themselves — it is never reachable from scan results, wordlists or
// any other untrusted input.
func (c *jabariConsole) shellKind(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	switch strings.ToLower(fields[0]) {
	case "cd", "pwd":
		return "shell"
	case "shell":
		if len(fields) == 1 {
			return "interactive"
		}
		return "shell"
	case "device":
		if len(fields) >= 2 && strings.EqualFold(fields[1], "shell") {
			if len(fields) == 2 {
				return "interactive"
			}
			return "shell"
		}
	}
	if strings.HasPrefix(fields[0], "!") {
		return "shell"
	}
	return ""
}

// changeDir updates the console's working directory. A leading "/" is taken
// as an absolute path, everything else resolves relative to the current
// console cwd (which stays across commands, mirroring a real shell).
func (c *jabariConsole) changeDir(arg string) error {
	base := c.cwd
	if base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		base = wd
	}
	target := base
	if arg != "" {
		if strings.HasPrefix(arg, "/") {
			target = filepath.Clean(arg)
		} else {
			target = filepath.Join(base, arg)
		}
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("cd: not a directory: %s", arg)
	}
	c.cwd = target
	c.ui.Status(">", "cwd %s", c.cwd)
	return nil
}

// printCwd prints the console's working directory.
func (c *jabariConsole) printCwd() {
	cwd := c.cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	c.ui.Status(">", "cwd %s", cwd)
}

// runHostCommand executes a one-shot host command with stdout/stderr wired to
// the console output so the operator sees the same thing they would in a
// terminal. The console's working directory is inherited so `cd` applies.
func (c *jabariConsole) runHostCommand(name string, args []string) error {
	bin, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s: command not found", name)
	}
	cwd := c.cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	cmd := exec.CommandContext(c.ctx, bin, args...)
	cmd.Dir = cwd
	cmd.Stdout = c.out
	cmd.Stderr = c.out
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runHostShell launches an interactive shell in the console's working
// directory, giving the operator a full terminal inside the console (the
// Metasploit `shell` / bettercap `!` escape hatch).
func (c *jabariConsole) runHostShell() error {
	c.ui.Status(">", "starting interactive shell; type 'exit' to return")
	cwd := c.cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.CommandContext(c.ctx, shell, "-i")
	cmd.Dir = cwd
	cmd.Stdout = c.out
	cmd.Stderr = c.out
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if c.ctx.Err() != nil {
			return nil
		}
		return err
	}
	return nil
}

// changeDirOrEscape implements the interactive "shell" sub-prompt: an empty
// line keeps the current directory, a directory argument changes it, and
// "exit"/"quit" returns to the console. It mirrors Metasploit's `shell` flow.
func (c *jabariConsole) changeDirOrEscape() (bool, error) {
	if c.rl == nil {
		return false, errors.New("interactive shell requires a terminal")
	}
	for {
		c.rl.SetPrompt(c.ui.DimWhite("cd (enter to keep, 'exit' to return)") + " > ")
		line, err := c.rl.Readline()
		c.rl.SetPrompt(c.ui.Prompt("jabari"))
		if err != nil {
			return true, nil // EOF (Ctrl-D) returns to the console
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return false, nil
		}
		low := strings.ToLower(line)
		if low == "exit" || low == "quit" {
			return true, nil
		}
		if err := c.changeDir(line); err != nil {
			c.ui.Err("%v", err)
			continue
		}
		return false, nil
	}
}
