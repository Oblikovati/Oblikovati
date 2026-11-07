// SPDX-License-Identifier: GPL-2.0-only
//! Owner-thread bridge from typed commands to certified profile operations.
use oblikovati_automation::ExtrudeRegionAccess;
use oblikovati_martensite::{CertifiedPrismEvaluator, PrismResolution, certified_sketch_regions};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{FeatureId, sketch_prism::SketchPrismSelection},
    history::DocumentSnapshot,
};

pub(crate) struct CertifiedExtrudeAccess(pub PrismResolution);
impl ExtrudeRegionAccess for CertifiedExtrudeAccess {
    fn region(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String> {
        let regions = certified_sketch_regions(snapshot, sketch, self.0)
            .map_err(|cause| cause.to_string())?;
        let region = regions.get(index).ok_or_else(|| {
            format!(
                "profile index {index}: expected an index in 0..{}",
                regions.len()
            )
        })?;
        let geometry = snapshot
            .sketches
            .get(sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        region
            .line_selection(geometry)
            .map_err(|cause| cause.to_string())
    }
    fn validate(&self, candidate: &DocumentSnapshot, feature: FeatureId) -> Result<usize, String> {
        let mut backend = CertifiedPrismEvaluator::new(self.0);
        let replay = candidate
            .evaluate_part_features(&mut backend)
            .map_err(|cause| cause.to_string())?;
        if replay.body(feature).is_none() {
            return Err(format!(
                "extrusion {feature:?} health {:?}: expected a certified body",
                replay.health()
            ));
        }
        Ok(replay.ordered_bodies().count())
    }
}
