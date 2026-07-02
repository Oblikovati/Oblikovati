// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"strconv"
	"strings"
)

// Custom parameter groups (the reference API's CustomParameterGroups, M02-F05,
// Oblikovati#604): named, add-in-attributable views over the parameter set. A
// group is identified for its whole life by an immutable, locale-stable
// internal name; the display name is editable. Membership never affects
// parameter semantics — a parameter may belong to several groups, leaving one
// touches nothing else, and deleting a parameter detaches it everywhere.

// ParameterGroup is one custom group record. The internal name is fixed at
// creation; the display name is editable through [ParameterGroup.SetDisplayName]
// and the membership lives beside it. ClientID records which add-in created the
// group (empty for user-created groups).
type ParameterGroup struct {
	internalName string
	displayName  string
	ClientID     string
}

// InternalName returns the immutable key the group is addressed by.
func (g *ParameterGroup) InternalName() string { return g.internalName }

// DisplayName returns the group's editable presentation name.
func (g *ParameterGroup) DisplayName() string { return g.displayName }

// SetDisplayName edits the presentation name. An empty name is refused here, on
// the aggregate, so the rule holds for every driver — it used to live only in
// the wire handler while the UI path skipped it (#1612, audit B1).
func (g *ParameterGroup) SetDisplayName(name string) error {
	if name == "" {
		return fmt.Errorf("param: display name for group %q must not be empty", g.internalName)
	}
	g.displayName = name
	return nil
}

// Groups returns the custom groups in creation order.
func (ps *Parameters) Groups() []*ParameterGroup {
	return append([]*ParameterGroup(nil), ps.groups...)
}

// GroupByKey returns the group with the given internal name.
func (ps *Parameters) GroupByKey(key string) (*ParameterGroup, bool) {
	for _, g := range ps.groups {
		if g.internalName == key {
			return g, true
		}
	}
	return nil, false
}

// AddGroup creates a new, empty custom group. The internal name must be unique
// and non-empty; an empty display name defaults to it.
func (ps *Parameters) AddGroup(internalName, displayName, clientID string) (*ParameterGroup, error) {
	if internalName == "" {
		return nil, fmt.Errorf("param: group internal name must not be empty")
	}
	if _, exists := ps.GroupByKey(internalName); exists {
		return nil, fmt.Errorf("param: a group named %q already exists", internalName)
	}
	if displayName == "" {
		displayName = internalName
	}
	g := &ParameterGroup{internalName: internalName, displayName: displayName, ClientID: clientID}
	ps.groups = append(ps.groups, g)
	return g, nil
}

// DeleteGroup removes a group. deleteParameters opts into the cascade that
// also deletes the member parameters; without it the members stay, only the
// group goes. The cascade may not sicken a survivor: it is refused when a
// member is still read by a parameter outside the doomed set, or is a model
// parameter whose owning feature dimension survives (#1612, audit B1 — the
// same invariant as [Parameters.Delete], stated once for the batch).
func (ps *Parameters) DeleteGroup(key string, deleteParameters bool) error {
	if _, ok := ps.GroupByKey(key); !ok {
		return fmt.Errorf(errNoGroup, key)
	}
	members := ps.GroupMembers(key)
	if deleteParameters {
		if err := ps.refuseCascadeBlockers(key, members); err != nil {
			return err
		}
	}
	for _, id := range members {
		delete(ps.memberships[id], key)
	}
	if deleteParameters {
		ps.deleteMembers(members)
	}
	ps.dropGroup(key)
	return nil
}

// refuseCascadeBlockers rejects a member cascade that would leave a surviving
// parameter (or a feature dimension) referencing a deleted member.
func (ps *Parameters) refuseCascadeBlockers(key string, members []ID) error {
	doomed := map[ID]bool{}
	for _, id := range members {
		doomed[id] = true
	}
	for _, id := range members {
		// GroupMembers walks ps.order, so every member is live.
		p := ps.byID[id]
		if blockers := ps.survivorBlockers(p, doomed); len(blockers) > 0 {
			return fmt.Errorf("param: cannot cascade-delete group %q: member %q is in use by [%s]; remove those references first",
				key, p.name, strings.Join(blockers, ", "))
		}
	}
	return nil
}

