// SPDX-License-Identifier: GPL-2.0-only

package part21

import "fmt"

// Header holds the three mandatory Part 21 HEADER records. Optional/list fields are
// preserved so the emitter can round-trip them. See ISO 10303-21 §8.
type Header struct {
	// FILE_DESCRIPTION
	Description         []string
	ImplementationLevel string
	// FILE_NAME
	Name                string
	TimeStamp           string
	Author              []string
	Organization        []string
	PreprocessorVersion string
	OriginatingSystem   string
	Authorization       string
	// FILE_SCHEMA
	SchemaIdentifiers []string
}

// parseHeader parses HEADER … ENDSEC;. The standard records are FILE_DESCRIPTION,
// FILE_NAME, FILE_SCHEMA, but exporters vary: OpenCASCADE writes FILE_NAME first, and some
// add optional records (FILE_POPULATION, SECTION_*). So records are read in ANY order and
// any unrecognized one is skipped, rather than demanding a fixed sequence.
func (p *parser) parseHeader() (Header, error) {
	if err := p.expectKeyword("HEADER"); err != nil {
		return Header{}, err
	}
	if err := p.expect(TokSemicolon); err != nil {
		return Header{}, err
	}
	var h Header
	for p.cur.Kind == TokKeyword && p.cur.Text != "ENDSEC" {
		if err := p.parseHeaderRecord(&h); err != nil {
			return Header{}, err
		}
	}
	return h, p.endSection()
}

// parseHeaderRecord reads one HEADER record by its keyword, skipping any record this
// importer does not consume so an optional/vendor record never aborts the import.
func (p *parser) parseHeaderRecord(h *Header) error {
	switch p.cur.Text {
	case "FILE_DESCRIPTION":
		return p.parseFileDescription(h)
	case "FILE_NAME":
		return p.parseFileName(h)
	case "FILE_SCHEMA":
		return p.parseFileSchema(h)
	default:
		return p.skipHeaderRecord()
	}
}

// skipHeaderRecord consumes an unrecognized `KEYWORD( … );` record verbatim.
func (p *parser) skipHeaderRecord() error {
	if err := p.advance(); err != nil { // past the keyword
		return err
	}
	if _, err := p.parseParamList(); err != nil {
		return err
	}
	return p.expect(TokSemicolon)
}

// headerRecord consumes `KEYWORD( … );` and returns its parameter list.
func (p *parser) headerRecord(keyword string) ([]Value, error) {
	if err := p.expectKeyword(keyword); err != nil {
		return nil, err
	}
	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}
	return params, p.expect(TokSemicolon)
}

// parseFileDescription reads FILE_DESCRIPTION((descr…), impl_level).
func (p *parser) parseFileDescription(h *Header) error {
	params, err := p.headerRecord("FILE_DESCRIPTION")
	if err != nil {
		return err
	}
	if len(params) != 2 {
		return fmt.Errorf("part21: FILE_DESCRIPTION wants 2 params, got %d", len(params))
	}
	h.Description, err = stringList(params[0])
	if err != nil {
		return err
	}
	h.ImplementationLevel, err = optString(params[1])
	return err
}

// parseFileName reads the 7-field FILE_NAME record.
func (p *parser) parseFileName(h *Header) error {
	params, err := p.headerRecord("FILE_NAME")
	if err != nil {
		return err
	}
	if len(params) != 7 {
		return fmt.Errorf("part21: FILE_NAME wants 7 params, got %d", len(params))
	}
	return assignFileName(h, params)
}

// assignFileName copies the seven FILE_NAME fields into the header.
func assignFileName(h *Header, params []Value) error {
	var err error
	if h.Name, err = optString(params[0]); err != nil {
		return err
	}
	if h.TimeStamp, err = optString(params[1]); err != nil {
		return err
	}
	if h.Author, err = stringList(params[2]); err != nil {
		return err
	}
	if h.Organization, err = stringList(params[3]); err != nil {
		return err
	}
	return assignFileNameTail(h, params)
}

// assignFileNameTail copies the last three FILE_NAME string fields.
func assignFileNameTail(h *Header, params []Value) error {
	var err error
	if h.PreprocessorVersion, err = optString(params[4]); err != nil {
		return err
	}
	if h.OriginatingSystem, err = optString(params[5]); err != nil {
		return err
	}
	h.Authorization, err = optString(params[6])
	return err
}

// parseFileSchema reads FILE_SCHEMA((schema…)).
func (p *parser) parseFileSchema(h *Header) error {
	params, err := p.headerRecord("FILE_SCHEMA")
	if err != nil {
		return err
	}
	if len(params) != 1 {
		return fmt.Errorf("part21: FILE_SCHEMA wants 1 param, got %d", len(params))
	}
	h.SchemaIdentifiers, err = stringList(params[0])
	return err
}

// stringList decodes a ( 'a', 'b', … ) value into []string.
func stringList(v Value) ([]string, error) {
	items, err := v.AsList()
	if err != nil {
		return nil, err
	}
	out := make([]string, len(items))
	for i, item := range items {
		if out[i], err = item.AsString(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// optString decodes a string or the '$' null marker (→ empty string).
func optString(v Value) (string, error) {
	if v.IsNull() {
		return "", nil
	}
	return v.AsString()
}
