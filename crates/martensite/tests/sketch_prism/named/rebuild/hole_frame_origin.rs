// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::{
    DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin, SketchFrameTopologyOrigin,
};
use oblikovati_model::identity::ReferenceEnvelope;
use oblikovati_model::sketch::SketchPlane;

#[test]
fn hole_circular_edge_keys_resolve_to_analytic_centers() {
    let mut history = owner(hole_document());
    let original = certified(&history);
    let center = original
        .face_report(&history, document(), &key("end", &[1]))
        .unwrap()
        .centroid;
    add_hole(&mut history, HoleExtent::ThroughAll);
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let body = result.body(FeatureId(2)).unwrap();
    let keys = body.subentity_reference_keys(&history, document()).unwrap();
    let circles: Vec<_> = keys
        .iter()
        .filter(|key| ReferenceEnvelope::decode(key).unwrap().kind == 2)
        .filter_map(|key| body.frame_origin(&history, document(), key).ok())
        .collect();
    assert_eq!(circles.len(), 2, "expected both named circular hole edges");
    for point in circles {
        assert_analytic_scalar(point[0], center[0]);
        assert_analytic_scalar(point[1], center[1]);
        assert!(point[2].abs() < 1e-7 || (point[2] - 1.).abs() < 1e-7);
    }
}

#[test]
fn replay_resolves_a_sketch_origin_from_a_named_hole_edge() {
    let mut history = owner(hole_document());
    add_hole(&mut history, HoleExtent::ThroughAll);
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let body = result.body(FeatureId(2)).unwrap();
    let (key, expected) = first_circular_edge(body, &history);
    let sketch = DocumentSketchId(99);
    let mut candidate = history.snapshot().clone();
    attach_to_edge(&mut candidate, sketch, key);
    let (replayed, sketches) = candidate
        .recompute_part_with_sketches(&mut evaluator, Default::default())
        .unwrap();
    candidate.sketches = sketches;
    let plane = candidate
        .resolve_sketch_plane_with_datums(sketch, &replayed.datum_evaluation())
        .unwrap();
    assert_eq!(plane.recipe().origin, expected);
}

fn first_circular_edge(body: &NamedSolid, history: &DocumentEditHistory) -> (Vec<u8>, [f64; 3]) {
    body.subentity_reference_keys(history, document())
        .unwrap()
        .into_iter()
        .filter(|key| ReferenceEnvelope::decode(key).unwrap().kind == 2)
        .find_map(|key| {
            body.frame_origin(history, document(), &key)
                .ok()
                .map(|point| (key, point))
        })
        .expect("expected a named circular hole edge")
}

fn attach_to_edge(candidate: &mut DocumentSnapshot, sketch: DocumentSketchId, key: Vec<u8>) {
    candidate
        .sketches
        .add(sketch, "Referenced", SketchPlane::xy(), Default::default())
        .unwrap();
    candidate
        .sketches
        .set_frame_edit(
            sketch,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Topology(
                    SketchFrameTopologyOrigin::new(key).unwrap(),
                )),
                alignment: None,
            }),
        )
        .unwrap();
    candidate.refresh_prism_reads().unwrap();
}
