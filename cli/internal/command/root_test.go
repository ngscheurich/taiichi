package command_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ngscheurich/taiichi/cli/internal/command"
)

func TestHelpListsNoCommands(t *testing.T) {
	t.Parallel()
	root := command.NewRoot()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--help"})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("taiichi --help: %v\nstderr:\n%s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Move Jobs through the Pipeline") {
		t.Errorf("taiichi --help printed no description:\n%s", stdout.String())
	}
	if root.HasAvailableSubCommands() {
		t.Errorf("taiichi --help lists commands, want none:\n%s", stdout.String())
	}
}
