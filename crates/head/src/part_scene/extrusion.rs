// SPDX-License-Identifier: GPL-2.0-only
//! Owner-thread bridge from typed commands to certified profile operations.
use oblikovati_automation::ExtrudeRegionAccess;
use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, PrismResolutionPolicy, certified_sketch_regions_with_policy,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{FeatureId, sketch_prism::SketchPrismSelection},
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator},
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
        let convert = |uses: &[oblikovati_martensite::SketchRegionCurveUse]| {
            uses.iter()
                .map(
                    |item| oblikovati_model::feature::sketch_prism::SketchDirectedCurve {
                        curve: item.curve,
                        reversed: item.reversed,
                    },
                )
                .collect()
        };
        Ok(SketchPrismSelection {
            outer: convert(&region.outer),
            holes: region.holes.iter().map(|hole| convert(hole)).collect(),
        })
    }
    fn validate(
        &self,
        candidate: &DocumentSnapshot,
        document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<usize, String> {
        let history = candidate_history(candidate)?;
        let mut backend = CertifiedNamedSolidEvaluator::with_policy(&history, document, self.0);
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
        document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<String, String> {
        use oblikovati_model::feature::FeatureEvaluationStatus;
        let history = candidate_history(candidate)?;
        let mut backend = CertifiedNamedSolidEvaluator::with_policy(&history, document, self.0);
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

fn candidate_history(candidate: &DocumentSnapshot) -> Result<DocumentEditHistory, String> {
    // Replay does not allocate IDs; include both live uses of the recipe stream (#4074).
    let highest = candidate
        .parameters
        .ids()
        .iter()
        .copied()
        .chain(
            candidate
                .model_states
                .iter()
                .flat_map(|states| states.list())
                .filter(|state| state.id != 1)
                .map(|state| state.id),
        )
        .max()
        .unwrap_or(0);
    let ids = RecipeIdAllocator::restore(Some(highest), candidate.parameters.ids())
        .map_err(|cause| cause.to_string())?;
    DocumentEditHistory::new(candidate.clone(), ids).map_err(|cause| cause.to_string())
}
