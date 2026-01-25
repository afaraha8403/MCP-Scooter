package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mcp-scooter/scooter/internal/cli/errors"
	"github.com/mcp-scooter/scooter/internal/domain/registry"
	"github.com/olekukonko/tablewriter"
)

type OutputFormat string

const (
	FormatText     OutputFormat = "text"
	FormatJSON     OutputFormat = "json"
	FormatRaw      OutputFormat = "raw"
	FormatMarkdown OutputFormat = "markdown"
)

type Formatter struct {
	format OutputFormat
	color  bool
}

func NewFormatter(format OutputFormat, useColor bool) *Formatter {
	return &Formatter{
		format: format,
		color:  useColor,
	}
}

func (f *Formatter) FormatResult(result *CallResult) string {
	if f.format == FormatJSON {
		s, _ := result.JSON()
		return s
	}
	if f.format == FormatMarkdown {
		return result.Markdown()
	}
	if f.format == FormatRaw {
		return result.Text("")
	}

	// Default text format
	if result.IsError() {
		return color.RedString("Error: ") + result.Text("\n")
	}
	return result.Text("\n")
}

func (f *Formatter) FormatError(err errors.ClassifiedError) string {
	if f.format == FormatJSON {
		data, _ := json.MarshalIndent(err, "", "  ")
		return string(data)
	}

	var msg string
	if f.color {
		msg = color.RedString("Error [%s]: %s", err.Kind, err.Message)
		if err.Hint != "" {
			msg += "\n" + color.YellowString("Hint: %s", err.Hint)
		}
	} else {
		msg = fmt.Sprintf("Error [%s]: %s", err.Kind, err.Message)
		if err.Hint != "" {
			msg += "\nHint: " + err.Hint
		}
	}
	return msg
}

func (f *Formatter) FormatTools(tools []registry.Tool) string {
	if f.format == FormatJSON {
		data, _ := json.MarshalIndent(tools, "", "  ")
		fmt.Println(string(data))
		return ""
	}

	// Use new tablewriter API
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader([]string{"Name", "Description"}),
	)

	for _, t := range tools {
		table.Append([]string{t.Name, t.Description})
	}

	table.Render()
	return "" // tablewriter writes directly to stdout
}

func (f *Formatter) FormatServers(entries []registry.MCPEntry) string {
	if f.format == FormatJSON {
		data, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Println(string(data))
		return ""
	}

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader([]string{"Name", "Category", "Description"}),
	)

	for _, e := range entries {
		table.Append([]string{e.Name, string(e.Category), e.Description})
	}

	table.Render()
	return ""
}
