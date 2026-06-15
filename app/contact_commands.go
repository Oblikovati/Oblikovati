// SPDX-License-Identifier: GPL-2.0-only

package app

import "fmt"

// The Contact panel (M12-F05, Oblikovati/Oblikovati#362/#368) on the Assemble tab: group the
// selected components into a contact set (they resist interpenetration when dragged), toggle
// the contact solver, and analyze interference (the overlapping volumes between components).

// contactCommands returns the Contact-panel commands.
func contactCommands() []*CommandDefinition {
	return []*CommandDefinition{
		contactCommand("Assembly.ContactSet", "Contact Set", "contact-set",
			"Group the selected components into a contact set — they resist interpenetration when the contact solver is on.", (*Session).CreateContactSet),
		contactCommand("Assembly.ContactEnable", "Enable Contact", "contact-enable",
			"Toggle the contact solver — whether moving a contact-set member stops at contact with another.", (*Session).ToggleContactSolver),
		contactCommand("Assembly.Interference", "Interference", "interference",
			"Analyze interference — report the overlapping volumes between the assembly's components.", (*Session).AnalyzeInterference),
	}
}

// contactCommand builds a Contact-panel command that runs run on an active assembly.
func contactCommand(id, name, icon, tooltip string, run func(*Session) error) *CommandDefinition {
	return NewCommand(id, name, "Contact", run).
		WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
		WithIcon(icon).WithButtonStyle(CompactIconButton).WithTooltip(tooltip)
}

// CreateContactSet groups the currently-selected component occurrences into a new contact set.
func (s *Session) CreateContactSet() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	cs := asm.ContactSolver().Create("")
	members := 0
	for _, it := range s.Selection().Items() {
		if h, ok := it.(OccurrenceHandle); ok {
			_ = asm.ContactSolver().AddMember(cs.ID(), h.Occurrence.ID())
			members++
		}
	}
	s.notice = fmt.Sprintf("Created %s with %d component(s)", cs.Name(), members)
	return nil
}

// ToggleContactSolver flips contact enforcement on or off.
func (s *Session) ToggleContactSolver() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	solver := asm.ContactSolver()
	solver.SetEnabled(!solver.Enabled())
	state := "disabled"
	if solver.Enabled() {
		state = "enabled"
	}
	s.notice = "Contact solver " + state
	return nil
}

// AnalyzeInterference reports the overlapping volumes between the assembly's components as a
// status notice.
func (s *Session) AnalyzeInterference() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	res := asm.AnalyzeInterference(nil)
	if len(res.Results) == 0 {
		s.notice = "No interference detected"
		return nil
	}
	s.notice = fmt.Sprintf("%d interference(s), total volume %.3f cm³", len(res.Results), res.Total)
	return nil
}
