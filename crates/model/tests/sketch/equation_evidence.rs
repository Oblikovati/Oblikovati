// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn prepared_dimension_evidence_keeps_its_target_and_row_inputs() {
    use oblikovati_model::sketch::{SketchRelationId, SketchTargetSource};
    let outcome = rectangle()
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    let rows = outcome.residual_rows();
    let width = rows
        .iter()
        .find(|row| row.relation == SketchRelationId::Authored(constraint(6)))
        .unwrap();
    assert_eq!(width.operand_indices.as_deref(), Some(&[0, 2][..]));
    let target = &width.targets.as_ref().unwrap()[0];
    assert_eq!(target.source, SketchTargetSource::Parameter(1));
    assert_eq!(target.value, 5.0);
    let a = width.operands[0].value;
    let b = width.operands[2].value;
    assert_eq!(width.value, (a - b).abs() - 5.0);
    let height = rows
        .iter()
        .find(|row| row.relation == SketchRelationId::Authored(constraint(7)))
        .unwrap();
    assert_eq!(height.operand_indices.as_deref(), Some(&[1, 3][..]));
    assert_eq!(
        height.targets.as_ref().unwrap()[0].source,
        SketchTargetSource::Parameter(2)
    );
    assert_eq!(height.targets.as_ref().unwrap()[0].value, 2.5);
}

#[test]
fn unsupported_equation_inputs_are_explicitly_unknown() {
    let mut sketch = Sketch2D::default();
    for (id, point) in [(1, [0., 0.]), (2, [2., 0.]), (3, [1., 0.])] {
        sketch.add_point(entity(id), point).unwrap();
    }
    sketch.add_line(entity(4), [entity(1), entity(2)]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::PointOnLine {
            point: entity(3),
            line: entity(4),
        },
    );
    let outcome = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert!(outcome.numerical().structure.degrees_of_freedom > 0);
    let rows = outcome.residual_rows();
    assert_eq!(rows.len(), 1);
    assert_eq!(rows[0].operand_indices, None);
    assert_eq!(rows[0].targets, None);
}

#[test]
fn projected_radius_keeps_its_target_but_not_an_invented_operand_mapping() {
    use oblikovati_model::sketch::{SketchRelationId, SketchTargetSource};
    let outcome = arcs::driven_arc(1.0, true)
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    let rows = outcome.residual_rows();
    let radius = rows
        .iter()
        .find(|row| row.relation == SketchRelationId::Authored(constraint(4)))
        .unwrap();
    assert_eq!(radius.operand_indices, None);
    assert_eq!(
        radius.targets.as_ref().unwrap()[0].source,
        SketchTargetSource::Parameter(1)
    );
    assert_eq!(radius.targets.as_ref().unwrap()[0].value, 5.0);
}

#[test]
fn interval_equations_enclose_real_coordinate_sources_and_jacobians() {
    use num_rational::BigRational;
    use oblikovati_model::sketch::SketchRelationId;
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(2), [0.0, 0.0]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Coincident {
            points: [entity(1), entity(2)],
        },
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Vertical {
            points: [entity(1), entity(2)],
        },
    );
    let inputs = [[0.1, 0.2], [-1.0, 0.3], [0.7, 1.0], [0.1, 0.5]];
    let rows = sketch.interval_equations(&parameters(), &inputs).unwrap();
    assert_eq!(rows.len(), 4);
    assert_eq!(rows[0].relation, SketchRelationId::Authored(constraint(1)));
    assert_eq!(rows[0].equation, 0);
    assert_exact_enclosure(rows[0].value, inputs[0], inputs[2]);
    assert_exact_enclosure(rows[1].value, inputs[1], inputs[3]);
    assert_eq!(rows[0].jacobian, [[1., 1.], [0., 0.], [-1., -1.], [0., 0.]]);
    assert_eq!(rows[1].jacobian, [[0., 0.], [1., 1.], [0., 0.], [-1., -1.]]);
    assert_eq!(rows[2].value, rows[1].value);
    assert_eq!(rows[3].value, rows[0].value);
    fn exact(value: f64) -> BigRational {
        BigRational::from_float(value).unwrap()
    }
    fn assert_exact_enclosure(result: [f64; 2], left: [f64; 2], right: [f64; 2]) {
        assert!(exact(result[0]) <= exact(left[0]) - exact(right[1]));
        assert!(exact(result[1]) >= exact(left[1]) - exact(right[0]));
    }
}

#[test]
fn interval_equations_decline_unsupported_and_unbounded_inputs() {
    use oblikovati_model::sketch::{SketchIntervalError, SketchRelationId};
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, 0.0],
        },
    );
    let bounds = [[0.0, 0.0], [0.0, 0.0]];
    assert_eq!(
        sketch
            .interval_equations(&parameters(), &bounds)
            .unwrap()
            .len(),
        2
    );
    assert!(matches!(
        sketch.interval_equations(&parameters(), &[[0.0, f64::INFINITY], [0.0, 0.0]]),
        Err(SketchIntervalError::Input(_))
    ));
    assert!(matches!(
        sketch.interval_equations(&parameters(), &[[1.0, -1.0], [0.0, 0.0]]),
        Err(SketchIntervalError::Input(_))
    ));
    assert!(matches!(
        sketch.interval_equations(&parameters(), &[]),
        Err(SketchIntervalError::Input(_))
    ));
    let mut coincidence = Sketch2D::default();
    coincidence.add_point(entity(1), [0.0, 0.0]).unwrap();
    coincidence.add_point(entity(2), [0.0, 0.0]).unwrap();
    relation(
        &mut coincidence,
        2,
        PlanarConstraint::Coincident {
            points: [entity(1), entity(2)],
        },
    );
    let overflow = [
        [f64::MAX, f64::MAX],
        [0.0, 0.0],
        [-f64::MAX, -f64::MAX],
        [0.0, 0.0],
    ];
    assert!(
        matches!(coincidence.interval_equations(&parameters(), &overflow),
        Err(SketchIntervalError::Arithmetic { relation: SketchRelationId::Authored(id), .. }) if id == constraint(2))
    );
}
