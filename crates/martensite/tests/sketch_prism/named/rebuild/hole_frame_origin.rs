// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::identity::ReferenceEnvelope;

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