// survivorBlockers names what outside the doomed set still reads p: dependents
// that are not themselves being deleted, or p's owning feature dimension.
func (ps *Parameters) survivorBlockers(p *Parameter, doomed map[ID]bool) []string {
	var names []string
	for _, dep := range ps.Dependents(p.id) {
		if d, ok := ps.byID[dep]; ok && !doomed[dep] {
			names = append(names, d.name)
		}
	}
	if len(names) == 0 && p.kind == ModelParam {
		names = []string{"its feature dimension"}
	}
	return names
}

// deleteMembers removes the cascade-deleted member parameters (a member may
// already be gone when an earlier removal cascaded — skip it).
func (ps *Parameters) deleteMembers(members []ID) {
	for _, id := range members {
		if p, ok := ps.byID[id]; ok {
			ps.remove(p)
		}
	}
}

// AddToGroup adds a parameter to an existing group. Membership in other groups
// is untouched.
func (ps *Parameters) AddToGroup(id ID, key string) error {
	if _, ok := ps.byID[id]; !ok {
		return fmt.Errorf(errNoParameter, id)
	}
	if _, ok := ps.GroupByKey(key); !ok {
		return fmt.Errorf(errNoGroup, key)
	}
	if ps.memberships[id] == nil {
		ps.memberships[id] = map[string]bool{}
	}
	ps.memberships[id][key] = true
	return nil
}

// RemoveFromGroup detaches a parameter from one group; the parameter itself
// and its other memberships are kept.
func (ps *Parameters) RemoveFromGroup(id ID, key string) error {
	if _, ok := ps.byID[id]; !ok {
		return fmt.Errorf(errNoParameter, id)
	}
	if _, ok := ps.GroupByKey(key); !ok {
		return fmt.Errorf(errNoGroup, key)
	}
	delete(ps.memberships[id], key)
	return nil
}

// RemoveFromAllGroups detaches a parameter from every group it belongs to.
func (ps *Parameters) RemoveFromAllGroups(id ID) error {
	if _, ok := ps.byID[id]; !ok {
		return fmt.Errorf(errNoParameter, id)
	}
	delete(ps.memberships, id)
	return nil
}

// GroupsOf returns the internal names of the groups a parameter belongs to,
// in group creation order (empty when ungrouped).
func (ps *Parameters) GroupsOf(id ID) []string {
	var out []string
	for _, g := range ps.groups {
		if ps.memberships[id][g.internalName] {
			out = append(out, g.internalName)
		}
	}
	return out
}

// GroupMembers returns the ids of the parameters in a group, in collection order.
func (ps *Parameters) GroupMembers(key string) []ID {
	var out []ID
	for _, id := range ps.order {
		if ps.memberships[id][key] {
			out = append(out, id)
		}
	}
	return out
}

func (ps *Parameters) dropGroup(key string) {
	for i, g := range ps.groups {
		if g.internalName == key {
			ps.groups = append(ps.groups[:i], ps.groups[i+1:]...)
			return
		}
	}
}

// CopyToUser duplicates a parameter as a new user parameter named "<name>_copy" (with a
// numeric suffix if needed), carrying over its value, comment, key, export, and any
// multi-value list. The copy is independent of the source.
func (ps *Parameters) CopyToUser(id ID) (*Parameter, error) {
	src, ok := ps.byID[id]
	if !ok {
		return nil, fmt.Errorf(errNoParameter, id)
	}
	copyParam, err := ps.addCopyValue(src)
	if err != nil {
		return nil, err
	}
	copyParam.Comment = src.Comment
	copyParam.IsKey = src.IsKey
	copyParam.ExposedAsProperty = src.ExposedAsProperty
	copyParam.Precision = src.Precision
	copyParam.tol = src.tol
	_ = copyParam.SetExpressionList(src.exprList, src.allowCustom)
	return copyParam, nil
}

// addCopyValue creates the user-parameter copy with the source's value, routed by flavor.
func (ps *Parameters) addCopyValue(src *Parameter) (*Parameter, error) {
	name := ps.uniqueCopyName(src.name)
	switch {
	case src.IsText():
		return ps.AddTextUserParameter(name, src.text)
	case src.IsBoolean():
		return ps.AddBooleanUserParameter(name, src.Bool())
	default:
		return ps.AddUserParameter(name, src.Expression())
	}
}

// uniqueCopyName returns "<base>_copy", appending a counter until the name is free.
func (ps *Parameters) uniqueCopyName(base string) string {
	name := base + "_copy"
	for n := 2; ; n++ {
		if _, taken := ps.byName[name]; !taken {
			return name
		}
		name = base + "_copy" + strconv.Itoa(n)
	}
}
