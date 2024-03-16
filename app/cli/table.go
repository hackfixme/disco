package cli

import (
	"io"

	"github.com/olekukonko/tablewriter"
)

func newTable(header []string, data [][]string, w io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(w)
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("   ")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data)

	return table
}
