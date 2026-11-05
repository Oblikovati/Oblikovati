// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn rotated_scaled_and_reversed_line_pairs_use_the_same_creation_and_solve_path() {
    for scale in [1e-6, 1.0, 1e6] {
        for turn in [0.0_f64, 0.37, 1.2, -0.81] {
            for reverse in [false, true] {
                assert_generated(scale, turn, reverse);
            }
        }
    }
}
fn assert_generated(scale: f64, turn: f64, reverse: bool) {
    let geometry = transformed_fixture(scale, turn, reverse);
    let mut history = owner(geometry);
    let input = direct(PlanarConstraint::Lines {
        lines: [entity(7), entity(8)],
        relation: LineRelation::Parallel,
    });
    let report = create(&mut history, &input);
    assert_created(&history, &report);
    let snapshot = history.snapshot();
    let health = snapshot.sketches.sketches()[0].health(&snapshot.parameters);
    let PlacedSketchHealth::Current(SketchEvaluation::Numerical(result)) = health else {
        panic!("{health:?}");
    };
    assert_eq!(
        result.termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
}
fn transformed_fixture(scale: f64, turn: f64, reverse: bool) -> Sketch2D {
    let mut recipe = fixture().recipe();
    let (sin, cos) = turn.sin_cos();
    for point in &mut recipe.points {
        let [x, y] = point.position;
        point.position = [scale * (cos * x - sin * y), scale * (sin * x + cos * y)];
    }
    for circle in &mut recipe.circles {
        circle.radius *= scale;
    }
    if reverse {
        recipe.lines[1].endpoints.reverse();
    }
    Sketch2D::from_recipe(&recipe).unwrap()
}
#[test]
fn raw_seed_dependent_relations_are_refused_in_favour_of_current_seed_intents() {
    for relation in raw_seed_inputs() {
        let mut history = owner(fixture());
        let marks = history.sketch_reservations();
        let cause = caught_failure(&mut history, &direct(relation));
        assert!(cause.contains("typed current-seed inputs"));
        assert_eq!(history.sketch_reservations(), marks);
    }
}
#[test]
fn owned_entity_classification_is_immutable_and_missing_entities_are_named() {
    use oblikovati_api::types::*;
    let geometry = fixture();
    let before = geometry.recipe();
    for (id, kind) in [
        (1, SKETCH_ENTITY_POINT),
        (7, SKETCH_ENTITY_LINE),
        (9, SKETCH_ENTITY_CIRCLE),
        (11, SKETCH_ENTITY_ARC),
    ] {
        assert_eq!(geometry.entity_kind(entity(id)).unwrap(), kind);
    }
    let cause = geometry.entity_kind(entity(99)).unwrap_err().to_string();
    assert!(cause.contains("99") && cause.contains("owned"));
    assert_eq!(geometry.recipe(), before);
}

fn raw_seed_inputs() -> [PlanarConstraint; 3] {
    [
        PlanarConstraint::Fix {
            point: entity(2),
            position: [99.0; 2],
        },
        PlanarConstraint::LineTangent {
            line: entity(8),
            curve: entity(9),
            side: oblikovati_model::sketch::LineContactSide::Left,
        },
        PlanarConstraint::CircularTangent {
            curves: [entity(9), entity(10)],
            contact: oblikovati_model::sketch::CircleContact::External,
        },
    ]
}

#[test]
fn zero_length_orientation_declines_locally_and_near_zero_lengths_remain_valid() {
    let mut history = owner(line_pair(0.0));
    let input = direct(PlanarConstraint::Lines {
        lines: [entity(5), entity(6)],
        relation: LineRelation::Parallel,
    });
    let cause = caught_failure(&mut history, &input);
    assert!(cause.contains("nonzero direction"), "{cause}");
    assert_eq!(history.sketch_reservations().constraints[&1], 1);
    for length in [1e-12, 1e-6, 1.0, 1e6, 1e12] {
        let mut history = owner(line_pair(length));
        let report = create(&mut history, &input);
        assert_created(&history, &report);
    }
}
fn line_pair(length: f64) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    for (index, position) in [[0.0, 0.0], [0.0, length], [2.0, 0.0], [2.0, 1.0]]
        .into_iter()
        .enumerate()
    {
        geometry
            .add_point(entity(index as u64 + 1), position)
            .unwrap();
    }
    geometry
        .add_line(entity(5), [entity(1), entity(2)])
        .unwrap();
    geometry
        .add_line(entity(6), [entity(3), entity(4)])
        .unwrap();
    geometry
}
