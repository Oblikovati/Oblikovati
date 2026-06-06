// SPDX-License-Identifier: GPL-2.0-only

// Package schema maps a Part 21 FILE_SCHEMA identifier to the application protocol
// (AP203/214/242) and back. The reader/writer use it to select the additive
// per-AP passes; the core geometry path is shared across all three (M17 plan §4).
package schema

import (
	"fmt"
	"strings"
)

// ApProtocol identifies which STEP application protocol a file conforms to.
type ApProtocol uint8

const (
	// ApUnknown is an unrecognized or absent schema.
	ApUnknown ApProtocol = iota
	// AP203 is config-controlled 3D design (CONFIG_CONTROL_DESIGN).
	AP203
	// AP214 is automotive design (assemblies, colours, layers).
	AP214
	// AP242 is managed model-based 3D engineering (supersedes 203/214; PMI, tessellation).
	AP242
)

// schemaTokens maps the substring that identifies each AP to its protocol. AP242's
// schema name embeds a version suffix, so a substring match is the robust test.
var schemaTokens = []struct {
	token string
	ap    ApProtocol
}{
	{"AP242", AP242},
	{"AUTOMOTIVE_DESIGN", AP214},
	{"CONFIG_CONTROL_DESIGN", AP203},
}

// Detect returns the protocol named by any of the FILE_SCHEMA identifiers, or
// ApUnknown when none match.
//
// Example:
//
//	ap := schema.Detect([]string{"CONFIG_CONTROL_DESIGN"}) // AP203
func Detect(identifiers []string) ApProtocol {
	for _, id := range identifiers {
		upper := strings.ToUpper(id)
		for _, st := range schemaTokens {
			if strings.Contains(upper, st.token) {
				return st.ap
			}
		}
	}
	return ApUnknown
}

// SchemaIdentifier returns the canonical FILE_SCHEMA string for an AP, used by the
// exporter. AP203 export uses CONFIG_CONTROL_DESIGN (the baseline target, plan §5).
func SchemaIdentifier(ap ApProtocol) (string, error) {
	switch ap {
	case AP203:
		return "CONFIG_CONTROL_DESIGN", nil
	case AP214:
		return "AUTOMOTIVE_DESIGN { 1 0 10303 214 1 1 1 1 }", nil
	case AP242:
		return "AP242_MANAGED_MODEL_BASED_3D_ENGINEERING_MIM_LF { 1 0 10303 442 1 1 4 }", nil
	default:
		return "", fmt.Errorf("schema: no canonical identifier for protocol %d", ap)
	}
}

// String renders the protocol for diagnostics.
func (a ApProtocol) String() string {
	switch a {
	case AP203:
		return "AP203"
	case AP214:
		return "AP214"
	case AP242:
		return "AP242"
	default:
		return "unknown"
	}
}
