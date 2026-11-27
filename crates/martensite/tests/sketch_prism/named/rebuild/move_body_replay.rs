// SPDX-License-Identifier: GPL-2.0-only
//! Replay named body motion through parameter edits and native YAML (#4097).
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    move_body::{MoveBodyDefinition, MoveBodyOperation},
    work_geometry::DatumAngle,
};

fn moving_document() -> DocumentSnapshot {
    let mut snapshot = cube_document();
    snapshot
        .parameters
        .insert_expression(4, "offset", oblikovati_api::types::USER_PARAM, "10 mm")
        .unwrap();
    snapshot
        .parameters
        .insert_expression(5, "turn", oblikovati_api::types::USER_PARAM, "90 deg")
        .unwrap();
    snapshot
}
fn add_move(history: &mut DocumentEditHistory) -> FeatureId {
    let definition = MoveBodyDefinition::new(
        key("body", &[1]),
        vec![
            MoveBodyOperation::Rotation {
                origin: [DatumLength::Constant(0.); 3],
                direction: [0., 0., 1.],
                angle: DatumAngle::parameter(5),
            },
            MoveBodyOperation::Translation([
                DatumLength::Parameter(4),
                DatumLength::Constant(0.),
                DatumLength::Constant(0.),
            ]),
        ],
    )
    .unwrap();
    let token = history.begin("Move Body").unwrap();
    let id = history
        .create_move_body(&token, "Move Body", definition)
        .unwrap();
    history.commit(&token, false).unwrap();
    id
}
fn assert_position(history: &DocumentEditHistory, feature: FeatureId, x: f64) -> Vec<Vec<u8>> {
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
        "health {:?}",
        result.health()
    );
    let body = result.body(feature).expect("named moved body");
    let face = key("end", &[1]);
    let center = body.face_bounds_center(history, document(), &face).unwrap();
    assert!(
        (center[0] - x).abs() < 1e-8
            && (center[1] - 0.5).abs() < 1e-8
            && (center[2] - 1.).abs() < 1e-8,
        "face center {center:?}"
    );
    assert!((body.volume(history, document()).unwrap() - 1.).abs() < 1e-8);
    body.reference_keys(history, document()).unwrap()
}
#[test]
fn moved_body_recomputes_undoes_redoes_and_reopens_its_named_face() {
    let mut history = owner(moving_document());
    let feature = add_move(&mut history);
    let mut recipe = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 23);
    recipe.version = 22;
    assert!(oblikovati_model::feature::FeatureSequence::from_part_recipe(&recipe).is_err());
    let names = assert_position(&history, feature, 0.5);
    let token = history.begin("Offset").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(4, "20 mm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(assert_position(&history, feature, 1.5), names);
    history.undo().unwrap();
    assert_eq!(assert_position(&history, feature, 0.5), names);
    history.redo().unwrap();
    assert_eq!(assert_position(&history, feature, 1.5), names);
    assert_eq!(
        assert_position(&restored_owner(history.snapshot()), feature, 1.5),
        names
    );
}

#[test]
fn candidate_tail_moves_a_retained_body_without_mutating_the_prefix() {
    use oblikovati_model::feature::{AuthoredFeature, part::PartFeatureDefinition};
    let history = owner(moving_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let prefix = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let definition = MoveBodyDefinition::new(
        key("body", &[1]),
        vec![MoveBodyOperation::Translation([
            DatumLength::Parameter(4),
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
        ])],
    )
    .unwrap();
    let reads = definition.reads();
    let mut candidate = history.snapshot().clone();
    candidate
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Move",
            PartFeatureDefinition::MoveBody(Box::new(definition)),
            reads,
        ))
        .unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let resumed = prefix
        .replay_candidate_tail(&candidate, &mut backend)
        .unwrap();
    let moved = resumed.body(FeatureId(2)).expect("moved candidate");
    let face = key("end", &[1]);
    let moved_center = moved
        .face_bounds_center(&history, document(), &face)
        .unwrap();
    assert!((moved_center[0] - 1.5).abs() < 1e-8);
    let source = prefix.body(FeatureId(1)).expect("retained source");
    assert!(
        (source
            .face_bounds_center(&history, document(), &face)
            .unwrap()[0]
            - 0.5)
            .abs()
            < 1e-8
    );
}

#[test]
fn unresolvable_body_motion_quarantines_failure_without_replacing_source() {
    use oblikovati_model::identity::SemanticReferenceName;
    let mut history = owner(moving_document());
    let foreign = SemanticReferenceName::new(
        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap(),
        1,
        4,
        "sketch-prism",
        "body",
        &[1],
    )
    .unwrap()
    .encode()
    .unwrap();
    let definition = MoveBodyDefinition::new(
        foreign,
        vec![MoveBodyOperation::Translation([
            DatumLength::Constant(1.),
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
        ])],
    )
    .unwrap();
    let token = history.begin("Invalid move").unwrap();
    let feature = history
        .create_move_body(&token, "Invalid move", definition)
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(result.body(feature).is_none());
    assert!(matches!(
        result.health().last().unwrap().status,
        FeatureEvaluationStatus::Failed(_)
    ));
    let source = result.body(FeatureId(1)).expect("source stays available");
    assert!((source.volume(&history, document()).unwrap() - 1.).abs() < 1e-8);
}

#[test]
fn later_motion_resolves_the_original_body_name_after_an_earlier_motion() {
    let mut history = owner(moving_document());
    add_move(&mut history);
    let definition = MoveBodyDefinition::new(
        key("body", &[1]),
        vec![MoveBodyOperation::Translation([
            DatumLength::Parameter(4),
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
        ])],
    )
    .unwrap();
    let token = history.begin("Second move").unwrap();
    let second = history
        .create_move_body(&token, "Second move", definition)
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(second, FeatureId(3));
    assert_position(&history, second, 1.5);
    let reopened = restored_owner(history.snapshot());
    assert_position(&reopened, second, 1.5);
}

fn add_hole(history: &mut DocumentEditHistory) -> FeatureId {
    use oblikovati_model::feature::hole::{HoleDefinition, HoleExtent, HolePosition, HoleRecipe};
    let hole = HoleDefinition::from_recipe(HoleRecipe {
        version: 1,
        body: key("body", &[1]),
        face: key("end", &[1]),
        position: HolePosition::FaceCentroid,
        diameter: DatumLength::Constant(0.2),
        extent: HoleExtent::ThroughAll,
    })
    .unwrap();
    let token = history.begin("Hole").unwrap();
    let feature = history.create_hole(&token, "Hole", hole).unwrap();
    history.commit(&token, false).unwrap();
    feature
}
fn assert_moved_hole(history: &DocumentEditHistory, feature: FeatureId) {
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
        "health {:?}",
        result.health()
    );
    let body = result.body(feature).expect("moved hole body");
    assert!(
        (body.volume(history, document()).unwrap() - (1. - std::f64::consts::PI * 0.01)).abs()
            < 1e-7
    );
    assert!(
        (body
            .face_bounds_center(history, document(), &key("end", &[1]))
            .unwrap()[0]
            - 0.5)
            .abs()
            < 1e-8
    );
}
#[test]
fn hole_after_move_resolves_the_original_face_at_its_new_position() {
    let mut history = owner(moving_document());
    add_move(&mut history);
    let feature = add_hole(&mut history);
    assert_moved_hole(&history, feature);
}
#[test]
fn move_after_hole_preserves_the_cut_and_named_face() {
    let mut history = owner(moving_document());
    add_hole(&mut history);
    let feature = add_move(&mut history);
    assert_moved_hole(&history, feature);
}
