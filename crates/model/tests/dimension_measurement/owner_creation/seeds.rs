// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn angle_creation_retains_the_current_solved_branch_instead_of_stale_recipe_coordinates() {
    let snapshot = stale_clockwise_snapshot();
    let mut history = history(snapshot);
    let input = request(
        angular_relation(false),
        Quantity::new(PI / 3.0, Dimension::Angle),
    );
    let report = create(&mut history, &input).unwrap();
    assert_clockwise_relation(&history, report.constraint);
}
fn assert_clockwise_relation(history: &DocumentEditHistory, id: SketchConstraintId) {
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let row = geometry
        .dimensional_constraints()
        .find(|(row, _)| row.id == id)
        .unwrap()
        .0;
    assert_eq!(angle_turn(&row.relation), AngleTurn::Clockwise);
    assert!((geometry.measure_dimension(id).unwrap().value - PI / 3.0).abs() < 1e-8);
}
#[test]
fn seeded_creation_refuses_unavailable_current_geometry_without_reserving_more_ids() {
    let mut history = history(snapshot());
    create(&mut history, &distance_request("futureWidth", false)).unwrap();
    let before = history.sketch_reservations();
    let parameter_mark = history.high_water();
    let input = request(
        angular_relation(false),
        Quantity::new(PI / 3.0, Dimension::Angle),
    );
    assert!(create(&mut history, &input).is_err());
    assert_eq!(history.sketch_reservations(), before);
    assert_eq!(history.high_water(), parameter_mark);
}

fn stale_clockwise_snapshot() -> DocumentSnapshot {
    let mut snapshot = snapshot();
    snapshot
        .parameters
        .insert_scalar_expression(1, "oldAngle", "60 deg", Dimension::Angle)
        .unwrap();
    let geometry = clockwise_geometry();
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    snapshot
}

fn clockwise_geometry() -> Sketch2D {
    let mut geometry = fixture();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Angle {
                lines: [entity(10), entity(8)],
                parameter: 1,
                turn: AngleTurn::Clockwise,
            },
        )
        .unwrap();
    geometry
}

fn angle_turn(relation: &PlanarConstraint) -> AngleTurn {
    match relation {
        PlanarConstraint::Angle { turn, .. } => *turn,
        other => panic!("{other:?}: expected the created angle relation"),
    }
}
