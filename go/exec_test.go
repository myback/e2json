package main

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func Test_parseArgs(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name    string
		args    args
		timeout time.Duration
		cmd     []string
		wantErr bool
	}{
		{
			"ParseArgumentsWithoutTimeout",
			args{
				[]string{"true"},
			},
			defaultDuration,
			[]string{"true"},
			false,
		},
		{
			"ParseArgumentsWithTimeout",
			args{
				[]string{"--timeout", "1s", "true"},
			},
			time.Second,
			[]string{"true"},
			false,
		},
		{
			"ParseArgumentsWithInvalidTimeoutArgumentName1",
			args{
				[]string{"-timeout", "1s", "true"},
			},
			0,
			nil,
			true,
		},
		{
			"ParseArgumentsWithInvalidTimeoutArgumentName2",
			args{
				[]string{"-t", "1s", "true"},
			},
			0,
			nil,
			true,
		},
		{
			"ParseArgumentsWithoutTimeoutValue1",
			args{
				[]string{"--timeout"},
			},
			0,
			nil,
			true,
		},
		{
			"ParseArgumentsWithoutTimeoutValue2",
			args{
				[]string{"--timeout", "true"},
			},
			0,
			nil,
			true,
		},
		{
			"ParseArgumentsWithInvalidTimeoutValue",
			args{
				[]string{"--timeout", "1z", "true"},
			},
			0,
			nil,
			true,
		},
		{
			"ParseScript",
			args{
				[]string{`false && echo -n "OK" || echo -n "NO"`},
			},
			defaultDuration,
			[]string{"false", "&&", "echo", "-n", `"OK"`, "||", "echo", "-n", `"NO"`},
			false,
		},
		{
			"ParseMultilineScript",
			args{
				[]string{`if [ -n "$PWD" ]; then
	echo "OK"
else
	echo "NO"
fi`},
			},
			defaultDuration,
			[]string{usedShell, "-ce", bashPipeFail + "if [ -n \"$PWD\" ]; then\n\techo \"OK\"\nelse\n\techo \"NO\"\nfi"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseArgs(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.timeout {
				t.Errorf("parseArgs() got = %v, want %v", got, tt.timeout)
			}
			if !reflect.DeepEqual(got1, tt.cmd) {
				t.Errorf("parseArgs() got1 = %v, want %v", got1, tt.cmd)
			}
		})
	}
}

func Test_run(t *testing.T) {
	type args struct {
		cmd *exec.Cmd
	}
	tests := []struct {
		name string
		args args
		want Output
	}{
		{"CommandFalse",
			args{
				exec.Command("false"),
			},
			Output{Rs: 1},
		},
		{"CommandTrue",
			args{
				exec.Command("true"),
			},
			Output{Rs: 0},
		},
		{"CommandLs",
			args{
				exec.Command("ls", "exec_test.go"),
			},
			Output{Stdout: "exec_test.go\n", Rs: 0},
		},
		{"CommandNotFound",
			args{
				exec.Command("command-not-found-bin"),
			},
			Output{Stderr: fmt.Sprintf("%s: %s", commandNotFoundError, "command-not-found-bin"), Rs: -2},
		},
		{"CommandTimeout",
			args{
				_execCommandWithTimoutTest(),
			},
			Output{Rs: -1},
		},
		{"CommandScriptTrue",
			args{
				exec.Command("/bin/bash", "-ce", `true && echo -n "OK" || echo -n "NO"`),
			},
			Output{Stdout: "OK", Rs: 0},
		},
		{"CommandScriptFalse",
			args{
				exec.Command("/bin/bash", "-ce", `false && echo -n "OK" || echo -n "NO"`),
			},
			Output{Stdout: "NO", Rs: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(tt.args.cmd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func _execCommandWithTimoutTest() *exec.Cmd {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	return exec.CommandContext(ctx, "sleep", "5")
}
