// SPDX-License-Identifier: GPL-2.0-only

use crate::parameters;
use oblikovati_api::{
    Error,
    types::{MODEL_PARAM, ParameterKind, REFERENCE_PARAM, USER_PARAM},
    wire::*,
};
use oblikovati_model::parameter::ParameterCollection;

pub(super) fn rename(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterRenameArgs,
) -> Result<ParameterInfo, Error> {
    let id = parameters::resolve(parameters, &request.name)?;
    parameters
        .rename(id, &request.new_name)
        .map_err(|error| Error(error.to_string()))?;
    parameters::information(parameters, id)
}

pub(super) fn convert(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterConvertArgs,
) -> Result<ParameterInfo, Error> {
    let id = parameters::resolve(parameters, &request.name)?;
    parameters
        .convert_kind(id, conversion_kind(&request.target_kind)?)
        .map_err(|error| Error(error.to_string()))?;
    parameters::information(parameters, id)
}

pub(super) fn delete(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterNameArgs,
) -> Result<Vec<u8>, Error> {
    let id = parameters::resolve(parameters, &request.name)?;
    parameters
        .remove(id)
        .map_err(|error| Error(error.to_string()))?;
    Ok(b"{}".to_vec())
}

fn conversion_kind(name: &str) -> Result<ParameterKind, Error> {
    match name.trim().to_lowercase().as_str() {
        "user" => Ok(USER_PARAM),
        "model" => Ok(MODEL_PARAM),
        "reference" => Ok(REFERENCE_PARAM),
        _ => Err(Error(format!(
            "parameter targetKind {name:?}: expected user, model, or reference"
        ))),
    }
}
