// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    quantity::{Dimension, Quantity},
    sketch::*,
};
use std::f64::consts::PI;
fn entity(id: u64) -> SketchEntityId {
    SketchEntityId(id)
}
fn fixture() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (offset, position) in [[0.0, 0.0], [3.0, 4.0], [0.0, 4.0], [4.0, 0.0], [4.0, 4.0]]
        .into_iter()
        .enumerate()
    {
        sketch
            .add_point(entity(offset as u64 + 1), position)
            .unwrap();
    }
    sketch.add_circle(entity(6), entity(1), 2.0).unwrap();
    sketch
        .add_arc(entity(7), entity(1), [entity(3), entity(4)], false)
        .unwrap();
    sketch.add_line(entity(8), [entity(1), entity(3)]).unwrap();
    sketch.add_line(entity(9), [entity(3), entity(4)]).unwrap();
    sketch.add_line(entity(10), [entity(1), entity(2)]).unwrap();
    append_tangent_lines(&mut sketch);
    sketch
}
fn length(value: f64) -> Quantity {
    Quantity::new(value, Dimension::Length)
}
fn base_cases() -> Vec<(PlanarConstraint, Quantity)> {
    let mut cases = distance_cases();
    for (curve, diameter, value) in [
        (6, false, 2.0),
        (7, false, 4.0),
        (6, true, 4.0),
        (7, true, 8.0),
    ] {
        let relation = radial_relation(curve, diameter);
        cases.push((relation, length(value)));
    }
    cases.push((
        PlanarConstraint::ArcLength {
            arc: entity(7),
            parameter: 99,
        },
        length(2.0 * PI),
    ));
    cases
}
fn distance_cases() -> Vec<(PlanarConstraint, Quantity)> {
    [
        (DistanceMeasure::Aligned, 5.0),
        (DistanceMeasure::Horizontal, 3.0),
        (DistanceMeasure::Vertical, 4.0),
    ]
    .into_iter()
    .map(|(measure, value)| {
        (
            PlanarConstraint::Distance {
                points: [entity(1), entity(2)],
                measure,
                parameter: 99,
            },
            length(value),
        )
    })
    .collect()
}
fn angle_cases() -> Vec<(PlanarConstraint, Quantity)> {
    let value = Quantity::new(3.0_f64.atan2(4.0), Dimension::Angle);
    [angular_relation(false), angular_relation(true)]
        .into_iter()
        .map(|relation| (relation, value))
        .collect()
}
fn angular_relation(three_points: bool) -> PlanarConstraint {
    if three_points {
        return PlanarConstraint::ThreePointAngle {
            vertex: entity(1),
            points: [entity(2), entity(3)],
            parameter: 99,
            turn: AngleTurn::CounterClockwise,
        };
    }
    PlanarConstraint::Angle {
        lines: [entity(10), entity(8)],
        parameter: 99,
        turn: AngleTurn::CounterClockwise,
    }
}
fn offset_cases() -> Vec<(PlanarConstraint, Quantity)> {
    let mut cases = Vec::new();
    for diameter in [false, true] {
        let distance = 4.0 / 2.0_f64.sqrt();
        let multiplier = if diameter { 2.0 } else { 1.0 };
        cases.push(offset_case(diameter, multiplier * distance));
        for far_side in [false, true] {
            cases.push(tangent_case(diameter, far_side, multiplier * distance));
        }
    }
    cases.extend(arc_tangent_cases());
    cases
}
fn tangent_case(diameter: bool, far_side: bool, distance: f64) -> (PlanarConstraint, Quantity) {
    let multiplier = if diameter { 2.0 } else { 1.0 };
    let value = distance
        + if far_side {
            2.0 * multiplier
        } else {
            -2.0 * multiplier
        };
    let relation = PlanarConstraint::TangentDistance {
        line: entity(9),
        curve: entity(6),
        parameter: 99,
        options: TangentDistanceOptions {
            far_side,
            linear_diameter: diameter,
        },
        contact: right_contact(),
    };
    (relation, length(value))
}
#[test]
fn every_dimensional_measure_is_geometric_without_a_backing_parameter_or_a_solve() {
    for (relation, expected) in base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
    {
        assert_measurement(relation, expected);
    }
}
fn assert_measurement(relation: PlanarConstraint, expected: Quantity) {
    let mut sketch = fixture();
    sketch
        .add_constraint(SketchConstraintId(17), relation)
        .unwrap();
    let before = sketch.recipe();
    let measured = sketch.measure_dimension(SketchConstraintId(17)).unwrap();
    assert_eq!(measured.dimension, expected.dimension);
    let tolerance = oblikovati_solve::nonlinear::SolveOptions::default().relative_tolerance
        * expected.value.abs().max(1.0);
    assert!(
        (measured.value - expected.value).abs() <= tolerance,
        "{measured:?}, expected {expected:?}"
    );
    assert_eq!(sketch.recipe(), before);
}
#[test]
fn dimensional_enumeration_is_ordered_and_excludes_geometric_and_structural_sources() {
    let mut sketch = fixture();
    sketch
        .add_constraint(
            SketchConstraintId(44),
            PlanarConstraint::LineHorizontal { line: entity(9) },
        )
        .unwrap();
    append_dimension_cases(&mut sketch);
    let ids = sketch
        .dimensional_constraints()
        .map(|(constraint, parameter)| (constraint.id.0, parameter))
        .collect::<Vec<_>>();
    assert_eq!(ids, (1..=24).map(|id| (id, 99)).collect::<Vec<_>>());
    assert_eq!(sketch.geometric_constraints().count(), 1);
    assert_eq!(sketch.parameter_dependencies(), [99]);
}
#[test]
fn geometric_and_missing_constraint_ids_cannot_be_measured_as_dimensions() {
    let mut sketch = fixture();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::LineHorizontal { line: entity(10) },
        )
        .unwrap();
    for id in [0, 1, 99] {
        let cause = sketch
            .measure_dimension(SketchConstraintId(id))
            .unwrap_err()
            .to_string();
        assert!(cause.contains(&id.to_string()) && cause.contains("owned dimensional relation"));
    }
}
#[test]
fn degenerate_dimension_measurements_are_named_declines_without_a_zero_substitute() {
    let sketch = degenerate_fixture();
    let before = sketch.recipe();
    let cause = sketch
        .measure_dimension(SketchConstraintId(5))
        .unwrap_err()
        .to_string();
    assert!(cause.contains("5") && cause.contains("defined geometric measurement"));
    assert_eq!(sketch.recipe(), before);
}

