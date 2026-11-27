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
        self.features()
            .iter()
            .filter(move |feature| match feature.definition() {
                PartFeatureDefinition::SketchPrism(definition) => definition.sketch() == sketch,
                PartFeatureDefinition::SketchLoft(definition) => {
                    definition.source_sketches().any(|source| source == sketch)
                }
                PartFeatureDefinition::SketchSweep(definition) => {
                    let recipe = definition.recipe();
                    recipe.profile_sketch == sketch || recipe.path_sketch == sketch
                }
                _ => false,
            })
    }
}
impl DocumentSnapshot {
    /// Validate current body source dependencies before owner adoption; example: `snapshot.validate_prism_reads()?`.
    pub fn validate_prism_reads(&self) -> Result<(), PartFeatureProgramError> {
        for (id, reads) in self.current_sketch_body_reads()? {
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
        let reads = self.current_sketch_body_reads()?;
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
    fn current_sketch_body_reads(
        &self,
    ) -> Result<Vec<(FeatureId, FeatureReadFootprint)>, PartFeatureProgramError> {
        let mut reads = Vec::new();
        for feature in self.work_geometry.features() {
            let current =
                match feature.definition() {
                    PartFeatureDefinition::SketchPrism(definition) => definition
                        .reads(self)
                        .map_err(|cause| PartFeatureProgramError::Prism {
                            feature: feature.id(),
                            cause,
                        })?,
                    PartFeatureDefinition::SketchLoft(definition) => definition
                        .reads(self)
                        .map_err(|cause| PartFeatureProgramError::Loft {
                            feature: feature.id(),
                            cause,
                        })?,
                    PartFeatureDefinition::SketchSweep(definition) => definition
                        .reads(self)
                        .map_err(|cause| PartFeatureProgramError::Sweep {
                            feature: feature.id(),
                            cause: Box::new(cause),
                        })?,
                    _ => continue,
                };
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
