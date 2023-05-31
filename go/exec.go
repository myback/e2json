package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Output struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Rs     int    `json:"rs"`
}

const (
	commandNotFoundError = "command not found"
	defaultDuration      = 3 * time.Second
	usage                = `Usage: %s [--timeout <timeout>] <command> [arguments]...

  --timeout duration	Timeout (default %s)
`

	bashPipeFail = "set -o pipefail\n"
)

var (
	errParser = fmt.Errorf("parse error")

	usedShell = func() string {
		if _, e := os.Stat("/bin/bash"); e == nil {
			return "/bin/bash"
		}

		return "/bin/sh"
	}()
)

func main() {
	timeout, command, err := parseArgs(os.Args[1:])
	if err != nil {
		if err != errParser {
			fmt.Println(err)
		}

		_, _ = fmt.Fprintf(os.Stderr, usage, filepath.Base(os.Args[0]), defaultDuration)
		os.Exit(3)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var signalNotify string
	go func() {
		s := <-sig
		if s != nil {
			signalNotify = s.String()
		}
		cancel()
	}()

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	out := run(cmd)

	eOut := make([]string, 0)
	if ctx.Err() != nil {
		eOut = append(eOut, ctx.Err().Error())
	}

	if len(out.Stderr) > 0 {
		eOut = append(eOut, out.Stderr)
	}

	if len(signalNotify) > 0 {
		eOut = append(eOut, "got signal: "+signalNotify)
	}

	out.Stderr = strings.Join(eOut, "; ")

	_ = json.NewEncoder(os.Stdout).Encode(out)

	close(sig)
}

func run(cmd *exec.Cmd) Output {
	stdout := bytes.Buffer{}

	cmd.Stdout = &stdout

	out := Output{}
	if err := cmd.Start(); err != nil {
		if strings.Contains(err.Error(), "not found") {
			out.Stderr = fmt.Sprintf("%s: %s", commandNotFoundError, cmd.Path)
		} else {
			out.Stderr = err.Error()
		}

		out.Rs = -2

		return out
	}

	if err := cmd.Wait(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			out.Rs = e.ExitCode()
			out.Stderr = string(e.Stderr)
		} else {
			out.Rs = -3
			out.Stderr = err.Error()

			return out
		}
	}

	out.Stdout = stdout.String()

	return out
}

func parseArgs(args []string) (time.Duration, []string, error) {
	if len(args) == 0 {
		return 0, nil, errParser
	}

	timeout := defaultDuration
	cmd := args
	if cmd[0][0] == '-' {
		if cmd[0] != "--timeout" {
			return 0, nil, fmt.Errorf("unknown argument: %s", args[0])
		}

		if len(cmd) == 1 {
			return 0, nil, errParser
		}

		var err error
		timeout, err = time.ParseDuration(cmd[1])
		if err != nil {
			return 0, nil, err
		}

		cmd = cmd[2:]
	}

	switch len(cmd) {
	case 0:
		return 0, nil, errParser
	case 1:
		if strings.Contains(cmd[0], "\n") {
			// multiline script
			pre := bashPipeFail + cmd[0]
			cmd = []string{usedShell, "-ce", pre}
		} else {

			// one line script
			script := strings.Split(cmd[0], " ")
			if len(script) > 1 {
				cmd = script
			}
		}
	}

	return timeout, cmd, nil
}
