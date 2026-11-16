// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[path = "fillet_candidate.rs"]
mod candidate;

#[test]
fn ordered_part_replay_builds_a_named_fillet() {
    use oblikovati_model::feature::{fillet::FilletDefinition, part::PartFeatureDefinition};
    let mut history = owner(cube_document());
    let edge = certified(&history)
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
    let reads = definition.reads();
    let token = history.begin("Fillet").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            assert_eq!(id, FeatureId(2));
            snapshot
                .work_geometry
                .append(oblikovati_model::feature::AuthoredFeature::new(
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
    assert_fillet_replay(&history, 1.);
    history.undo().unwrap();
    assert!(
        history
            .snapshot()
            .work_geometry
            .features()
            .iter()
            .all(|feature| feature.id() != FeatureId(2))
    );
    history.redo().unwrap();
    assert_fillet_replay(&history, 1.);
    let token = history.begin("Size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_fillet_replay(&history, 2.);
    let reopened = restored_owner(history.snapshot());
    assert_fillet_replay(&reopened, 2.);
}

fn assert_fillet_replay(history: &DocumentEditHistory, size: f64) {
    use oblikovati_martensite::CertifiedNamedSolidEvaluator;
    use oblikovati_model::feature::{FeatureEvaluationStatus, work_plane::WorkPlaneFaceReference};
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(history, document(), certified_resolution());
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    assert!(replay.body(FeatureId(1)).is_none());
    let result = replay.body(FeatureId(2)).unwrap();
    assert!(std::ptr::eq(
        result,
        replay.body_from_source(FeatureId(1)).unwrap()
    ));
    let expected = size.powi(3) - (0.01 - std::f64::consts::PI * 0.0025) * size;
    assert!((result.volume(history, document()).unwrap() - expected).abs() < 1e-8);
    let body_key = result.body_reference_key(history, document()).unwrap();
    assert_eq!(body_key, key("body", &[1]));
    let center = result
        .body_bounds_center(history, document(), body_key)
        .unwrap();
    assert!(center.iter().all(|coordinate| coordinate.is_finite()));
    let face_center = result
        .face_bounds_center(history, document(), &key("end", &[1]))
        .unwrap();
    assert!(face_center.iter().all(|coordinate| coordinate.is_finite()));
    let properties = result
        .physical_properties(history, document(), body_key)
        .unwrap();
    assert!((properties.geometry.volume - expected).abs() < 1e-8);
    let topology = result
        .topology_summary(history, document(), body_key)
        .unwrap();
    assert_eq!(
        (topology.faces, topology.edges, topology.vertices),
        (7, 15, 10)
    );
    assert!(
        result
            .face_area(history, document(), &key("end", &[1]))
            .unwrap()
            > 0.
    );
    let mut areas: Vec<_> = result
        .face_reference_keys(history, document())
        .unwrap()
        .iter()
        .map(|key| result.face_area(history, document(), key).unwrap())
        .collect();
    areas.sort_by(f64::total_cmp);
    let corner = 0.01 - std::f64::consts::PI * 0.0025;
    let mut expected_areas = [
        size * size,
        size * size,
        size * size - size * 0.1,
        size * size - size * 0.1,
        size * size - corner,
        size * size - corner,
        std::f64::consts::PI * 0.05 * size,
    ];
    expected_areas.sort_by(f64::total_cmp);
    for (actual, expected) in areas.iter().zip(expected_areas) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "face area {actual}, expected {expected}"
        );
    }
    let keys = result.face_reference_keys(history, document()).unwrap();
    let mut plane_count = 0;
    let mut curved_count = 0;
    for key in &keys {
        assert_eq!(result.face_loop_count(history, document(), key).unwrap(), 1);
        match result
            .face_topology(history, document(), key)
            .unwrap()
            .surface
        {
            oblikovati_martensite::PrismSurfaceKind::Plane => plane_count += 1,
            oblikovati_martensite::PrismSurfaceKind::Cylinder
            | oblikovati_martensite::PrismSurfaceKind::Spline => curved_count += 1,
            other => panic!("face surface {other:?}: expected plane or cylindrical representation"),
        }
    }
    assert_eq!((plane_count, curved_count), (6, 1));
    let edges = result
        .subentity_reference_keys(history, document())
        .unwrap();
    let edge = edges
        .iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 2
        })
        .unwrap();
    assert!(result.edge_length(history, document(), edge).unwrap() > 0.);
    assert!(
        result
            .subentity_point(history, document(), edge)
            .unwrap()
            .iter()
            .all(|coordinate| coordinate.is_finite())
    );
    let line = [[size / 2., size / 2., size + 2.], [0., 0., -1.]];
    let (picked, distance) = result
        .face_key_on_line(history, document(), line)
        .unwrap()
        .unwrap();
    assert_eq!(picked, key("end", &[1]));
    assert!((distance - 2.).abs() < 1e-8);
    let options = oblikovati_martensite::PrismMeshOptions {
        chord_error: 0.01,
        angular_error: 0.2,
    };
    assert!(
        !result
            .face_outlines(history, document(), Some(&picked), options)
            .unwrap()
            .is_empty()
    );
    assert!(
        replay
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy)
    );
    let face = WorkPlaneFaceReference::new(key("end", &[1])).unwrap();
    assert!(replay.datum_evaluation_for_face(&evaluator, &face).is_ok());
}

