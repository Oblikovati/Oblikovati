// SPDX-License-Identifier: GPL-2.0-only
//! Diagnostic prism queries isolate failures on independent bodies.
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature, fillet::FilletDefinition, part::PartFeatureDefinition,
};

#[test]
fn diagnostic_refuses_only_the_body_with_uncertain_blend() {
    let mut snapshot = cube_document();
    let first = snapshot.work_geometry.get(FeatureId(1)).unwrap();
    let second = AuthoredFeature::new(
        FeatureId(2),
        "Second",
        first.definition().clone(),
        first.reads().clone(),
    );
    snapshot.work_geometry.append(second).unwrap();
    let mut history = owner(snapshot);
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::new(
        &history,
        document(),
        certified_resolution(),
    );
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let body = replay.body(FeatureId(2)).unwrap();
    let body_key = body
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let edge = body
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
        FilletDefinition::new(body_key, vec![edge], DatumLength::Constant(0.1)).unwrap();
    let reads = definition.reads();
    let token = history.begin("Fillet second body").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            assert_eq!(id, FeatureId(3));
            snapshot
                .work_geometry
                .append(AuthoredFeature::new(
                    id,
                    "Fillet",
                    PartFeatureDefinition::Fillet(Box::new(definition)),
                    reads,
                ))
                .map_err(|cause| cause.to_string())?;
            Ok::<(), String>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::new(
        &history,
        document(),
        certified_resolution(),
    );
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    assert!(replay.body(FeatureId(3)).is_some());
    assert!(
        oblikovati_martensite::retain_named_sketch_prism(&history, document(), FeatureId(1))
            .is_ok()
    );
    assert!(
        oblikovati_martensite::retain_named_sketch_prism(&history, document(), FeatureId(2))
            .is_err()
    );
}
