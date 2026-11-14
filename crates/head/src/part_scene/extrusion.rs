// SPDX-License-Identifier: GPL-2.0-only
//! Owner-thread bridge from typed commands to certified profile operations.
use oblikovati_automation::ExtrudeRegionAccess;
use oblikovati_martensite::{
    CertifiedPrismEvaluator, PrismResolutionPolicy, certified_sketch_regions_with_policy,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{FeatureId, sketch_prism::SketchPrismSelection},
    history::DocumentSnapshot,
};

pub(crate) struct CertifiedExtrudeAccess(pub PrismResolutionPolicy);
impl ExtrudeRegionAccess for CertifiedExtrudeAccess {
    fn region(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String> {
        let regions = certified_sketch_regions_with_policy(snapshot, sketch, self.0)
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
        let mut backend = CertifiedPrismEvaluator::with_policy(self.0);
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
    fn feature_health(
        &self,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<String, String> {
        use oblikovati_model::feature::FeatureEvaluationStatus;
        let mut backend = CertifiedPrismEvaluator::with_policy(self.0);
        let replay = candidate
            .evaluate_part_features(&mut backend)
            .map_err(|cause| cause.to_string())?;
        let record = replay
            .health()
            .iter()
            .find(|record| record.feature == feature)
            .ok_or_else(|| format!("feature {feature:?}: expected a replay health record"))?;
        Ok(match &record.status {
            FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed => String::new(),
            status => format!("{status:?}"),
        })
    }
}
