// SPDX-License-Identifier: GPL-2.0-only
//! Verify the saved recipe, certified geometry, and ownership of both Extrude solids.
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid};
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureEvaluationStatus,
        part::{DeleteBodyDefinition, PartFeatureEvaluation},
        sketch_prism::{SketchDirectedCurve, SketchPrismOutput, SketchPrismSelection},
    },
    sketch::{PlanarConstraint, SketchConstraintId, SketchEntityId},
};
use std::{collections::BTreeSet, f64::consts::PI};

fn circle_output() -> SketchPrismOutput {
    SketchPrismOutput {
        slot: 7,
        selection: SketchPrismSelection {
            outer: vec![SketchDirectedCurve {
                curve: SketchEntityId(22),
                reversed: false,
            }],
            holes: Vec::new(),
        },
    }
}

fn grouped_document(outputs: Vec<SketchPrismOutput>) -> DocumentSnapshot {
    let mut snapshot = selected_prism_document();
    let mut geometry = snapshot.sketches.get(SKETCH).unwrap().geometry().clone();
    let center = SketchEntityId(21);
    let circle = SketchEntityId(22);
    let position = [12., 1.];
    geometry.add_point(center, position).unwrap();
    geometry.add_circle(circle, center, 3.).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(21),
            PlanarConstraint::Fix {
                point: center,
                position,
            },
        )
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(22),
            PlanarConstraint::Radius {
                curve: circle,
                parameter: 2,
            },
        )
        .unwrap();
    snapshot
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    solve_selected_sketches(&mut snapshot);
    let definition = SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
    .with_additional_outputs(outputs)
    .unwrap();
    let reads = definition.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .replace(
            FeatureId(1),
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        )
        .unwrap();
    snapshot
}

fn evaluator(history: &DocumentEditHistory) -> CertifiedNamedSolidEvaluator<'_> {
    CertifiedNamedSolidEvaluator::new(history, document(), certified_resolution())
}

fn verify_pair(
    history: &DocumentEditHistory,
    result: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
    depth: f64,
) -> BTreeSet<Vec<u8>> {
    assert!(
        result
            .health()
            .iter()
            .all(|item| item.status == FeatureEvaluationStatus::Healthy)
    );
    let bodies: Vec<_> = result.ordered_bodies().collect();
    let [rect, circle] = bodies.as_slice() else {
        panic!("expected two bodies, got {}", bodies.len());
    };
    let mut all_names = BTreeSet::new();
    for (body, sources, volume) in [
        (*rect, vec![1], 12. * depth),
        (*circle, vec![1, 7], 9. * PI * depth),
    ] {
        let body_key = key("body", &sources);
        assert_eq!(
            body.body_reference_key(history, document()).unwrap(),
            body_key
        );
        assert_analytic_scalar(
            body.physical_properties(history, document(), &body_key)
                .unwrap()
                .geometry
                .volume,
            volume,
        );
        let cap_area = body
            .face_area(history, document(), &key("end", &sources))
            .unwrap();
        assert_analytic_scalar(cap_area, volume / depth);
        for name in body
            .reference_keys(history, document())
            .unwrap()
            .into_iter()
            .chain(body.subentity_reference_keys(history, document()).unwrap())
        {
            assert!(
                all_names.insert(name),
                "sibling topology names must be distinct"
            );
        }
    }
    assert!(result.body_from_source(FeatureId(1)).is_err());
    all_names
}

#[test]
fn grouped_extrude_names_survive_depth_changes_package_replay_and_deletion() {
    let mut history = owner(grouped_document(vec![circle_output()]));
    let replay = |history: &DocumentEditHistory| {
        history
            .snapshot()
            .evaluate_part_features(&mut evaluator(history))
            .unwrap()
    };
    let names = verify_pair(&history, &replay(&history), 2.);
    let token = history.begin("Extrusion depth").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(verify_pair(&history, &replay(&history), 3.), names);
    history.undo().unwrap();
    assert_eq!(verify_pair(&history, &replay(&history), 2.), names);
    history.redo().unwrap();

    let program = &history.snapshot().work_geometry;
    let recipe = program.work_plane_recipe().unwrap();
    assert_eq!(recipe.version, 43);
    let mut old_recipe = recipe;
    old_recipe.version = 42;
    assert!(
        oblikovati_model::feature::FeatureSequence::from_work_plane_recipe(&old_recipe).is_err()
    );

    let reopened = restored_owner(history.snapshot());
    let checkpoints = reopened
        .snapshot()
        .evaluate_part_with_checkpoints(&mut evaluator(&reopened))
        .unwrap();
    assert_eq!(verify_pair(&reopened, checkpoints.result(), 3.), names);

    let mut candidate = reopened.snapshot().clone();
    let second = key("body", &[1, 7]);
    let delete =
        PartFeatureDefinition::DeleteBody(DeleteBodyDefinition::new(second.clone()).unwrap());
    let reads = delete.reads();
    candidate
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Delete circle",
            delete,
            reads,
        ))
        .unwrap();
    let result = checkpoints
        .replay_with_checkpoints(&candidate, &mut evaluator(&reopened), || false)
        .unwrap();
    let result = result.result();
    assert_eq!(result.deleted_body(FeatureId(2)), Some(second.as_slice()));
    assert_eq!(result.ordered_bodies().count(), 1);
    let remaining = result.body_from_source(FeatureId(1)).unwrap();
    let first = key("body", &[1]);
    assert_eq!(
        remaining.body_reference_key(&reopened, document()).unwrap(),
        first
    );
    assert_analytic_scalar(
        remaining
            .physical_properties(&reopened, document(), &first)
            .unwrap()
            .geometry
            .volume,
        36.,
    );
}

#[test]
fn grouped_extrude_rejects_duplicate_or_invalid_outputs_atomically() {
    let mut duplicate = selected_sketch_boundary();
    duplicate.outer.rotate_left(1);
    let mut missing = circle_output().selection;
    missing.outer[0].curve = SketchEntityId(99);
    for selection in [duplicate, missing] {
        let mut outputs = vec![circle_output()];
        outputs.push(SketchPrismOutput {
            slot: 14,
            selection,
        });
        let history = owner(grouped_document(outputs));
        let result = history
            .snapshot()
            .evaluate_part_features(&mut evaluator(&history))
            .unwrap();
        assert_eq!(result.ordered_bodies().count(), 0);
        assert!(matches!(
            result.health()[0].status,
            FeatureEvaluationStatus::Failed(_)
        ));
    }

    let history = owner(grouped_document(vec![circle_output()]));
    let entry = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let PartFeatureDefinition::SketchPrism(definition) = entry.definition() else {
        panic!("expected a saved Extrude");
    };
    let mut recipe = definition.recipe();
    recipe.additional_outputs.push(circle_output());
    assert!(SketchPrismDefinition::from_recipe(&recipe).is_err());
    recipe.additional_outputs.pop();
    recipe.additional_outputs[0].slot = 0;
    assert!(SketchPrismDefinition::from_recipe(&recipe).is_err());
}
