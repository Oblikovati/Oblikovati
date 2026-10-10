// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::*;

impl DocumentPackage {
    /// Return a package with edited parameter sections; preserve all other sections.
    /// Example: `let updated = package.with_parameters(&parameters, units)?;`.
    pub fn with_parameters(
        &self,
        parameters: &ParameterCollection<u64>,
        units: UnitsOfMeasure,
    ) -> Result<Self, PackageCodecError> {
        validate_health(parameters)?;
        super::super::units::require_preserved_scale(self, &units)?;
        let mut candidate = self.clone();
        let mut model = self
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        model.insert("units".into(), super::super::units::capture_units(&units)?);
        let records = parameters
            .ids()
            .iter()
            .map(|id| ParameterRecipe::capture(parameters.get(*id).expect("owned parameter")))
            .collect::<Result<Vec<_>, _>>()?;
        model.insert(
            "parameters".into(),
            serde_yaml_ng::to_value(records).map_err(codec_error)?,
        );
        model.insert(
            "parameterGroups".into(),
            groups::capture_groups(parameters)?,
        );
        model.insert(
            "parameterSettings".into(),
            settings::capture_settings(parameters.settings())?,
        );
        model.insert(
            "derivedParameterTables".into(),
            derived::capture_tables(parameters)?,
        );
        candidate.root.insert("model".into(), Value::Mapping(model));
        verify_parameter_capture(
            parameters,
            &candidate.restore_document_parameters()?.parameters,
        )?;
        Ok(candidate)
    }
}

fn verify_parameter_capture(
    original: &ParameterCollection<u64>,
    restored: &ParameterCollection<u64>,
) -> Result<(), PackageCodecError> {
    for (before, after) in original.ids().iter().zip(restored.ids()) {
        let before = original.get(*before).expect("owned parameter");
        if before != restored.get(*after).expect("restored parameter") {
            return Err(PackageCodecError(format!(
                "parameter {:?}: expected recipe-representable state with an exact value round trip",
                before.name
            )));
        }
    }
    Ok(())
}

pub(super) fn kind_name(kind: ParameterKind) -> Result<&'static str, PackageCodecError> {
    match kind {
        USER_PARAM => Ok("user"),
        MODEL_PARAM => Ok("model"),
        TABLE_PARAM => Ok("table"),
        REFERENCE_PARAM => Ok("reference"),
        DERIVED_PARAM => Ok("derived"),
        found => Err(PackageCodecError(format!(
            "parameter kind {found:?}: expected a supported API kind"
        ))),
    }
}

pub(super) fn display_name(
    display: ParameterDisplayFormat,
) -> Result<&'static str, PackageCodecError> {
    match display {
        DISPLAY_FORMAT_DECIMAL => Ok("decimal"),
        DISPLAY_FORMAT_FRACTIONAL => Ok("fractional"),
        DISPLAY_FORMAT_ARCHITECTURAL => Ok("architectural"),
        found => Err(PackageCodecError(format!(
            "parameter display {found:?}: expected a supported API format"
        ))),
    }
}

pub(super) fn selection_name(selection: ModelValueType) -> Result<&'static str, PackageCodecError> {
    match selection {
        MODEL_VALUE_NOMINAL => Ok(""),
        MODEL_VALUE_LOWER => Ok("lower"),
        MODEL_VALUE_UPPER => Ok("upper"),
        MODEL_VALUE_MEDIAN => Ok("median"),
        found => Err(PackageCodecError(format!(
            "model selection {found:?}: expected nominal, lower, upper, or median"
        ))),
    }
}