fn offset_case(diameter: bool, value: f64) -> (PlanarConstraint, Quantity) {
    (
        PlanarConstraint::Offset {
            point: entity(1),
            line: entity(9),
            parameter: 99,
            side: LineContactSide::Right,
            linear_diameter: diameter,
        },
        length(value),
    )
}

fn right_contact() -> TangentDistanceContact {
    TangentDistanceContact {
        side: LineContactSide::Right,
        lean: None,
    }
}

fn append_dimension_cases(sketch: &mut Sketch2D) {
    for (offset, (relation, _)) in base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
        .enumerate()
    {
        sketch
            .add_constraint(SketchConstraintId(offset as u64 + 1), relation)
            .unwrap();
    }
}

fn degenerate_fixture() -> Sketch2D {
    let mut sketch = fixture();
    sketch.add_line(entity(13), [entity(1), entity(1)]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(5),
            PlanarConstraint::Offset {
                point: entity(2),
                line: entity(13),
                parameter: 99,
                side: LineContactSide::Left,
                linear_diameter: false,
            },
        )
        .unwrap();
    sketch
}

fn radial_relation(curve: u64, diameter: bool) -> PlanarConstraint {
    if diameter {
        PlanarConstraint::Diameter {
            curve: entity(curve),
            parameter: 99,
        }
    } else {
        PlanarConstraint::Radius {
            curve: entity(curve),
            parameter: 99,
        }
    }
}

#[path = "dimension_measurement/generated.rs"]
mod generated;

fn arc_tangent_cases() -> Vec<(PlanarConstraint, Quantity)> {
    let mut cases = Vec::new();
    for shared in [false, true] {
        for diameter in [false, true] {
            for far_side in [false, true] {
                cases.push(arc_tangent_case(shared, diameter, far_side));
            }
        }
    }
    cases
}
fn arc_tangent_case(shared: bool, diameter: bool, far_side: bool) -> (PlanarConstraint, Quantity) {
    let contact = arc_contact(shared);
    let line = entity(if shared { 12 } else { 11 });
    let relation = PlanarConstraint::TangentDistance {
        line,
        curve: entity(7),
        parameter: 99,
        options: TangentDistanceOptions {
            far_side,
            linear_diameter: diameter,
        },
        contact,
    };
    let measured = match (far_side, diameter) {
        (false, _) => 0.0,
        (true, false) => 8.0,
        (true, true) => 16.0,
    };
    (relation, length(measured))
}

fn append_tangent_lines(sketch: &mut Sketch2D) {
    sketch.add_line(entity(11), [entity(2), entity(5)]).unwrap();
    sketch.add_line(entity(12), [entity(3), entity(5)]).unwrap();
}

fn arc_contact(shared: bool) -> TangentDistanceContact {
    TangentDistanceContact {
        side: LineContactSide::Right,
        lean: shared.then_some(TangentLean::Positive),
    }
}

#[test]
fn every_dimension_form_can_be_removed_without_changing_surviving_identity_order() {
    for (relation, _) in base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
    {
        assert_dimension_removal(relation);
    }
}
fn assert_dimension_removal(relation: PlanarConstraint) {
    let mut geometry = fixture();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::LineHorizontal { line: entity(9) },
        )
        .unwrap();
    geometry
        .add_constraint(SketchConstraintId(7), relation.clone())
        .unwrap();
    geometry
        .add_constraint(SketchConstraintId(9), relation)
        .unwrap();
    verify_dimension_removal(geometry);
}
fn verify_dimension_removal(mut geometry: Sketch2D) {
    let removed = geometry
        .remove_dimensional_constraint(SketchConstraintId(7))
        .unwrap();
    assert_eq!(removed.id, SketchConstraintId(7));
    let survivor_ids: Vec<_> = geometry.constraints().iter().map(|row| row.id.0).collect();
    assert_eq!(survivor_ids, [1, 9]);
    let before = geometry.recipe();
    for id in [1, 7, 99] {
        assert!(
            geometry
                .remove_dimensional_constraint(SketchConstraintId(id))
                .is_err()
        );
    }
    assert_eq!(geometry.recipe(), before);
}
