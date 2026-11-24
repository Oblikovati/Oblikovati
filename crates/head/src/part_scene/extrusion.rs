// SPDX-License-Identifier: GPL-2.0-only
//! Owner-thread bridge from typed commands to certified profile operations.
use oblikovati_automation::ExtrudeRegionAccess;
use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, PrismResolutionPolicy, certified_sketch_regions_with_policy,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{FeatureId, sketch_prism::SketchPrismSelection},
    history::{DocumentSnapshot, SnapshotReplay},
    identity::DocumentGuid,
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
        let mut backend =
            CertifiedNamedSolidEvaluator::with_policy(history.history(), document, self.0);
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
    fn replay_policy(&self) -> Option<PrismResolutionPolicy> {
        Some(self.0)
    }
    fn capture_with_prefix(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
        prefix: &oblikovati_automation::ExtrudeReplayPrefix,
    ) -> Result<Option<oblikovati_automation::ExtrudePrefixReplay>, String> {
        if prefix.policy() != Some(self.0) || !prefix.owns_identity(document) {
            return Err(format!(
                "document {document:?} policy {:?}: expected the extrusion replay owner and policy {:?}",
                prefix.policy(),
                self.0
            ));
        }
        prefix
            .prepare(history, candidate, feature)
            .map_err(|error| error.to_string())
    }
    fn checkpoint_health_policy(&self) -> Option<PrismResolutionPolicy> {
        Some(self.0)
    }
    fn feature_health(
        &self,
        candidate: &DocumentSnapshot,
        document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<String, String> {
        use oblikovati_model::feature::FeatureEvaluationStatus;
        let history = candidate_history(candidate)?;
        let mut backend =
            CertifiedNamedSolidEvaluator::with_policy(history.history(), document, self.0);
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

fn candidate_history(candidate: &DocumentSnapshot) -> Result<SnapshotReplay, String> {
    SnapshotReplay::new(candidate).map_err(|cause| cause.to_string())
}
