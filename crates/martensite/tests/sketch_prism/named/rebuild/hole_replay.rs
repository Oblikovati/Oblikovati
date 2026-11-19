// SPDX-License-Identifier: GPL-2.0-only
//! Replay a named Hole on the parameter-driven cube (#4093).
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, NamedSolid, PrismSurfaceKind};
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    hole::{HoleDefinition, HoleExtent, HolePosition, HoleRecipe},
};

#[path = "hole_frame_origin.rs"]
mod hole_frame_origin;

fn hole_document() -> DocumentSnapshot {
    let mut snapshot = cube_document();
    snapshot
        .parameters
        .insert_expression(
            4,
            "hole_diameter",
            oblikovati_api::types::USER_PARAM,
            "2 mm",
        )
        .unwrap();
    snapshot
}
fn add_hole(history: &mut DocumentEditHistory, extent: HoleExtent) -> FeatureId {
    let definition = HoleDefinition::from_recipe(HoleRecipe {
        version: 1,
        body: key("body", &[1]),
        face: key("end", &[1]),
        position: HolePosition::FaceCentroid,
        diameter: DatumLength::Parameter(4),
        extent,
    })
    .unwrap();
    let token = history.begin("Hole").unwrap();
    let id = history.create_hole(&token, "Hole", definition).unwrap();
    history.commit(&token, false).unwrap();
    id
}

fn check_cut(
    history: &DocumentEditHistory,
    size: f64,
    depth: f64,
    diameter: f64,
    feature: FeatureId,
) -> Vec<Vec<u8>> {
    let mut backend =
        CertifiedNamedSolidEvaluator::new(history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        result
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        result.health()
    );
    let body = result
        .body(feature)
        .expect("expected the completed Hole body");
    let achieved = body
        .achieved_tolerance()
        .expect("Hole must record measured tolerances");
    for value in [achieved.vertex, achieved.edge, achieved.face] {
        assert!(
            value.is_finite() && value >= 0. && value <= certified_resolution().maximum_tolerance
        );
    }
    let expected = size.powi(3) - std::f64::consts::PI * (diameter / 2.).powi(2) * depth;
    assert!((body.volume(history, document()).unwrap() - expected).abs() < 1e-7);
    let body_key = key("body", &[1]);
    assert_eq!(
        body.body_reference_key(history, document()).unwrap(),
        body_key
    );
    assert!(
        body.face_area(history, document(), &key("end", &[1]))
            .unwrap()
            > 0.
    );
    let topology = body
        .topology_summary(history, document(), &body_key)
        .unwrap();
    assert!(topology.solid, "Hole result must be a solid: {topology:?}");
    assert_eq!(topology.faces, if depth < size { 8 } else { 7 });
    check_face_areas(body, history, size, depth, diameter);
    if size == 1. && diameter == 0.2 {
        compare_occt_faces(
            body,
            history,
            if depth < size { "blind" } else { "through" },
        );
    }
    body.reference_keys(history, document()).unwrap()
}

fn compare_occt_faces(body: &NamedSolid, history: &DocumentEditHistory, case: &str) {
    let mut actual = body
        .face_reference_keys(history, document())
        .unwrap()
        .into_iter()
        .map(|key| {
            let surface = match body
                .face_topology(history, document(), &key)
                .unwrap()
                .surface
            {
                PrismSurfaceKind::Plane => 0,
                PrismSurfaceKind::Cylinder => 1,
                other => panic!("face surface {other:?}: expected a plane or cylinder"),
            };
            (
                surface,
                body.face_area(history, document(), &key).unwrap(),
                body.face_loop_count(history, document(), &key).unwrap(),
            )
        })
        .collect::<Vec<_>>();
    let mut expected = include_str!("../../../hole_oracle/occt-7.6.3.csv")
        .lines()
        .filter_map(|line| {
            let parts = line.split(',').collect::<Vec<_>>();
            (parts[0] == case).then(|| {
                (
                    parts[1].parse::<u32>().unwrap(),
                    parts[2].parse::<f64>().unwrap(),
                    parts[3].parse::<usize>().unwrap(),
                )
            })
        })
        .collect::<Vec<_>>();
    let order = |left: &(u32, f64, usize), right: &(u32, f64, usize)| {
        left.0
            .cmp(&right.0)
            .then(left.1.total_cmp(&right.1))
            .then(left.2.cmp(&right.2))
    };
    actual.sort_by(order);
    expected.sort_by(order);
    assert_eq!(actual.len(), expected.len(), "OCCT face count");
    for (actual, expected) in actual.iter().zip(expected) {
        assert_eq!(
            (actual.0, actual.2),
            (expected.0, expected.2),
            "OCCT face type and loop count"
        );
        assert!(
            (actual.1 - expected.1).abs() < 1e-7,
            "face area {}: expected OCCT {}",
            actual.1,
            expected.1
        );
    }
}

