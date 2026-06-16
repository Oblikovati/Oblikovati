// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

// FlatPatternSettings is the per-part flat-pattern settings (M13-F05, #635). DeferUpdate
// suppresses the automatic flat-pattern recompute, so a heavy flat is developed only on
// demand (when a drawing view or an export asks for it).
type FlatPatternSettings struct {
	DeferUpdate bool
}
