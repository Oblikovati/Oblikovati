// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"encoding/xml"
	"fmt"
)

// Parameter-set XML exchange (M02-F07, Oblikovati#606): user parameters
// round-trip as a flat, documented XML so spreadsheet/PDM tooling can edit
// them outside the application. Model/reference parameters never export —
// they are owned by features and have no life outside the document.
//
// The format, one element per parameter:
//
//	<parameters>
//	  <parameter name="len" expression="10 mm" comment="the length" isKey="true" exposedAsProperty="true"/>
//	  <parameter name="material" valueType="text" text="steel"/>
//	  <parameter name="vented" valueType="boolean" bool="true"/>
//	</parameters>

// parameterSetXML is the document element of the exchange format.
type parameterSetXML struct {
	XMLName    xml.Name            `xml:"parameters"`
	Parameters []parameterEntryXML `xml:"parameter"`
}

// parameterEntryXML is one exchanged parameter. ValueType is empty for
// numeric, "text" or "boolean" otherwise (the parameterRecipe spellings).
type parameterEntryXML struct {
	Name              string `xml:"name,attr"`
	ValueType         string `xml:"valueType,attr,omitempty"`
	Expression        string `xml:"expression,attr,omitempty"`
	Text              string `xml:"text,attr,omitempty"`
	Bool              bool   `xml:"bool,attr,omitempty"`
	Comment           string `xml:"comment,attr,omitempty"`
	IsKey             bool   `xml:"isKey,attr,omitempty"`
	ExposedAsProperty bool   `xml:"exposedAsProperty,attr,omitempty"`
}

// ExportXML renders the user parameters as the exchange XML.
func (ps *Parameters) ExportXML() (string, error) {
	var set parameterSetXML
	for _, p := range ps.All() {
		if p.Kind() != UserParam {
			continue
		}
		set.Parameters = append(set.Parameters, exportEntry(p))
	}
	out, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return "", fmt.Errorf("param: marshal parameter set: %w", err)
	}
	return string(out), nil
}

// exportEntry projects one user parameter into its exchange element.
func exportEntry(p *Parameter) parameterEntryXML {
	e := parameterEntryXML{
		Name: p.Name(), Comment: p.Comment,
		IsKey: p.IsKey, ExposedAsProperty: p.ExposedAsProperty,
	}
	switch {
	case p.IsText():
		e.ValueType, e.Text = "text", p.Text()
	case p.IsBoolean():
		e.ValueType, e.Bool = "boolean", p.Bool()
	default:
		e.Expression = p.Expression()
	}
	return e
}

// ImportXML applies an exchange document: unknown names are created as user
// parameters, known names updated in place. The set is structurally validated
// first; the first invalid entry rejects the import naming the offending
// value. Expression errors surface as they apply — run the import against a
// snapshot the caller can restore (app.Session.ImportParameters does).
func (ps *Parameters) ImportXML(doc string) (added, updated int, err error) {
	var set parameterSetXML
	if err := xml.Unmarshal([]byte(doc), &set); err != nil {
		return 0, 0, fmt.Errorf("param: parameter set is not valid XML: %w", err)
	}
	if err := validateEntries(set.Parameters); err != nil {
		return 0, 0, err
	}
	for _, e := range set.Parameters {
		wasNew, err := ps.importEntry(e)
		if err != nil {
			return added, updated, fmt.Errorf("param: import %q: %w", e.Name, err)
		}
		if wasNew {
			added++
		} else {
			updated++
		}
	}
	return added, updated, nil
}

// validateEntries rejects structurally bad entries before anything mutates.
func validateEntries(entries []parameterEntryXML) error {
	seen := map[string]bool{}
	for _, e := range entries {
		if err := validateEntry(e); err != nil {
			return err
		}
		if seen[e.Name] {
			return fmt.Errorf("param: parameter %q appears twice in the import set", e.Name)
		}
		seen[e.Name] = true
	}
	return nil
}

// validateEntry checks one entry's shape: a name, a known value type, and a
// value matching that type.
func validateEntry(e parameterEntryXML) error {
	if e.Name == "" {
		return fmt.Errorf("param: import entry without a name (expression %q)", e.Expression)
	}
	switch e.ValueType {
	case "":
		if e.Expression == "" {
			return fmt.Errorf("param: numeric import entry %q needs an expression", e.Name)
		}
	case "text", "boolean":
	default:
		return fmt.Errorf("param: import entry %q has unknown value type %q (want text|boolean or empty for numeric)", e.Name, e.ValueType)
	}
	return nil
}

// importEntry creates or updates one parameter from its exchange element,
// reporting whether it was newly created.
func (ps *Parameters) importEntry(e parameterEntryXML) (wasNew bool, err error) {
	p, exists := ps.ByName(e.Name)
	if !exists {
		if p, err = ps.addImported(e); err != nil {
			return false, err
		}
	} else if err = applyImportedValue(p, e); err != nil {
		return false, err
	}
	p.Comment, p.IsKey, p.ExposedAsProperty = e.Comment, e.IsKey, e.ExposedAsProperty
	return !exists, nil
}

// addImported creates the user parameter for one entry, routed by value type.
func (ps *Parameters) addImported(e parameterEntryXML) (*Parameter, error) {
	switch e.ValueType {
	case "text":
		return ps.AddTextUserParameter(e.Name, e.Text)
	case "boolean":
		return ps.AddBooleanUserParameter(e.Name, e.Bool)
	default:
		return ps.AddUserParameter(e.Name, e.Expression)
	}
}

// applyImportedValue updates an existing parameter's value from one entry; the
// entry's flavor must match the parameter's.
func applyImportedValue(p *Parameter, e parameterEntryXML) error {
	switch e.ValueType {
	case "text":
		return p.SetText(e.Text)
	case "boolean":
		return p.SetBool(e.Bool)
	default:
		return p.SetExpression(e.Expression)
	}
}
