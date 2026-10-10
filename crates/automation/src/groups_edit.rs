// SPDX-License-Identifier: GPL-2.0-only

use crate::{collection, parameters};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::parameter::ParameterCollection;

type Parameters = ParameterCollection<u64>;

pub(super) fn add(
    parameters: &mut Parameters,
    request: ParameterGroupAddArgs,
) -> Result<ParameterGroupInfo, Error> {
    parameters
        .add_group(
            &request.internal_name,
            &request.display_name,
            &request.client_id,
        )
        .map_err(|error| Error(error.to_string()))?;
    information(parameters, &request.internal_name)
}

pub(super) fn delete(
    parameters: &mut Parameters,
    request: ParameterGroupDeleteArgs,
) -> Result<Vec<u8>, Error> {
    parameters
        .delete_group(&request.internal_name, request.delete_parameters)
        .map_err(|error| Error(error.to_string()))?;
    // The canonical void response is an empty object, as in the Go C-ABI router.
    Ok(b"{}".to_vec())
}

pub(super) fn rename(
    parameters: &mut Parameters,
    request: ParameterGroupDisplayNameArgs,
) -> Result<ParameterGroupInfo, Error> {
    parameters
        .set_group_display_name(&request.internal_name, &request.display_name)
        .map_err(|error| Error(error.to_string()))?;
    information(parameters, &request.internal_name)
}

pub(super) fn membership(
    parameters: &mut Parameters,
    request: ParameterGroupMemberArgs,
    add: bool,
) -> Result<ParameterGroupInfo, Error> {
    // Wire membership is strict: unlike the UI journey, it must not create a missing group.
    require_group(parameters, &request.internal_name)?;
    let id = parameters::resolve(parameters, &request.parameter)?;
    let result = if add {
        parameters.add_to_group(id, &request.internal_name)
    } else {
        parameters.remove_from_group(id, &request.internal_name)
    };
    result.map_err(|error| Error(error.to_string()))?;
    information(parameters, &request.internal_name)
}

fn require_group<'a>(
    parameters: &'a Parameters,
    key: &str,
) -> Result<&'a oblikovati_model::parameter::ParameterGroup<u64>, Error> {
    parameters.group(key).ok_or_else(|| {
        Error(format!(
            "parameter group {key:?}: expected an existing internal name"
        ))
    })
}

fn information(parameters: &Parameters, key: &str) -> Result<ParameterGroupInfo, Error> {
    collection::group_info(parameters, require_group(parameters, key)?)
}
