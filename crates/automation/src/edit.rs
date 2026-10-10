// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    collection, decode, encode, groups_edit, parameter_edit, parameter_value, settings_edit,
};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::{history::RecipeReservations, parameter::ParameterCollection};

/// Implemented edits; queries are enumerated by `QUERY_METHODS`.
pub const EDIT_METHODS: [&str; 11] = [
    METHOD_PARAMETERS_SET_SETTINGS,
    METHOD_PARAMETERS_ADD,
    METHOD_PARAMETERS_SET,
    METHOD_PARAMETERS_RENAME,
    METHOD_PARAMETERS_CONVERT,
    METHOD_PARAMETERS_DELETE,
    METHOD_PARAMETERS_GROUPS_ADD,
    METHOD_PARAMETERS_GROUPS_DELETE,
    METHOD_PARAMETERS_GROUPS_SET_DISPLAY_NAME,
    METHOD_PARAMETERS_GROUPS_ADD_MEMBER,
    METHOD_PARAMETERS_GROUPS_REMOVE_MEMBER,
];

pub(super) enum PreparedEdit {
    Settings(ParameterSettingsUpdateArgs),
    Add(ParameterSetArgs),
    Set(ParameterSetArgs),
    Rename(ParameterRenameArgs),
    Convert(ParameterConvertArgs),
    Delete(ParameterNameArgs),
    GroupAdd(ParameterGroupAddArgs),
    GroupDelete(ParameterGroupDeleteArgs),
    GroupRename(ParameterGroupDisplayNameArgs),
    GroupMember(ParameterGroupMemberArgs, bool),
}

impl PreparedEdit {
    pub(super) fn decode(method: &str, request: &[u8]) -> Result<Self, Error> {
        match method {
            METHOD_PARAMETERS_ADD => Ok(Self::Add(decode(method, request)?)),
            METHOD_PARAMETERS_SET => Ok(Self::Set(decode(method, request)?)),
            METHOD_PARAMETERS_RENAME => Ok(Self::Rename(decode(method, request)?)),
            METHOD_PARAMETERS_CONVERT => Ok(Self::Convert(decode(method, request)?)),
            METHOD_PARAMETERS_DELETE => Ok(Self::Delete(decode(method, request)?)),
            METHOD_PARAMETERS_SET_SETTINGS => Ok(Self::Settings(decode(method, request)?)),
            METHOD_PARAMETERS_GROUPS_ADD => Ok(Self::GroupAdd(decode(method, request)?)),
            METHOD_PARAMETERS_GROUPS_DELETE => Ok(Self::GroupDelete(decode(method, request)?)),
            METHOD_PARAMETERS_GROUPS_SET_DISPLAY_NAME => {
                Ok(Self::GroupRename(decode(method, request)?))
            }
            METHOD_PARAMETERS_GROUPS_ADD_MEMBER => {
                Ok(Self::GroupMember(decode(method, request)?, true))
            }
            METHOD_PARAMETERS_GROUPS_REMOVE_MEMBER => {
                Ok(Self::GroupMember(decode(method, request)?, false))
            }
            _ => Err(Error(format!(
                "method {method:?}: expected an edit from {EDIT_METHODS:?}"
            ))),
        }
    }

    pub(super) fn label(&self) -> &'static str {
        match self {
            Self::Add(_) => "Add Parameter",
            Self::Set(_) => "Edit Parameter",
            Self::Rename(_) => "Rename Parameter",
            Self::Convert(_) => "Convert Parameter",
            Self::Delete(_) => "Delete Parameter",
            Self::Settings(_) => "Edit Parameter Settings",
            Self::GroupAdd(_) => "Add Parameter Group",
            Self::GroupDelete(_) => "Delete Parameter Group",
            Self::GroupRename(_) => "Rename Parameter Group",
            Self::GroupMember(_, true) => "Edit Parameters",
            Self::GroupMember(_, false) => "Edit Parameter Group",
        }
    }

    pub(super) fn apply(
        self,
        parameters: &mut ParameterCollection<u64>,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<Vec<u8>, Error> {
        match self {
            Self::Add(request) => encode(&parameter_value::add(parameters, reservations, request)?),
            Self::Set(request) => encode(&parameter_value::set(parameters, request)?),
            Self::Rename(request) => encode(&parameter_edit::rename(parameters, request)?),
            Self::Convert(request) => encode(&parameter_edit::convert(parameters, request)?),
            Self::Delete(request) => parameter_edit::delete(parameters, request),
            Self::Settings(request) => {
                settings_edit::apply(parameters, request)?;
                encode(&collection::settings(parameters)?)
            }
            Self::GroupAdd(request) => encode(&groups_edit::add(parameters, request)?),
            Self::GroupDelete(request) => groups_edit::delete(parameters, request),
            Self::GroupRename(request) => encode(&groups_edit::rename(parameters, request)?),
            Self::GroupMember(request, add) => {
                encode(&groups_edit::membership(parameters, request, add)?)
            }
        }
    }
}
