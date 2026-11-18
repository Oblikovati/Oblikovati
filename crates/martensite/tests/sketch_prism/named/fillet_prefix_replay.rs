// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::feature::{
    AuthoredFeature, fillet::FilletDefinition, part::PartFeatureDefinition,
};

#[test]
fn retained_cube_prefix_builds_the_same_named_fillet_as_full_replay() {
    let history = owner(cube_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let retained = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let prism = retained.body(FeatureId(1)).unwrap();
    let edge = prism
        .subentity_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 2
        })
        .unwrap();
    let definition =
        FilletDefinition::new(key("body", &[1]), vec![edge], DatumLength::Constant(0.1)).unwrap();
    let mut candidate = history.snapshot().clone();
    let reads = definition.reads();
    candidate
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        ))
        .unwrap();
    let tail = retained
        .replay_candidate_tail(&candidate, &mut backend)
        .unwrap();
    let full = candidate.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(tail.health(), full.health());
    let actual = tail.body(FeatureId(2)).unwrap();
    let expected = full.body(FeatureId(2)).unwrap();
    assert_eq!(
        actual.reference_keys(&history, document()).unwrap(),
        expected.reference_keys(&history, document()).unwrap()
    );
    assert!(
        (actual.volume(&history, document()).unwrap()
            - expected.volume(&history, document()).unwrap())
        .abs()
            < 1e-10
    );
    let body_key = actual.body_reference_key(&history, document()).unwrap();
    assert_eq!(
        actual
            .topology_summary(&history, document(), body_key)
            .unwrap(),
        expected
            .topology_summary(&history, document(), body_key)
            .unwrap()
    );
    for face in actual.face_reference_keys(&history, document()).unwrap() {
        let left = actual.face_area(&history, document(), &face).unwrap();
        let right = expected.face_area(&history, document(), &face).unwrap();
        assert!(
            (left - right).abs() < 1e-10,
            "face {face:?}: {left} != {right}"
        );
    }
    assert!(retained.body(FeatureId(1)).is_some());
}
