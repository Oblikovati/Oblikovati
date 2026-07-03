// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// Custom parameter groups over the wire (M02-F05, Oblikovati#604):
// parameters.groups.list/add/delete/setDisplayName/addMember/removeMember.
// Mutations delegate to the Session verbs the head UI also uses, so both
// drivers share one seam (#1612, audit B1); undo/replication ride the central
// MutatingMethod seam.

// parameterGroupInfo marshals one group with its member names in collection order.
func parameterGroupInfo(holder compdef.ParameterHolder, g *param.ParameterGroup) wire.ParameterGroupInfo {
	ps := holder.Parameters()
	return wire.ParameterGroupInfo{
		InternalName: g.InternalName(), DisplayName: g.DisplayName(), ClientID: g.ClientID,
		Members: paramNames(ps, ps.GroupMembers(g.InternalName())),
	}
}

// groupByKey resolves one group on the active part or assembly, naming the method in the
// not-found error.
func groupByKey(s *app.Session, method, key string) (compdef.ParameterHolder, *param.ParameterGroup, error) {
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		return nil, nil, err
	}
	g, ok := holder.Parameters().GroupByKey(key)
	if !ok {
		return nil, nil, fmt.Errorf("%s: no parameter group named %q", method, key)
	}
	return holder, g, nil
}

// listParameterGroups returns the custom groups with their members, in creation order.
func listParameterGroups(_ *app.Session, holder compdef.ParameterHolder) (wire.ListParameterGroupsResult, error) {
	var out wire.ListParameterGroupsResult
	for _, g := range holder.Parameters().Groups() {
		out.Groups = append(out.Groups, parameterGroupInfo(holder, g))
	}
	return out, nil
}

// addParameterGroup creates an empty group keyed by its immutable internal name.
func addParameterGroup(s *app.Session, in wire.ParameterGroupAddArgs) (wire.ParameterGroupInfo, error) {
	g, err := s.AddParameterGroup(in.InternalName, in.DisplayName, in.ClientID)
	if err != nil {
		return wire.ParameterGroupInfo{}, err
	}
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		return wire.ParameterGroupInfo{}, err
	}
	return parameterGroupInfo(holder, g), nil
}

// deleteParameterGroup removes a group; the deleteParameters flag opts into
// the cascade that also deletes the member parameters. The cascade's
// may-not-sicken-a-survivor refusal comes from the aggregate (#1612).
func deleteParameterGroup(s *app.Session, in wire.ParameterGroupDeleteArgs) (struct{}, error) {
	if err := s.DeleteParameterGroup(in.InternalName, in.DeleteParameters); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

// setParameterGroupDisplayName edits the editable half of the group's naming;
// the non-empty rule comes from the aggregate, shared with the UI (#1612).
func setParameterGroupDisplayName(s *app.Session, in wire.ParameterGroupDisplayNameArgs) (wire.ParameterGroupInfo, error) {
	if err := s.RenameParameterGroup(in.InternalName, in.DisplayName); err != nil {
		return wire.ParameterGroupInfo{}, err
	}
	holder, g, err := groupByKey(s, wire.MethodParametersGroupsSetDisplayName, in.InternalName)
	if err != nil {
		return wire.ParameterGroupInfo{}, err
	}
	return parameterGroupInfo(holder, g), nil
}

// addParameterGroupMember / removeParameterGroupMember manage one membership.
// The group's existence was checked by the caller, so AddParameterToGroup's
// create-on-first-use (a UI journey) can never trigger on the wire path.
func addParameterGroupMember(s *app.Session, in wire.ParameterGroupMemberArgs) (wire.ParameterGroupInfo, error) {
	return editParameterGroupMember(s, wire.MethodParametersGroupsAddMember, in, (*app.Session).AddParameterToGroup)
}

func removeParameterGroupMember(s *app.Session, in wire.ParameterGroupMemberArgs) (wire.ParameterGroupInfo, error) {
	return editParameterGroupMember(s, wire.MethodParametersGroupsRemoveMember, in, (*app.Session).DetachParameterFromGroup)
}

// editParameterGroupMember resolves the group and the parameter (wire-strict:
// both must exist), applies one membership edit through the shared Session
// verb, and returns the updated group.
func editParameterGroupMember(s *app.Session, method string, in wire.ParameterGroupMemberArgs, edit func(*app.Session, param.ID, string) error) (wire.ParameterGroupInfo, error) {
	holder, g, err := groupByKey(s, method, in.InternalName)
	if err != nil {
		return wire.ParameterGroupInfo{}, err
	}
	p, ok := holder.Parameters().ByName(in.Parameter)
	if !ok {
		return wire.ParameterGroupInfo{}, fmt.Errorf("%s: no parameter named %q", method, in.Parameter)
	}
	if err := edit(s, p.ID(), in.InternalName); err != nil {
		return wire.ParameterGroupInfo{}, err
	}
	return parameterGroupInfo(holder, g), nil
}
