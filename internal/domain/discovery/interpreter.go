package discovery

import (
	"fmt"

	"github.com/dop251/goja"
)

// CodeInterpreter runs sandboxed JS scripts with access to MCP tools.
type CodeInterpreter struct {
	vm         *goja.Runtime
	toolCaller func(name string, params map[string]interface{}) (interface{}, error)
}

func NewCodeInterpreter(caller func(name string, params map[string]interface{}) (interface{}, error)) *CodeInterpreter {
	vm := goja.New()
	return &CodeInterpreter{
		vm:         vm,
		toolCaller: caller,
	}
}

// Execute runs a JS script with provided arguments and returns the result.
func (i *CodeInterpreter) Execute(script string, args map[string]interface{}) (interface{}, error) {
	// Set arguments in a dedicated 'args' object
	i.vm.Set("args", args)

	// Add helper functions
	i.vm.Set("log", func(msg interface{}) {
		fmt.Printf("[CodeInterpreter] %v\n", msg)
	})

	i.vm.Set("callTool", func(name string, params map[string]interface{}) interface{} {
		if i.toolCaller == nil {
			return "Error: no tool caller available"
		}
		result, err := i.toolCaller(name, params)
		if err != nil {
			return fmt.Sprintf("Error calling %s: %v", name, err)
		}
		return result
	})

	// Wrap script in an IIFE to support 'return'
	fullScript := fmt.Sprintf("(function() { %s })()", script)
	value, err := i.vm.RunString(fullScript)
	if err != nil {
		return nil, err
	}

	return value.Export(), nil
}
