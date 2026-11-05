// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::{
    parameter::{ParameterState, ParameterValue},
    quantity::Quantity,
    units::parse_category,
};
use serde::{Deserialize, Serialize};

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(super) struct NumericDiagnostic {
    value: f64,
    unit: String,
}

impl NumericDiagnostic {
    pub(super) fn capture(state: &ParameterState<u64>) -> Result<Option<Self>, PackageCodecError> {
        if matches!(state.health, ParameterHealth::Healthy) {
            return Ok(None);
        }
        match &state.last_good_value {
            None => Ok(None),
            Some(ParameterValue::Numeric(quantity)) => Ok(Some(Self {
                value: quantity.value,
                unit: oblikovati_model::units::category_name(quantity.dimension).into(),
            })),
            Some(value) => Err(PackageCodecError(format!(
                "parameter {:?} diagnostic {value:?}: expected numeric last-good value",
                state.name
            ))),
        }
    }

    fn quantity(&self) -> Result<Quantity, PackageCodecError> {
        let dimension = parse_category(&self.unit)
            .filter(|dimension| dimension.is_arithmetic())
            .ok_or_else(|| {
                PackageCodecError(format!(
                    "diagnostic unit {:?}: expected a numeric quantity category",
                    self.unit
                ))
            })?;
        if !self.value.is_finite() {
            return Err(PackageCodecError(format!(
                "diagnostic value {}: expected a finite value",
                self.value
            )));
        }
        Ok(Quantity::new(self.value, dimension))
    }
}

pub(super) fn restore_diagnostic(
    id: u64,
    record: &ParameterRecipe,
    parameters: &mut ParameterCollection<u64>,
    schema: u32,
) -> Result<bool, PackageCodecError> {
    let failed = !matches!(
        parameters.get(id).expect("restored ID").health,
        ParameterHealth::Healthy
    );
    if record.last_good.is_some() && (schema < 3 || !failed) {
        return Err(PackageCodecError(format!(
            "parameter {:?} lastGood: expected a failed numeric parameter in schema 3",
            record.name
        )));
    }
    if !failed {
        return Ok(false);
    }
    let diagnostic = record
        .last_good
        .as_ref()
        .map(NumericDiagnostic::quantity)
        .transpose()?;
    let tolerance = record
        .tolerance
        .as_ref()
        .map(tolerance::ToleranceRecipe::restore)
        .transpose()?
        .unwrap_or_default();
    let selection = if record.model_value_type.is_empty() {
        oblikovati_api::types::MODEL_VALUE_NOMINAL
    } else {
        tolerance::model_selection(&record.model_value_type)?
    };
    parameters
        .restore_numeric_diagnostics(id, diagnostic, tolerance, selection)
        .map_err(parameter_error)?;
    Ok(true)
}
