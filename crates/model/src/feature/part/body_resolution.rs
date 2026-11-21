// SPDX-License-Identifier: GPL-2.0-only
//! Resolve an exact body or face name without accepting ambiguous matches.
use super::{PartFeatureEvaluator, body_lineage::BodyLineageIndex};
use crate::feature::{
    FeatureId, work_geometry::WorkGeometryError, work_plane::WorkPlaneFaceReference,
};
use std::collections::BTreeMap;

pub(super) fn resolve_body_slot<B: PartFeatureEvaluator>(
    bodies: &BTreeMap<FeatureId, B::Body>,
    lineage: &BodyLineageIndex,
    backend: &B,
    source: FeatureId,
    key: &[u8],
) -> Result<FeatureId, String> {
    resolve_reference(bodies, lineage, source, key, |body| {
        backend.verify_body_reference(body, key)
    })
    .map(|(id, ())| id)
}

pub(super) fn resolve_body_face<B: PartFeatureEvaluator>(
    bodies: &BTreeMap<FeatureId, B::Body>,
    lineage: &BodyLineageIndex,
    backend: &B,
    reference: &WorkPlaneFaceReference,
) -> Result<crate::sketch::SketchPlane, WorkGeometryError> {
    resolve_reference(
        bodies,
        lineage,
        reference.source(),
        reference.key(),
        |body| backend.resolve_planar_face(body, reference.key()),
    )
    .map(|(_, plane)| plane)
    .map_err(WorkGeometryError::BodyReference)
}

pub(super) fn resolve_body_frame_origin<B: PartFeatureEvaluator>(
    bodies: &BTreeMap<FeatureId, B::Body>,
    lineage: &BodyLineageIndex,
    backend: &B,
    reference: &crate::document_sketch::SketchFrameTopologyOrigin,
) -> Result<[f64; 3], WorkGeometryError> {
    resolve_reference(
        bodies,
        lineage,
        reference.source(),
        reference.key(),
        |body| backend.resolve_frame_origin(body, reference.key()),
    )
    .map(|(_, point)| point)
    .map_err(WorkGeometryError::BodyReference)
}

fn resolve_reference<Body, Value>(
    bodies: &BTreeMap<FeatureId, Body>,
    lineage: &BodyLineageIndex,
    source: FeatureId,
    key: &[u8],
    resolve: impl Fn(&Body) -> Result<Value, String>,
) -> Result<(FeatureId, Value), String> {
    let candidates: Vec<_> = lineage.candidates(source).collect();
    let [id] = candidates.as_slice() else {
        return Err(format!(
            "reference {key:?}, source {source:?}, candidates {candidates:?}: expected one available unconsumed body"
        ));
    };
    let body = bodies.get(id).ok_or_else(|| {
        format!("reference {key:?}, source {source:?}, missing body {id:?}: expected one available unconsumed body")
    })?;
    resolve(body)
        .map(|value| (*id, value))
        .map_err(|cause| format!("reference {key:?}, source {source:?}, body {id:?}: {cause}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn multiple_candidate_owners_are_rejected_without_selecting_a_match() {
        let source = FeatureId(1);
        let mut lineage = BodyLineageIndex::default();
        lineage.insert_test_owners(source, &[FeatureId(2), FeatureId(3)]);
        let bodies = BTreeMap::from([(FeatureId(2), true), (FeatureId(3), false)]);
        let result = resolve_reference(&bodies, &lineage, source, b"key", |body| {
            if *body {
                Ok(())
            } else {
                Err("invalid face".into())
            }
        });
        assert!(
            result
                .unwrap_err()
                .contains("expected one available unconsumed body")
        );
    }

    #[test]
    fn failed_body_lookup_retains_the_backend_error_and_reference() {
        let source = FeatureId(1);
        let mut lineage = BodyLineageIndex::default();
        lineage.register(source).unwrap();
        let bodies = BTreeMap::from([(source, ())]);
        let result: Result<(FeatureId, ()), _> =
            resolve_reference(&bodies, &lineage, source, b"key", |_| {
                Err("face is not planar".into())
            });
        let error = result.unwrap_err();
        assert!(error.contains("face is not planar"));
        assert!(error.contains("reference [107, 101, 121]"));
    }
}
