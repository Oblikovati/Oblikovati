// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, ParameterRecipe, parameter_error};
use oblikovati_api::types::*;
use oblikovati_model::parameter::ParameterCollection;
use serde::Deserialize;

#[derive(Default, Deserialize)]
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
    fn restore(&self) -> Result<Tolerance, PackageCodecError> {
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
) -> Result<(), PackageCodecError> {
    // Forward drivers must be resolved before numeric metadata can be validated (#3602).
    let ids = parameters.ids().to_vec();
    for (id, record) in ids.into_iter().zip(records) {
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

const TOLERANCE_NAMES: [(&str, ToleranceType); 14] = [
    ("default", TOLERANCE_DEFAULT),
    ("override", TOLERANCE_OVERRIDE),
    ("symmetric", TOLERANCE_SYMMETRIC),
    ("deviation", TOLERANCE_DEVIATION),
    ("limitsStacked", TOLERANCE_LIMITS_STACKED),
    ("limitLinear", TOLERANCE_LIMIT_LINEAR),
    ("max", TOLERANCE_MAX),
    ("min", TOLERANCE_MIN),
    ("limitsFitsStacked", TOLERANCE_LIMITS_FITS_STACKED),
    ("limitsFitsLinear", TOLERANCE_LIMITS_FITS_LINEAR),
    ("limitsFitsShowSize", TOLERANCE_LIMITS_FITS_SHOW_SIZE),
    (
        "limitsFitsShowTolerance",
        TOLERANCE_LIMITS_FITS_SHOW_TOLERANCE,
    ),
    ("basic", TOLERANCE_BASIC),
    ("reference", TOLERANCE_REFERENCE),
];

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

fn model_selection(name: &str) -> Result<ModelValueType, PackageCodecError> {
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
