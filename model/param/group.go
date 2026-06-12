// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"strconv"
)

// Custom parameter groups (the reference API's CustomParameterGroups, M02-F05,
// Oblikovati#604): named, add-in-attributable views over the parameter set. A
// group is identified for its whole life by an immutable, locale-stable
// internal name; the display name is editable. Membership never affects
// parameter semantics — a parameter may belong to several groups, leaving one
// touches nothing else, and deleting a parameter detaches it everywhere.

// ParameterGroup is one custom group record. The internal name is fixed at
// creation; DisplayName and the membership live beside it. ClientID records
// which add-in created the group (empty for user-created groups).
type ParameterGroup struct {
	internalName string
	DisplayName  string
	ClientID     string
}

// InternalName returns the immutable key the group is addressed by.
func (g *ParameterGroup) InternalName() string { return g.internalName }

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
	g := &ParameterGroup{internalName: internalName, DisplayName: displayName, ClientID: clientID}
	ps.groups = append(ps.groups, g)
	return g, nil
}

// DeleteGroup removes a group. deleteParameters opts into the cascade that
// also deletes the member parameters; without it the members stay, only the
// group goes.
func (ps *Parameters) DeleteGroup(key string, deleteParameters bool) error {
	if _, ok := ps.GroupByKey(key); !ok {
		return fmt.Errorf("param: no group named %q", key)
	}
	members := ps.GroupMembers(key)
	for _, id := range members {
		delete(ps.memberships[id], key)
	}
	if deleteParameters {
		ps.deleteMembers(members)
	}
	ps.dropGroup(key)
	return nil
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
		return fmt.Errorf("param: no parameter with id %d", id)
	}
	if _, ok := ps.GroupByKey(key); !ok {
		return fmt.Errorf("param: no group named %q", key)
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
		return fmt.Errorf("param: no parameter with id %d", id)
	}
	if _, ok := ps.GroupByKey(key); !ok {
		return fmt.Errorf("param: no group named %q", key)
	}
	delete(ps.memberships[id], key)
	return nil
}

// RemoveFromAllGroups detaches a parameter from every group it belongs to.
func (ps *Parameters) RemoveFromAllGroups(id ID) error {
	if _, ok := ps.byID[id]; !ok {
		return fmt.Errorf("param: no parameter with id %d", id)
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
		return nil, fmt.Errorf("param: no parameter with id %d", id)
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
