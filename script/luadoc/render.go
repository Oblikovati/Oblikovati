// SPDX-License-Identifier: GPL-2.0-only

package luadoc

import "strings"

// writeReference renders the grouped method reference: one table per wire group, each row a
// scriptable call with its arguments and description.
func writeReference(b *strings.Builder, methods []Method) {
	groups, by := groupMethods(methods)
	b.WriteString("## API reference\n\n")
	b.WriteString("Every method below is callable two ways: the typed group form shown, or the " +
		"generic `oblikovati.call(\"" + "group.method" + "\", { … })`. Arguments are passed in a single " +
		"table; the columns list each table key and its type. There are " + itoa(len(methods)) +
		" methods across " + itoa(len(groups)) + " groups.\n\n")
	for _, g := range groups {
		b.WriteString("### " + g + "\n\n")
		b.WriteString("| Call | Arguments | Description |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, m := range by[g] {
			b.WriteString("| `oblikovati." + m.Group + "." + m.Leaf + "{…}` | " +
				argsCell(m.Args) + " | " + summaryCell(m.Summary) + " |\n")
		}
		b.WriteString("\n")
	}
}

// argsCell renders a method's argument fields as backticked `name type` pairs, or "—".
func argsCell(args []Field) string {
	if len(args) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, "`"+a.Name+" "+a.Type+"`")
	}
	return strings.Join(parts, ", ")
}

// summaryCell returns the method summary, or a placeholder when the API carries none.
func summaryCell(s string) string {
	if strings.TrimSpace(s) == "" {
		return "_(no summary)_"
	}
	return s
}

// writeExamples renders the bundled example library: each program's description and full
// source in a Lua code block.
func writeExamples(b *strings.Builder, examples []Example) {
	if len(examples) == 0 {
		return
	}
	b.WriteString("## Examples\n\n")
	b.WriteString("These programs ship with the application (`script/examples`) and run unchanged " +
		"from the CLI, the GUI console, or `scripts.run`.\n\n")
	for _, ex := range examples {
		b.WriteString("### " + ex.Name + "\n\n")
		if ex.Description != "" {
			b.WriteString(ex.Description + "\n\n")
		}
		b.WriteString("```lua\n" + strings.TrimRight(ex.Source, "\n") + "\n```\n\n")
	}
}

// itoa is a tiny non-allocating-ish int→string for the few counts in the prose.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
