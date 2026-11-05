// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::feature::FeatureScalarOwnershipError;

impl DocumentSnapshot {
    /// Check feature scalar ownership without requiring healthy evaluated values.
    /// Example: `snapshot.validate_scalar_ownership()?` before persistence.
    pub fn validate_scalar_ownership(&self) -> Result<(), FeatureScalarOwnershipError> {
        self.work_geometry
            .validate_scalar_parameters(&self.parameters)
    }
    pub(super) fn validate_scalar_transition(
        &self,
        before: &Self,
    ) -> Result<(), super::HistoryError> {
        self.validate_scalar_ownership()
            .map_err(super::HistoryError::ScalarOwnership)?;
        for feature in before.work_geometry.features() {
            for binding in feature.owned_scalars() {
                self.require_retained_owner(feature.id(), binding)
                    .map_err(super::HistoryError::ScalarOwnership)?;
            }
        }
        before
            .validate_removed_datum_owners(self)
            .map_err(super::HistoryError::DatumDeletion)
    }
    fn require_retained_owner(
        &self,
        owner: crate::feature::FeatureId,
        binding: &crate::feature::FeatureScalarBinding,
    ) -> Result<(), FeatureScalarOwnershipError> {
        if self.parameters.get(binding.parameter).is_none()
            && self.work_geometry.get(owner).is_err()
        {
            return Ok(());
        }
        if self.retains_scalar_owner(owner, binding) {
            return Ok(());
        }
        Err(FeatureScalarOwnershipError {
            feature: owner,
            binding: binding.clone(),
            expected: "the original owning feature and role, or atomic removal of owner and parameter",
        })
    }
    fn retains_scalar_owner(
        &self,
        owner: crate::feature::FeatureId,
        binding: &crate::feature::FeatureScalarBinding,
    ) -> bool {
        self.work_geometry
            .get(owner)
            .is_ok_and(|feature| feature.owned_scalars().contains(binding))
    }
}
