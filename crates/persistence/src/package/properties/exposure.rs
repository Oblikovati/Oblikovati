// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};

impl DocumentPackage {
    /// Rebuild parameter provenance from saved export flags after decoding authored records.
    /// Example: `let properties = package.restore_parameter_properties(&parameters)?;`.
    pub fn restore_parameter_properties(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<DocumentProperties, PackageCodecError> {
        let mut properties = self.restore_properties()?;
        properties
            .sync_exposed_parameters(parameters)
            .map_err(exposure_error)?;
        Ok(properties)
    }

    /// Capture a coherent parameter/property snapshot, including typed exposure provenance.
    /// Example: `let updated = package.with_parameter_properties(&parameters, &properties, units)?;`.
    pub fn with_parameter_properties(
        &self,
        parameters: &ParameterCollection<u64>,
        properties: &DocumentProperties,
        units: UnitsOfMeasure,
    ) -> Result<Self, PackageCodecError> {
        self.restore_properties()?;
        let mut synchronized = properties.clone();
        synchronized
            .sync_exposed_parameters(parameters)
            .map_err(exposure_error)?;
        let mut candidate = self.clone();
        sets::capture_sections(&mut candidate, &synchronized, true)?;
        candidate = candidate.with_parameters(parameters, units)?;
        let restored = candidate.restore_document_parameters()?;
        if candidate.restore_parameter_properties(&restored.parameters)? != synchronized {
            return Err(PackageCodecError(
                "exposed property export: expected an exact reconstructed property collection"
                    .into(),
            ));
        }
        Ok(candidate)
    }
}

fn exposure_error(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!("parameter property exposure: {error}"))
}
