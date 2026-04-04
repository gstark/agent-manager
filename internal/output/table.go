package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Reverse(true).
			Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	dimCellStyle = cellStyle.Foreground(lipgloss.Color("#888888"))

	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
)

// IsTerminal reports whether stdout is connected to a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// Column defines a table column with a name and min width ratio.
type Column struct {
	Name    string
	MinPct  int // minimum percentage of terminal width for this column
	MaxPct  int // maximum percentage of terminal width
}

// PrintTable renders a styled table that fits the terminal.
// When stdout is not a TTY (e.g. piped), it prints tab-separated values
// with no truncation or styling.
func PrintTable(columns []Column, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	if !IsTerminal() {
		printPlain(columns, rows)
		return
	}

	tw := termWidth()

	// Reserve space for borders and padding (3 chars per col boundary + outer)
	overhead := len(columns)*3 + 1
	available := tw - overhead
	if available < 40 {
		available = 40
	}

	// Allocate using the configured percentages directly
	finalWidths := make([]int, len(columns))
	totalAllocated := 0
	for i, col := range columns {
		// Use MaxPct as the target allocation
		w := col.MaxPct * tw / 100
		minW := col.MinPct * tw / 100
		if w < minW {
			w = minW
		}
		if w < 6 {
			w = 6
		}
		finalWidths[i] = w
		totalAllocated += w
	}

	// If we exceed available, scale down proportionally
	if totalAllocated > available {
		for i := range finalWidths {
			finalWidths[i] = finalWidths[i] * available / totalAllocated
			if finalWidths[i] < 6 {
				finalWidths[i] = 6
			}
		}
	}

	// Build header names
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		BorderRow(true).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.Width(finalWidths[col])
			}
			// Dim the last column
			if col == len(columns)-1 {
				return dimCellStyle.Width(finalWidths[col])
			}
			return cellStyle.Width(finalWidths[col])
		}).
		Headers(headers...).
		Rows(rows...)

	fmt.Println(t)
}

// printPlain writes tab-separated output with no truncation, suitable for piped usage.
func printPlain(columns []Column, rows [][]string) {
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	fmt.Println(strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Println(strings.Join(row, "\t"))
	}
}

// PrintJSON marshals data as indented JSON to stdout.
func PrintJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
