package discovery

import (
	"context"
	"io"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASMWorker handles execution of a single WASM-based MCP tool.
type WASMWorker struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
	active  api.Module // Track the running instance
	ctx     context.Context
}

func NewWASMWorker(ctx context.Context) *WASMWorker {
	r := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	return &WASMWorker{
		runtime: r,
		ctx:     ctx,
	}
}

// Load loads a WASM file from disk.
func (w *WASMWorker) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	mod, err := w.runtime.CompileModule(w.ctx, data)
	if err != nil {
		return err
	}
	w.module = mod
	return nil
}

// Execute handles the MCP JSON-RPC interaction.
func (w *WASMWorker) Execute(stdin io.Reader, stdout io.Writer, env map[string]string) error {
	config := wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(stdout).
		WithStderr(os.Stderr).
		WithArgs("mcp-tool")

	for k, v := range env {
		config = config.WithEnv(k, v)
	}

	// For standard MCP tools over stdio, instantiation *is* the execution.
	// It will block until the module completes or the context is cancelled.
	mod, err := w.runtime.InstantiateModule(w.ctx, w.module, config)
	if err != nil {
		return err
	}
	defer mod.Close(w.ctx)
	w.active = mod
	return nil
}

func (w *WASMWorker) Close() error {
	if w.active != nil {
		w.active.Close(w.ctx)
	}
	return w.runtime.Close(w.ctx)
}
