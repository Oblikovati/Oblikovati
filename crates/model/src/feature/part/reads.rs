// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    feature::{FeatureId, FeatureReadFootprint, FeatureReplayError},
    history::DocumentSnapshot,
};
impl crate::feature::FeatureSequence<PartFeatureDefinition> {
    /// Enumerate consuming features in authored program order without resolving names.
    /// Example: `snapshot.work_geometry.sketch_consumers(sketch).next()` gives its primary owner.
    pub fn sketch_consumers(
        &self,
        sketch: crate::document_sketch::DocumentSketchId,
    ) -> impl Iterator<Item = &crate::feature::AuthoredFeature<PartFeatureDefinition>> {
        self.features().iter().filter(move |feature| {
            matches!(feature.definition(),
            PartFeatureDefinition::SketchPrism(definition) if definition.sketch() == sketch)
        })
    }
}
impl DocumentSnapshot {
    /// Validate current body source dependencies before owner adoption; example: `snapshot.validate_prism_reads()?`.
    pub fn validate_prism_reads(&self) -> Result<(), PartFeatureProgramError> {
        for (id, reads) in self.current_prism_reads()? {
            let feature = self
                .work_geometry
                .get(id)
                .map_err(PartFeatureProgramError::Sequence)?;
            require_current_read_keys(feature, &reads)?;
        }
        Ok(())
    }
    /// Atomically refresh typed source dependencies after sketch edits; example: `candidate.refresh_prism_reads()?`.
    pub fn refresh_prism_reads(&mut self) -> Result<(), PartFeatureProgramError> {
        let reads = self.current_prism_reads()?;
        let mut program = self.work_geometry.clone();
        for (id, reads) in reads {
            let feature = program.get(id).map_err(PartFeatureProgramError::Sequence)?;
            if feature.definition_reads == reads {
                continue;
            }
            let definition = feature.definition().clone();
            program
                .replace(id, definition, reads)
                .map_err(PartFeatureProgramError::Sequence)?;
        }
        self.work_geometry = program;
        Ok(())
    }
    fn current_prism_reads(
        &self,
    ) -> Result<Vec<(FeatureId, FeatureReadFootprint)>, PartFeatureProgramError> {
        let mut reads = Vec::new();
        for feature in self.work_geometry.features() {
            let PartFeatureDefinition::SketchPrism(definition) = feature.definition() else {
                continue;
            };
            let current =
                definition
                    .reads(self)
                    .map_err(|cause| PartFeatureProgramError::Prism {
                        feature: feature.id(),
                        cause,
                    })?;
            reads.push((feature.id(), current));
        }
        Ok(reads)
    }
}

fn require_current_read_keys(
    feature: &crate::feature::AuthoredFeature<PartFeatureDefinition>,
    reads: &FeatureReadFootprint,
) -> Result<(), PartFeatureProgramError> {
    let missing = reads
        .keys()
        .difference(feature.reads().keys())
        .copied()
        .collect::<Vec<_>>();
    if !missing.is_empty() {
        return Err(PartFeatureProgramError::Reads(
            FeatureReplayError::Footprint {
                feature: feature.id(),
                missing,
            },
        ));
    }
    Ok(())
}
