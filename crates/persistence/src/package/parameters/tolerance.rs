// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, ParameterRecipe, parameter_error};
use oblikovati_api::types::*;
use oblikovati_model::parameter::ParameterCollection;
use oblikovati_model::parameter::vocabulary::TOLERANCE_NAMES;
use serde::{Deserialize, Serialize};

#[derive(Default, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
pub(super) struct ToleranceRecipe {
    #[serde(rename = "type")]
    kind: String,
    upper: f64,
    lower: f64,
    hole_tolerance: String,
    shaft_tolerance: String,
}

impl ToleranceRecipe {
    pub(super) fn restore(&self) -> Result<Tolerance, PackageCodecError> {
        Ok(Tolerance {
            type_: tolerance_kind(&self.kind)?,
            upper: self.upper,
            lower: self.lower,
            hole_tolerance: self.hole_tolerance.clone(),
            shaft_tolerance: self.shaft_tolerance.clone(),
        })
    }
}

pub(super) fn restore_tolerances(
    records: &[ParameterRecipe],
    parameters: &mut ParameterCollection<u64>,
    schema: u32,
) -> Result<(), PackageCodecError> {
    // Forward drivers must be resolved before numeric metadata can be validated (#3602).
    let ids = parameters.ids().to_vec();
    for (id, record) in ids.into_iter().zip(records) {
        if super::diagnostics::restore_diagnostic(id, record, parameters, schema)? {
            continue;
        }
        if let Some(tolerance) = &record.tolerance {
            parameters
                .set_tolerance(id, tolerance.restore()?)
                .map_err(parameter_error)?;
        }
        if !record.model_value_type.is_empty() {
            parameters
                .set_model_value_type(id, model_selection(&record.model_value_type)?)
                .map_err(parameter_error)?;
            parameters.model_value(id).map_err(parameter_error)?;
        }
    }
    Ok(())
}

fn tolerance_kind(name: &str) -> Result<ToleranceType, PackageCodecError> {
    TOLERANCE_NAMES
        .into_iter()
        .find(|(label, _)| *label == name)
        .map(|(_, kind)| kind)
        .ok_or_else(|| {
            PackageCodecError(format!(
                "tolerance type {name:?}: expected a known API tolerance name"
            ))
        })
}

pub(super) fn model_selection(name: &str) -> Result<ModelValueType, PackageCodecError> {
    match name {
        "nominal" => Ok(MODEL_VALUE_NOMINAL),
        "lower" => Ok(MODEL_VALUE_LOWER),
        "upper" => Ok(MODEL_VALUE_UPPER),
        "median" => Ok(MODEL_VALUE_MEDIAN),
        _ => Err(PackageCodecError(format!(
            "model value type {name:?}: expected nominal, lower, upper, or median"
        ))),
    }
}

impl ToleranceRecipe {
    pub(super) fn capture(tolerance: &Tolerance) -> Result<Self, PackageCodecError> {
        let kind = TOLERANCE_NAMES
            .iter()
            .find(|(_, kind)| *kind == tolerance.type_)
            .ok_or_else(|| {
                PackageCodecError(format!(
                    "tolerance {:?}: expected a known API type",
                    tolerance.type_
                ))
            })?
            .0;
        Ok(Self {
            kind: kind.into(),
            upper: tolerance.upper,
            lower: tolerance.lower,
            hole_tolerance: tolerance.hole_tolerance.clone(),
            shaft_tolerance: tolerance.shaft_tolerance.clone(),
        })
    }
}
