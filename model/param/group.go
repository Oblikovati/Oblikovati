// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"strconv"
)

// Custom parameter groups (Inventor's CustomParameterGroups) let a user organize
// parameters into named groups. Membership is a simple parameter→group-name map; group
// names are kept in creation order for stable display. Deleting a group also deletes its
// member parameters (per the journey: "rename or delete a group (and its parameters)").

// Groups returns the custom group names in creation order.
func (ps *Parameters) Groups() []string {
	return append([]string(nil), ps.groupOrder...)
}

// AddGroup creates a new, empty custom group. It rejects an empty or duplicate name.
func (ps *Parameters) AddGroup(name string) error {
	if name == "" {
		return fmt.Errorf("param: group name must not be empty")
	}
	if ps.hasGroup(name) {
		return fmt.Errorf("param: a group named %q already exists", name)
	}
	ps.groupOrder = append(ps.groupOrder, name)
	return nil
}

// RenameGroup renames a group, retargeting its members. It rejects a clash with another
// group and an unknown source name.
func (ps *Parameters) RenameGroup(oldName, newName string) error {
	if !ps.hasGroup(oldName) {
		return fmt.Errorf("param: no group named %q", oldName)
	}
	if newName != oldName && ps.hasGroup(newName) {
		return fmt.Errorf("param: a group named %q already exists", newName)
	}
	for i, g := range ps.groupOrder {
		if g == oldName {
			ps.groupOrder[i] = newName
		}
	}
	for id, g := range ps.groupOf {
		if g == oldName {
			ps.groupOf[id] = newName
		}
	}
	return nil
}

// DeleteGroup removes a group and deletes every parameter that belonged to it.
func (ps *Parameters) DeleteGroup(name string) error {
	if !ps.hasGroup(name) {
		return fmt.Errorf("param: no group named %q", name)
	}
	for _, id := range ps.GroupMembers(name) {
		if p, ok := ps.byID[id]; ok {
			ps.remove(p)
		}
	}
	ps.dropGroupName(name)
	return nil
}

// AddToGroup assigns a parameter to a group, creating the group if it does not exist yet.
func (ps *Parameters) AddToGroup(id ID, name string) error {
	if _, ok := ps.byID[id]; !ok {
		return fmt.Errorf("param: no parameter with id %d", id)
	}
	if !ps.hasGroup(name) {
		if err := ps.AddGroup(name); err != nil {
			return err
		}
	}
	ps.groupOf[id] = name
	return nil
}

// RemoveFromGroup detaches a parameter from its group (the parameter itself is kept).
func (ps *Parameters) RemoveFromGroup(id ID) error {
	if _, ok := ps.byID[id]; !ok {
		return fmt.Errorf("param: no parameter with id %d", id)
	}
	delete(ps.groupOf, id)
	return nil
}

// GroupOf returns the group a parameter belongs to, or false when it is ungrouped.
func (ps *Parameters) GroupOf(id ID) (string, bool) {
	g, ok := ps.groupOf[id]
	return g, ok
}

// GroupMembers returns the ids of the parameters in a group, in collection order.
func (ps *Parameters) GroupMembers(name string) []ID {
	var out []ID
	for _, id := range ps.order {
		if ps.groupOf[id] == name {
			out = append(out, id)
		}
	}
	return out
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

func (ps *Parameters) hasGroup(name string) bool {
	for _, g := range ps.groupOrder {
		if g == name {
			return true
		}
	}
	return false
}

func (ps *Parameters) dropGroupName(name string) {
	for i, g := range ps.groupOrder {
		if g == name {
			ps.groupOrder = append(ps.groupOrder[:i], ps.groupOrder[i+1:]...)
			return
		}
	}
}
