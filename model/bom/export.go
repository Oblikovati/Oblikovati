// SPDX-License-Identifier: GPL-2.0-only

package bom

import (
	"encoding/csv"
	"strconv"
	"strings"
)

// Column is one exported BOM column: a header and a value extracted from a row. Custom
// columns ([PropertyColumn]) source their values from a component's properties, so a
// BOM can carry iProperty-driven columns (M11-F05, PBI-124).
type Column struct {
	Header string
	Value  func(*Row) string
}

// StandardColumns are the always-present BOM columns: item number, part number,
// description, quantity, and BOM structure.
func StandardColumns() []Column {
	return []Column{
		{"Item", func(r *Row) string { return strconv.Itoa(r.ItemNumber) }},
		{"Part Number", func(r *Row) string { return r.PartNumber }},
		{"Description", func(r *Row) string { return r.Description }},
		{"QTY", func(r *Row) string { return strconv.Itoa(r.Quantity) }},
		{"Structure", func(r *Row) string { return r.Structure.String() }},
	}
}

// PropertyColumn builds a custom column sourced from a component property (iProperty),
// e.g. "Material" or "Vendor"; a row missing the property exports an empty cell.
func PropertyColumn(name string) Column {
	return Column{Header: name, Value: func(r *Row) string { return r.Properties[name] }}
}

// ExportCSV renders view as CSV with the given columns — typically [StandardColumns]
// plus any [PropertyColumn]. A structured view is flattened depth-first (each row
// before its children); a parts-only view is already flat. Returns the CSV text.
func ExportCSV(view *View, columns []Column) (string, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	header := make([]string, len(columns))
	for i, c := range columns {
		header[i] = c.Header
	}
	if err := w.Write(header); err != nil {
		return "", err
	}
	for _, r := range flatten(view.Rows) {
		record := make([]string, len(columns))
		for i, c := range columns {
			record[i] = c.Value(r)
		}
		if err := w.Write(record); err != nil {
			return "", err
		}
	}
	w.Flush()
	return sb.String(), w.Error()
}

// flatten returns rows depth-first — each row immediately before its children — so a
// structured view exports in hierarchy order and a parts-only view passes through.
func flatten(rows []*Row) []*Row {
	var out []*Row
	for _, r := range rows {
		out = append(out, r)
		out = append(out, flatten(r.Children)...)
	}
	return out
}