fn check_face_areas(
    body: &NamedSolid,
    history: &DocumentEditHistory,
    size: f64,
    depth: f64,
    diameter: f64,
) {
    let keys = body.face_reference_keys(history, document()).unwrap();
    let mut areas = Vec::new();
    let mut cylinders = 0;
    for key in keys {
        let topology = body.face_topology(history, document(), &key).unwrap();
        cylinders += usize::from(topology.surface == PrismSurfaceKind::Cylinder);
        areas.push(body.face_area(history, document(), &key).unwrap());
    }
    assert_eq!(cylinders, 1);
    let circle = std::f64::consts::PI * (diameter / 2.).powi(2);
    let mut expected = vec![size * size; if depth < size { 5 } else { 4 }];
    expected.push(size * size - circle);
    expected.push(std::f64::consts::PI * diameter * depth);
    expected.push(if depth < size {
        circle
    } else {
        size * size - circle
    });
    expected.sort_by(f64::total_cmp);
    areas.sort_by(f64::total_cmp);
    for (actual, expected) in areas.iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-7,
            "face area {actual}: expected {expected}"
        );
    }
}

#[test]
fn through_hole_replays_across_resize_undo_redo_and_native_reopen() {
    let mut history = owner(hole_document());
    add_hole(&mut history, HoleExtent::ThroughAll);
    let names = check_cut(&history, 1., 1., 0.2, FeatureId(2));
    history.undo().unwrap();
    assert!(history.snapshot().work_geometry.get(FeatureId(2)).is_err());
    history.redo().unwrap();
    assert_eq!(check_cut(&history, 1., 1., 0.2, FeatureId(2)), names);
    let token = history.begin("Size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(check_cut(&history, 2., 2., 0.2, FeatureId(2)), names);
    let token = history.begin("Diameter").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(4, "3 mm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(check_cut(&history, 2., 2., 0.3, FeatureId(2)), names);
    assert_eq!(
        check_cut(
            &restored_owner(history.snapshot()),
            2.,
            2.,
            0.3,
            FeatureId(2)
        ),
        names
    );
}

#[test]
fn blind_hole_replays_with_a_flat_end() {
    let mut history = owner(hole_document());
    add_hole(
        &mut history,
        HoleExtent::Distance(DatumLength::Constant(0.4)),
    );
    check_cut(&history, 1., 0.4, 0.2, FeatureId(2));
}

#[test]
fn hole_uses_the_current_body_after_a_fillet() {
    use oblikovati_model::{feature::fillet::FilletDefinition, identity::ReferenceEnvelope};
    let mut history = owner(hole_document());
    let edge = certified(&history)
        .subentity_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|key| ReferenceEnvelope::decode(key).unwrap().kind == 2)
        .unwrap();
    let fillet =
        FilletDefinition::new(key("body", &[1]), vec![edge], DatumLength::Constant(0.1)).unwrap();
    let token = history.begin("Fillet").unwrap();
    assert_eq!(
        history.create_fillet(&token, "Fillet", fillet).unwrap(),
        FeatureId(2)
    );
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let before = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let body = before.body(FeatureId(2)).expect("expected fillet body");
    let volume = body.volume(&history, document()).unwrap();
    let old_names = body.reference_keys(&history, document()).unwrap();
    assert_eq!(add_hole(&mut history, HoleExtent::ThroughAll), FeatureId(3));
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        after
            .health()
            .iter()
            .all(|entry| entry.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        after.health()
    );
    let body = after.body(FeatureId(3)).expect("expected fillet with Hole");
    let actual = body.volume(&history, document()).unwrap();
    assert!((actual - (volume - std::f64::consts::PI * 0.01)).abs() < 1e-7);
    let names = body.reference_keys(&history, document()).unwrap();
    assert!(
        old_names.iter().all(|name| names.contains(name)),
        "all prior face and body names must survive"
    );
}

#[test]
fn second_hole_replaces_the_current_body_and_preserves_the_first_names() {
    let mut history = owner(hole_document());
    add_hole(&mut history, HoleExtent::ThroughAll);
    let prior_names = check_cut(&history, 1., 1., 0.2, FeatureId(2));
    let second = HoleDefinition::from_recipe(HoleRecipe {
        version: 1,
        body: key("body", &[1]),
        face: key("end", &[1]),
        position: HolePosition::ModelPoint([0.25, 0.25, 1.].map(DatumLength::Constant)),
        diameter: DatumLength::Parameter(4),
        extent: HoleExtent::ThroughAll,
    })
    .unwrap();
    let token = history.begin("Second hole").unwrap();
    assert_eq!(
        history.create_hole(&token, "Hole2", second).unwrap(),
        FeatureId(3)
    );
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        result
            .health()
            .iter()
            .all(|entry| entry.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        result.health()
    );
    let body = result.body(FeatureId(3)).expect("expected two-hole body");
    let actual = body.volume(&history, document()).unwrap();
    assert!((actual - (1. - std::f64::consts::PI * 0.02)).abs() < 1e-7);
    let names = body.reference_keys(&history, document()).unwrap();
    assert!(prior_names.iter().all(|name| names.contains(name)));
}

#[test]
fn hole_outside_placement_face_does_not_replace_the_body() {
    let mut history = owner(hole_document());
    let bad = HoleDefinition::from_recipe(HoleRecipe {
        version: 1,
        body: key("body", &[1]),
        face: key("end", &[1]),
        position: HolePosition::ModelPoint([2., 2., 1.].map(DatumLength::Constant)),
        diameter: DatumLength::Parameter(4),
        extent: HoleExtent::ThroughAll,
    })
    .unwrap();
    let token = history.begin("Invalid hole").unwrap();
    history.create_hole(&token, "Hole", bad).unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(result.body(FeatureId(1)).is_some());
    assert!(result.body(FeatureId(2)).is_none());
    assert!(matches!(
        result.health()[1].status,
        FeatureEvaluationStatus::Failed(_)
    ));
}

#[test]
fn through_hole_follows_the_face_normal_on_a_negative_extrusion() {
    let mut snapshot = hole_document();
    let extrusion = SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [0., 0., -1.].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = extrusion.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .replace(
            FeatureId(1),
            PartFeatureDefinition::SketchPrism(Box::new(extrusion)),
            reads,
        )
        .unwrap();
    let mut history = owner(snapshot);
    add_hole(&mut history, HoleExtent::ThroughAll);
    check_cut(&history, 1., 1., 0.2, FeatureId(2));
}

#[test]
fn cut_that_splits_the_body_is_refused_without_consuming_the_input() {
    let mut history = owner(hole_document());
    add_hole(&mut history, HoleExtent::ThroughAll);
    let token = history.begin("Oversize hole").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(4, "14 mm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(result.body(FeatureId(1)).is_some());
    assert!(result.body(FeatureId(2)).is_none());
    assert!(matches!(
        result.health()[1].status,
        FeatureEvaluationStatus::Failed(_)
    ));
    let status = format!("{:?}", result.health()[1].status);
    assert!(
        status.contains("solids"),
        "expected the single-solid gate: {status}"
    );
}

#[test]
fn hole_without_a_certified_source_resolution_is_declined() {
    let mut history = owner(hole_document());
    add_hole(&mut history, HoleExtent::ThroughAll);
    let mut backend =
        CertifiedNamedSolidEvaluator::with_optional_policy(&history, document(), None);
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(result.body(FeatureId(1)).is_some());
    assert!(result.body(FeatureId(2)).is_none());
    let status = format!("{:?}", result.health()[1].status);
    assert!(
        status.contains("certified resolution"),
        "expected a resolution decline: {status}"
    );
}

#[test]
fn through_hole_keeps_face_names_across_model_scales() {
    let mut history = owner(hole_document());
    add_hole(&mut history, HoleExtent::ThroughAll);
    let names = check_cut(&history, 1., 1., 0.2, FeatureId(2));
    for (size, diameter, size_formula, diameter_formula) in
        [(0.1, 0.02, "1 mm", "0.2 mm"), (10., 2., "100 mm", "20 mm")]
    {
        let token = history.begin("Scale").unwrap();
        history
            .edit(&token, |snapshot, _| {
                snapshot.parameters.set_expression(1, size_formula)?;
                snapshot.parameters.set_expression(4, diameter_formula)?;
                solve_selected_sketches(snapshot);
                Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        assert_eq!(
            check_cut(&history, size, size, diameter, FeatureId(2)),
            names
        );
    }
}
