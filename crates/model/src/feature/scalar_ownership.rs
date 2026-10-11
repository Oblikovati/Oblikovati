// SPDX-License-Identifier: GPL-2.0-only
use super::{AuthoredFeature, FeatureId, FeatureReadKey, FeatureSequence, FeatureSequenceError};
use std::{collections::BTreeSet, fmt};

/// Stable semantic role and parameter identity; expressions remain in the document DAG.
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FeatureScalarBinding {
    pub role: String,
    pub parameter: u64,
}

/// Refuse ambiguous ownership before replacing authored state.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FeatureScalarOwnershipError {
    pub feature: FeatureId,
    pub binding: FeatureScalarBinding,
    pub expected: &'static str,
}
impl fmt::Display for FeatureScalarOwnershipError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            output,
            "feature {:?} scalar {:?}: expected {}",
            self.feature, self.binding, self.expected
        )
    }
}
impl std::error::Error for FeatureScalarOwnershipError {}

impl<Definition> AuthoredFeature<Definition> {
    /// Attribute an existing scalar read; example: `feature.with_owned_scalar("offset", id)?`.
    pub fn with_owned_scalar(
        mut self,
        role: &str,
        parameter: u64,
    ) -> Result<Self, FeatureScalarOwnershipError> {
        self.owned_scalars.push(FeatureScalarBinding {
            role: role.into(),
            parameter,
        });
        self.validate_scalar_bindings()?;
        Ok(self)
    }
    /// Inspect durable scalar roles; example: `feature.owned_scalars()[0].parameter`.
    pub fn owned_scalars(&self) -> &[FeatureScalarBinding] {
        &self.owned_scalars
    }
    fn validate_scalar_bindings(&self) -> Result<(), FeatureScalarOwnershipError> {
        let mut roles = BTreeSet::new();
        let mut parameters = BTreeSet::new();
        for binding in &self.owned_scalars {
            self.validate_scalar_binding(binding)?;
            if !roles.insert(&binding.role) || !parameters.insert(binding.parameter) {
                return Err(self.scalar_error(binding, "one unique role and parameter per feature"));
            }
        }
        Ok(())
    }
    fn validate_scalar_binding(
        &self,
        binding: &FeatureScalarBinding,
    ) -> Result<(), FeatureScalarOwnershipError> {
        if !valid_scalar_role(&binding.role) || binding.parameter == 0 {
            return Err(
                self.scalar_error(binding, "a lowercase ASCII role and positive parameter ID")
            );
        }
        self.validate_scalar_read(binding)
    }
    fn validate_scalar_read(
        &self,
        binding: &FeatureScalarBinding,
    ) -> Result<(), FeatureScalarOwnershipError> {
        let key = FeatureReadKey::Parameter(binding.parameter);
        if !self.definition_reads.keys().contains(&key) {
            return Err(self.scalar_error(
                binding,
                "an attributed scalar read in the feature definition",
            ));
        }
        Ok(())
    }
    fn scalar_error(
        &self,
        binding: &FeatureScalarBinding,
        expected: &'static str,
    ) -> FeatureScalarOwnershipError {
        FeatureScalarOwnershipError {
            feature: self.id,
            binding: binding.clone(),
            expected,
        }
    }
}
impl<Definition> FeatureSequence<Definition> {
    pub(super) fn validate_scalar_claims(
        &self,
        feature: &AuthoredFeature<Definition>,
    ) -> Result<(), FeatureSequenceError> {
        feature
            .validate_scalar_bindings()
            .map_err(FeatureSequenceError::ScalarOwnership)?;
        for binding in &feature.owned_scalars {
            let claimed = self.scalar_is_claimed(feature.id, binding.parameter);
            if claimed {
                return Err(FeatureSequenceError::ScalarOwnership(
                    feature.scalar_error(binding, "one owning feature per parameter"),
                ));
            }
        }
        Ok(())
    }
    fn scalar_is_claimed(&self, claimant: FeatureId, parameter: u64) -> bool {
        self.features
            .iter()
            .filter(|entry| entry.id != claimant)
            .any(|entry| {
                entry
                    .owned_scalars
                    .iter()
                    .any(|binding| binding.parameter == parameter)
            })
    }
}
fn valid_scalar_role(role: &str) -> bool {
    role.as_bytes().first().is_some_and(u8::is_ascii_lowercase)
        && role
            .bytes()
            .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-')
}
