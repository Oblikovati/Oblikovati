// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId, part::PartFeatureProgramError, sketch_prism::SketchPrismDefinition,
};
impl DocumentEditHistory {
    /// Add authored prism inputs to the common feature owner and transaction.
    /// Example: `history.create_sketch_prism(&token, "Extrude", definition)?` before commit.
    pub fn create_sketch_prism(
        &mut self,
        token: &TransactionToken,
        name: &str,
        definition: SketchPrismDefinition,
    ) -> Result<FeatureId, EditError<HistoryError>> {
        self.edit(token, |snapshot, ids| {
            let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
            let feature =
                AuthoredFeature::sketch_prism(id, name, definition, snapshot).map_err(|cause| {
                    HistoryError::PartProgram(PartFeatureProgramError::Prism { feature: id, cause })
                })?;
            snapshot.work_geometry.append(feature).map_err(|cause| {
                HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause))
            })?;
            Ok(id)
        })
    }
}
