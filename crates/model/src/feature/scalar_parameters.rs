// SPDX-License-Identifier: GPL-2.0-only
use super::{FeatureScalarOwnershipError, FeatureSequence};
use crate::parameter::ParameterCollection;

impl<Definition> FeatureSequence<Definition> {
    /// Validate ownership against document parameters, including failed formulas.
    /// Example: `program.validate_scalar_parameters(&parameters)?` before owner adoption.
    pub fn validate_scalar_parameters(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<(), FeatureScalarOwnershipError> {
        for feature in &self.features {
            for binding in feature.owned_scalars() {
                require_model_parameter(parameters, feature, binding)?;
            }
        }
        Ok(())
    }
}

fn require_model_parameter<Definition>(
    parameters: &ParameterCollection<u64>,
    feature: &super::AuthoredFeature<Definition>,
    binding: &super::FeatureScalarBinding,
) -> Result<(), FeatureScalarOwnershipError> {
    if parameters.validate_owned_scalar(binding.parameter).is_ok() {
        return Ok(());
    }
    Err(feature.scalar_error(
        binding,
        "an existing Model parameter with a numeric formula in this document",
    ))
}
