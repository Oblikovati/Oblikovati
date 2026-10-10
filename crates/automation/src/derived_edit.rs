// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, json::EmptyObject, wire::DerivedParameterTableDeleteArgs};
use oblikovati_model::parameter::ParameterCollection;

pub(super) fn delete(
    parameters: &mut ParameterCollection<u64>,
    request: DerivedParameterTableDeleteArgs,
) -> Result<Vec<u8>, Error> {
    parameters
        .delete_derived_table(request.id)
        .map_err(|error| Error(error.to_string()))?;
    crate::encode(&EmptyObject {})
}
