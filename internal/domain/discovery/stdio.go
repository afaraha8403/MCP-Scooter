package discovery

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// StdioWorker handles execution of an MCP tool over stdio.
type StdioWorker struct {
	command string
	args    []string
	cmd     *exec.Cmd
	ctx     context.Context
}

func NewStdioWorker(ctx context.Context, command string, args []string) *StdioWorker {
	return &StdioWorker{
		command: command,
		args:    args,
		ctx:     ctx,
	}
}

// Execute runs the command and pipes stdin/stdout.
// For one-off testing, we can use this. For persistent servers, we'd keep the process open.
func (w *StdioWorker) Execute(stdin io.Reader, stdout io.Writer, env map[string]string) error {
	w.cmd = exec.CommandContext(w.ctx, w.command, w.args...)
	w.cmd.Stdin = stdin
	w.cmd.Stdout = stdout
	w.cmd.Stderr = os.Stderr

	// Merge provided env with current process env
	w.cmd.Env = os.Environ()
	for k, v := range env {
		w.cmd.Env = append(w.cmd.Env, k+"="+v)
	}

	return w.cmd.Run()
}

func (w *StdioWorker) Close() error {
	if w.cmd != nil && w.cmd.Process != nil {
		return w.cmd.Process.Kill()
	}
	return nil
}
