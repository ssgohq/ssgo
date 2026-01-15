package cmd

import (
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

func TestRunSqlcCommand_UnknownSubcommand(t *testing.T) {
	ctx := &cmdctx.Context{
		Args: []string{"unknown"},
	}

	err := runSqlcCommand(ctx)
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestRunSqlcCommand_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "help", args: []string{"help"}},
		{name: "-h", args: []string{"-h"}},
		{name: "--help", args: []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &cmdctx.Context{
				Args: tt.args,
			}

			err := runSqlcCommand(ctx)
			if err != nil {
				t.Errorf("help command returned error: %v", err)
			}
		})
	}
}

func TestRunSqlcCommand_NoArgs(t *testing.T) {
	ctx := &cmdctx.Context{
		Args: []string{},
	}

	// Should print help, not error
	err := runSqlcCommand(ctx)
	if err != nil {
		t.Errorf("empty args should print help, not error: %v", err)
	}
}
