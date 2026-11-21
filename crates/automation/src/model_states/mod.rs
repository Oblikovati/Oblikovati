// SPDX-License-Identifier: GPL-2.0-only
//! Canonical model-state operations over document-owned part members.
use oblikovati_api::{Error, wire::*};
use oblikovati_model::{
    history::{DocumentSnapshot, RecipeReservations},
    model_state::{ModelStateMember, ModelStateMembers},
};
mod parameter_projection;
mod parameter_rename;
mod property_edit;
mod property_projection;
mod property_view;
mod query;
pub(super) mod table_apply;
mod table_apply_values;
mod table_choices;
mod table_rows;
mod table_values;
mod table_verify;
mod table_view;
pub(super) use parameter_projection::record_edits as record_parameter_edits;
pub(super) use parameter_rename::plan as plan_parameter_rename;
pub(super) use property_projection::record_edit as record_property_edit;
pub(super) use query::list;
pub(super) use table_verify::verify as verify_table;
pub(super) use table_view::read as table;
pub(super) enum MemberEdit {
    Create(CreateModelStateArgs),
    Rename(ModelStateNameArgs),
    Copy(ModelStateNameArgs),
    Activate(RepRef),
    Delete(RepRef),
}
impl MemberEdit {
    pub(super) fn label(&self) -> &'static str {
        match self {
            Self::Create(_) => "Create Model State",
            Self::Rename(_) => "Rename Model State",
            Self::Copy(_) => "Duplicate Model State",
            Self::Activate(_) => "Activate Model State",
            Self::Delete(_) => "Delete Model State",
        }
    }

    pub(super) fn apply(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<Vec<u8>, Error> {
        let members = snapshot
            .model_states
            .as_mut()
            .ok_or_else(|| Error("model-state edit: expected an active part document".into()))?;
        let project = matches!(&self, Self::Activate(_) | Self::Delete(_));
        let response = match self {
            Self::Create(request) => create(members, ids, request),
            Self::Rename(request) => {
                members.rename(request.id, &request.name).map_err(Error)?;
                reply(members, request.id)
            }
            Self::Copy(request) => copy(members, ids, request),
            Self::Activate(request) => {
                members.activate(request.id).map_err(Error)?;
                reply(members, request.id)
            }
            Self::Delete(request) => {
                members.delete(request.id).map_err(Error)?;
                crate::encode(&member_list(members))
            }
        }?;
        if project {
            parameter_projection::project(snapshot)?;
            property_projection::project(snapshot)?;
        }
        Ok(response)
    }
}
fn create(
    members: &mut ModelStateMembers,
    ids: &mut RecipeReservations<'_>,
    request: CreateModelStateArgs,
) -> Result<Vec<u8>, Error> {
    if !request.design_view.is_empty()
        || !request.positional.is_empty()
        || !request.level_of_detail.is_empty()
    {
        return Err(Error(format!(
            "part model-state request {request:?}: expected no assembly representation references"
        )));
    }
    let reserved = ids
        .allocate_model_state()
        .map_err(|cause| Error(cause.to_string()))?;
    let name = if request.name.is_empty() {
        members.generated_name("Model State")
    } else {
        request.name
    };
    members.insert(reserved, &name).map_err(Error)?;
    reply(members, reserved)
}
fn copy(
    members: &mut ModelStateMembers,
    ids: &mut RecipeReservations<'_>,
    request: ModelStateNameArgs,
) -> Result<Vec<u8>, Error> {
    let source = members
        .list()
        .iter()
        .find(|member| member.id == request.id)
        .ok_or_else(|| {
            Error(format!(
                "model-state source {}: expected an existing member",
                request.id
            ))
        })?;
    let name = if request.name.is_empty() {
        members.generated_name(&format!("{} Copy", source.name))
    } else {
        request.name
    };
    let reserved = ids
        .allocate_model_state()
        .map_err(|cause| Error(cause.to_string()))?;
    members.copy(request.id, reserved, &name).map_err(Error)?;
    reply(members, reserved)
}
fn reply(members: &ModelStateMembers, id: u64) -> Result<Vec<u8>, Error> {
    let member = members
        .list()
        .iter()
        .find(|member| member.id == id)
        .ok_or_else(|| {
            Error(format!(
                "model-state ID {id}: expected an existing member for response"
            ))
        })?;
    crate::encode(&ModelStateResult {
        model_state: info(members, member),
    })
}
fn info(members: &ModelStateMembers, member: &ModelStateMember) -> ModelStateInfo {
    ModelStateInfo {
        id: member.id,
        name: member.name.clone(),
        primary: member.id == 1,
        active: members
            .active()
            .is_some_and(|active| active.id == member.id),
        ..Default::default()
    }
}

fn member_list(members: &ModelStateMembers) -> ModelStatesResult {
    ModelStatesResult {
        model_states: Some(
            members
                .list()
                .iter()
                .map(|member| info(members, member))
                .collect(),
        ),
    }
}
