// SPDX-License-Identifier: GPL-2.0-only
//! Resolve an exact body or face name without accepting ambiguous matches.
use super::{
    PartFeatureEvaluator,
    body_lineage::{BodyLineageIndex, BodySlot},
};
use crate::feature::{
    FeatureId, work_geometry::WorkGeometryError, work_plane::WorkPlaneFaceReference,
};
use std::collections::BTreeMap;

pub(super) fn resolve_body_slot<B: PartFeatureEvaluator>(
    bodies: &BTreeMap<BodySlot, B::Body>,
    lineage: &BodyLineageIndex,
    backend: &B,
    source: FeatureId,
    key: &[u8],
) -> Result<BodySlot, String> {
    resolve_reference(bodies, lineage, source, key, |body| {
        backend.verify_body_reference(body, key)
    })
    .map(|(id, ())| id)
}

pub(super) fn resolve_body_face<B: PartFeatureEvaluator>(
    bodies: &BTreeMap<BodySlot, B::Body>,
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

pub(super) fn resolve_named_entity<Body, Value>(
    bodies: &BTreeMap<BodySlot, Body>,
    lineage: &BodyLineageIndex,
    source: FeatureId,
    key: &[u8],
    resolve: impl Fn(&Body) -> Result<Value, String>,
) -> Result<Value, String> {
    resolve_reference(bodies, lineage, source, key, resolve).map(|(_, value)| value)
}

pub(super) fn resolve_body_frame_origin<B: PartFeatureEvaluator>(
    bodies: &BTreeMap<BodySlot, B::Body>,
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

impl<Body, Failure> super::PartFeatureEvaluation<Body, Failure> {
    /// Inspect an exact named reference in the current outputs of a producer.
    /// Example: `evaluation.inspect_body_reference(feature, key, |body| verify(body, key))?`.
    pub fn inspect_body_reference<Value>(
        &self,
        feature: FeatureId,
        key: &[u8],
        inspect: impl Fn(&Body) -> Result<Value, String>,
    ) -> Result<Value, String> {
        resolve_reference(&self.bodies, &self.body_lineage, feature, key, inspect)
            .map(|(_, value)| value)
    }
}

fn resolve_reference<Body, Value>(
    bodies: &BTreeMap<BodySlot, Body>,
    lineage: &BodyLineageIndex,
    source: FeatureId,
    key: &[u8],
    resolve: impl Fn(&Body) -> Result<Value, String>,
) -> Result<(BodySlot, Value), String> {
    let candidates: Vec<_> = lineage.candidates(source).collect();
    if let [slot] = candidates.as_slice() {
        let body = bodies.get(slot).ok_or_else(|| {
            format!("reference {key:?}, source {source:?}, missing body {slot:?}: expected one available unconsumed body")
        })?;
        return resolve(body).map(|value| (*slot, value)).map_err(|cause| {
            format!("reference {key:?}, source {source:?}, body {slot:?}: {cause}")
        });
    }
    // Only a registered output group can resolve several siblings by their exact names.
    if !lineage.has_grouped_outputs(source) || candidates.is_empty() {
        return Err(format!(
            "reference {key:?}, source {source:?}, candidates {candidates:?}: expected one available unconsumed body"
        ));
    }
    let mut matched = None;
    let mut declined = Vec::new();
    for slot in candidates {
        let body = bodies.get(&slot).ok_or_else(|| {
            format!("reference {key:?}, source {source:?}, missing body {slot:?}: expected a current output")
        })?;
        match resolve(body) {
            Ok(value) if matched.is_none() => matched = Some((slot, value)),
            Ok(_) => {
                return Err(format!(
                    "reference {key:?}, source {source:?}: expected a name belonging to only one current output"
                ));
            }
            Err(cause) => declined.push((slot, cause)),
        }
    }
    matched.ok_or_else(|| format!(
        "reference {key:?}, source {source:?}, declines {declined:?}: expected a name in a current output"
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn multiple_candidate_owners_are_rejected_without_selecting_a_match() {
        let source = FeatureId(1);
        let mut lineage = BodyLineageIndex::default();
        lineage.insert_test_owners(source, &[FeatureId(2), FeatureId(3)]);
        let bodies = BTreeMap::from([(FeatureId(2).into(), true), (FeatureId(3).into(), false)]);
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
        let bodies = BTreeMap::from([(source.into(), ())]);
        let result: Result<(BodySlot, ()), _> =
            resolve_reference(&bodies, &lineage, source, b"key", |_| {
                Err("face is not planar".into())
            });
        let error = result.unwrap_err();
        assert!(error.contains("face is not planar"));
        assert!(error.contains("reference [107, 101, 121]"));
    }

    #[test]
    fn grouped_body_names_follow_one_replacement_and_preserve_its_sibling() {
        use crate::feature::{
            FeatureEvaluationStatus, FeatureHealthRecord, part::PartFeatureEvaluation,
        };
        use std::collections::BTreeSet;

        let source = FeatureId(1);
        let primary = BodySlot::from(source);
        let sibling = BodySlot {
            feature: source,
            output: 7,
        };
        let replacement = FeatureId(3);
        let mut result = PartFeatureEvaluation::<&str, ()>::empty();
        result
            .body_lineage
            .register_outputs(source, BTreeSet::from([sibling, primary]))
            .unwrap();
        result
            .bodies
            .extend([(primary, "left"), (sibling, "right")]);
        result.health.push(FeatureHealthRecord {
            feature: source,
            status: FeatureEvaluationStatus::Healthy,
        });
        assert_eq!(
            result.ordered_bodies().copied().collect::<Vec<_>>(),
            ["left", "right"]
        );
        assert!(result.body_from_source(source).is_err());
        assert!(
            resolve_reference(
                &result.bodies,
                &result.body_lineage,
                source,
                b"shared",
                |_| Ok(())
            )
            .unwrap_err()
            .contains("only one current output")
        );

        result.body_lineage.replace(primary, replacement).unwrap();
        result.bodies.remove(&primary);
        result.bodies.insert(replacement.into(), "left");
        result.health.push(FeatureHealthRecord {
            feature: replacement,
            status: FeatureEvaluationStatus::Healthy,
        });
        let captured = result
            .try_map_bodies(|owner, body| Ok::<_, ()>((owner, *body)))
            .unwrap();
        assert_eq!(
            captured.ordered_bodies().copied().collect::<Vec<_>>(),
            [(source, "right"), (replacement, "left")]
        );
        let resolve = |name: &'static str| {
            resolve_reference(
                &captured.bodies,
                &captured.body_lineage,
                source,
                name.as_bytes(),
                |(_, body)| {
                    (*body == name)
                        .then_some(*body)
                        .ok_or_else(|| "unknown name".into())
                },
            )
            .unwrap()
            .0
        };
        assert_eq!(resolve("left"), replacement.into());
        assert_eq!(resolve("right"), sibling);
        let mut restored = captured
            .try_map_owned_bodies(|_, (_, body)| Ok::<_, ()>(body))
            .unwrap();

        restored.body_lineage.retire(sibling);
        restored.bodies.remove(&sibling);
        assert_eq!(
            restored.ordered_bodies().copied().collect::<Vec<_>>(),
            ["left"]
        );
        assert_eq!(restored.body_from_source(source), Ok(&"left"));
    }
}
