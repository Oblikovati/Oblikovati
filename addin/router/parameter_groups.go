// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
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
func listParameterGroups(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		return nil, err
	}
	var out wire.ListParameterGroupsResult
	for _, g := range holder.Parameters().Groups() {
		out.Groups = append(out.Groups, parameterGroupInfo(holder, g))
	}
	return json.Marshal(out)
}

// addParameterGroup creates an empty group keyed by its immutable internal name.
func addParameterGroup(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterGroupAddArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	g, err := s.AddParameterGroup(in.InternalName, in.DisplayName, in.ClientID)
	if err != nil {
		return nil, err
	}
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(parameterGroupInfo(holder, g))
}

// deleteParameterGroup removes a group; the deleteParameters flag opts into
// the cascade that also deletes the member parameters. The cascade's
// may-not-sicken-a-survivor refusal comes from the aggregate (#1612).
func deleteParameterGroup(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterGroupDeleteArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if err := s.DeleteParameterGroup(in.InternalName, in.DeleteParameters); err != nil {
		return nil, err
	}
	return json.Marshal(struct{}{})
}

// setParameterGroupDisplayName edits the editable half of the group's naming;
// the non-empty rule comes from the aggregate, shared with the UI (#1612).
func setParameterGroupDisplayName(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterGroupDisplayNameArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if err := s.RenameParameterGroup(in.InternalName, in.DisplayName); err != nil {
		return nil, err
	}
	holder, g, err := groupByKey(s, wire.MethodParametersGroupsSetDisplayName, in.InternalName)
	if err != nil {
		return nil, err
	}
	return json.Marshal(parameterGroupInfo(holder, g))
}

// addParameterGroupMember / removeParameterGroupMember manage one membership.
// The group's existence was checked by the caller, so AddParameterToGroup's
// create-on-first-use (a UI journey) can never trigger on the wire path.
func addParameterGroupMember(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	return editParameterGroupMember(s, wire.MethodParametersGroupsAddMember, args, (*app.Session).AddParameterToGroup)
}

func removeParameterGroupMember(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	return editParameterGroupMember(s, wire.MethodParametersGroupsRemoveMember, args, (*app.Session).DetachParameterFromGroup)
}

// editParameterGroupMember resolves the group and the parameter (wire-strict:
// both must exist), applies one membership edit through the shared Session
// verb, and returns the updated group.
func editParameterGroupMember(s *app.Session, method string, args json.RawMessage, edit func(*app.Session, param.ID, string) error) (json.RawMessage, error) {
	var in wire.ParameterGroupMemberArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	holder, g, err := groupByKey(s, method, in.InternalName)
	if err != nil {
		return nil, err
	}
	p, ok := holder.Parameters().ByName(in.Parameter)
	if !ok {
		return nil, fmt.Errorf("%s: no parameter named %q", method, in.Parameter)
	}
	if err := edit(s, p.ID(), in.InternalName); err != nil {
		return nil, err
	}
	return json.Marshal(parameterGroupInfo(holder, g))
}