#[test]
fn fillet_topology_names_are_stable_across_radius_changes() {
    let history = owner(cube_document());
    let body = certified(&history);
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
    let names = |radius| {
        body.fillet_candidate(&history, document(), std::slice::from_ref(&edge), radius)
            .unwrap()
            .name_topology(&history, document(), FeatureId(2))
            .unwrap()
    };
    let first = names(0.1);
    let second = names(0.2);
    let (picked, distance) = first
        .face_key_on_line(&history, document(), [[0.5, 0.5, 3.], [0., 0., -1.]])
        .unwrap()
        .unwrap();
    assert_eq!(picked, key("end", &[1]));
    assert!((distance - 2.).abs() < 1e-8);
    assert_eq!(first.live_keys(), second.live_keys());
    assert_eq!(first.deleted_keys(), second.deleted_keys());
    assert!(first.deleted_keys().contains(&edge));
    for (kind, count) in [(1, 7), (2, 15), (3, 10), (4, 2)] {
        assert_eq!(
            first
                .live_keys()
                .iter()
                .filter(
                    |key| oblikovati_model::identity::ReferenceEnvelope::decode(key)
                        .unwrap()
                        .kind
                        == kind
                )
                .count(),
            count
        );
    }
}

#[test]
fn fillet_names_rebuild_after_source_prism_save_and_reopen() {
    let original = owner(cube_document());
    let reopened = restored_owner(original.snapshot());
    let build = |history: &DocumentEditHistory| {
        let body = certified(history);
        let edge = body
            .subentity_reference_keys(history, document())
            .unwrap()
            .into_iter()
            .find(|key| {
                oblikovati_model::identity::ReferenceEnvelope::decode(key)
                    .unwrap()
                    .kind
                    == 2
            })
            .unwrap();
        body.fillet_candidate(history, document(), &[edge], 0.1)
            .unwrap()
            .name_topology(history, document(), FeatureId(2))
            .unwrap()
    };
    let before = build(&original);
    let after = build(&reopened);
    assert_eq!(before.live_keys(), after.live_keys());
    assert_eq!(before.deleted_keys(), after.deleted_keys());
    for key in after.live_keys() {
        let kind = oblikovati_model::identity::ReferenceEnvelope::decode(key)
            .unwrap()
            .kind;
        assert!(after.contains(&reopened, document(), key, kind).unwrap());
        assert!(before.contains(&reopened, document(), key, kind).is_err());
    }
}

#[test]
fn fillet_candidate_rejects_stale_names_and_leaves_edited_document_usable() {
    let mut history = owner(cube_document());
    let body = certified(&history);
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
    let token = history.begin("Size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        body.fillet_candidate(&history, document(), std::slice::from_ref(&edge), 0.1)
            .is_err()
    );
    let rebuilt = certified(&history);
    assert!(
        rebuilt
            .fillet_candidate(&history, document(), &[edge], 0.1)
            .is_ok()
    );
}
