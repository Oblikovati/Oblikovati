// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use SeededDimensionOperands as Retained;
use SketchDimensionalConstraintCreation::{Measured, Seeded};
#[test]
fn native_dimensional_inputs_preserve_every_measured_family_and_selected_option() {
    for (relation, expected) in base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
    {
        assert_created_input(input(relation), expected);
    }
}
fn assert_created_input(input: SketchDimensionalConstraintCreation, expected: Quantity) {
    let mut geometry = fixture();
    let id = SketchConstraintId(19);
    input.append(&mut geometry, id, 99).unwrap();
    assert_eq!(geometry.dimensional_constraints().count(), 1);
    assert_eq!(
        geometry.dimension_source(id).unwrap(),
        (99, expected.dimension)
    );
    assert_eq!(geometry.measure_dimension(id).unwrap(), expected);
    assert_eq!(input.requires_current_seed(), matches!(input, Seeded(_)));
    let before = geometry.recipe();
    for invalid in [id, SketchConstraintId(0)] {
        assert!(input.append(&mut geometry, invalid, 99).is_err());
        assert_eq!(geometry.recipe(), before);
    }
}
pub(super) fn input(relation: PlanarConstraint) -> SketchDimensionalConstraintCreation {
    match relation {
        PlanarConstraint::Distance {
            points, measure, ..
        } => Measured(MeasuredDimensionOperands::Distance { points, measure }),
        PlanarConstraint::Radius { curve, .. } => {
            Measured(MeasuredDimensionOperands::Radius { curve })
        }
        PlanarConstraint::Diameter { curve, .. } => {
            Measured(MeasuredDimensionOperands::Diameter { curve })
        }
        PlanarConstraint::ArcLength { arc, .. } => {
            Measured(MeasuredDimensionOperands::ArcLength { arc })
        }
        relation => seeded_input(relation),
    }
}
fn seeded_input(relation: PlanarConstraint) -> SketchDimensionalConstraintCreation {
    let operands = match relation {
        PlanarConstraint::Angle { lines, .. } => Retained::Angle(lines),
        PlanarConstraint::ThreePointAngle { vertex, points, .. } => {
            Retained::ThreePointAngle([vertex, points[0], points[1]])
        }
        relation => linear_seeded_input(relation),
    };
    Seeded(operands)
}
fn linear_seeded_input(relation: PlanarConstraint) -> Retained {
    match relation {
        PlanarConstraint::Offset {
            point,
            line,
            linear_diameter,
            ..
        } => offset_operands(point, line, linear_diameter),
        PlanarConstraint::TangentDistance {
            line,
            curve,
            options,
            ..
        } => tangent_operands(line, curve, options),
        unexpected => panic!("{unexpected:?}: expected an offset or tangent-distance family"),
    }
}
#[test]
fn native_constructor_ownership_errors_cannot_append_partial_relations() {
    let mut geometry = fixture();
    for creation in invalid_inputs() {
        let before = geometry.recipe();
        assert!(
            creation
                .append(&mut geometry, SketchConstraintId(1), 99)
                .is_err()
        );
        assert_eq!(geometry.recipe(), before);
    }
}
fn invalid_inputs() -> Vec<SketchDimensionalConstraintCreation> {
    let mut inputs = vec![
        Measured(MeasuredDimensionOperands::Distance {
            points: [entity(1), entity(99)],
            measure: DistanceMeasure::Aligned,
        }),
        Measured(MeasuredDimensionOperands::Radius { curve: entity(8) }),
        Measured(MeasuredDimensionOperands::Diameter { curve: entity(99) }),
        Measured(MeasuredDimensionOperands::ArcLength { arc: entity(6) }),
    ];
    inputs.extend(invalid_seeded_inputs());
    inputs
}

fn invalid_seeded_inputs() -> Vec<SketchDimensionalConstraintCreation> {
    vec![
        Seeded(Retained::Angle([entity(1), entity(8)])),
        Seeded(Retained::ThreePointAngle([
            entity(1),
            entity(2),
            entity(99),
        ])),
        Seeded(Retained::Offset {
            point_line: [entity(6), entity(8)],
            linear_diameter: false,
        }),
        Seeded(Retained::TangentDistance {
            line_curve: [entity(8), entity(9)],
            options: Default::default(),
        }),
    ]
}

fn offset_operands(point: SketchEntityId, line: SketchEntityId, linear_diameter: bool) -> Retained {
    Retained::Offset {
        point_line: [point, line],
        linear_diameter,
    }
}
fn tangent_operands(
    line: SketchEntityId,
    curve: SketchEntityId,
    options: TangentDistanceOptions,
) -> Retained {
    Retained::TangentDistance {
        line_curve: [line, curve],
        options,
    }
}
