// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    sketch::{SketchEntityId, residual::PlanarEquation},
    units::UnitsOfMeasure,
};

#[test]
fn reduced_candidate_shape_is_checked() {
    let layout = GroundedLayout::new(&[1.0, 2.0, 3.0], &BTreeSet::from([0, 1]));
    assert!(
        layout
            .restore(&[])
            .unwrap_err()
            .to_string()
            .contains("expected 1")
    );
    assert_eq!(layout.restore(&[7.0]).unwrap(), [1.0, 2.0, 7.0]);
}
fn residual_fixture() -> GroundedResidual {
    let layout = GroundedLayout::new(&[1.0, 2.0], &BTreeSet::from([0, 1]));
    let original = PreparedConstraint {
        projection: None,
        equation: PlanarEquation::Fix([1.0, 2.0]),
        variables: vec![0, 1],
    };
    GroundedResidual::new(original, &layout)
}
#[test]
fn residual_input_shape_is_checked() {
    let residual = residual_fixture();
    let unexpected = DualNumber::seed(&[5.0]).unwrap();
    assert!(
        residual
            .evaluate(&unexpected)
            .unwrap_err()
            .to_string()
            .contains("expected 0")
    );
}
#[test]
fn constant_only_residual_retains_its_equations() {
    let residual = residual_fixture();
    assert_eq!(
        residual
            .evaluate(&[])
            .unwrap()
            .iter()
            .map(DualNumber::value)
            .collect::<Vec<_>>(),
        [0.0, 0.0]
    );
}
#[test]
fn all_active_source_layout_preserves_existing_value_order() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [2.0, 3.0]).unwrap();
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let prepared = prepare(&sketch, &parameters, &[]).unwrap();
    assert_eq!(prepared.values, [2.0, 3.0]);
    assert_eq!(prepared.restore(&prepared.values).unwrap(), prepared.values);
}

#[test]
fn coincident_to_projected_anchor_is_bitwise_exact_after_solve() {
    use crate::sketch::{PlanarConstraint, SketchConstraintId};
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    sketch.add_point(SketchEntityId(2), [0.4, -0.3]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Coincident {
                points: [SketchEntityId(1), SketchEntityId(2)],
            },
        )
        .unwrap();
    let anchor = SketchPoint2D {
        id: SketchEntityId(1),
        position: [2., -3.],
    };
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let result = sketch
        .solve_with_grounded_points(&parameters, &[anchor], Default::default(), || false)
        .unwrap();
    assert_eq!(
        result.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
    assert_eq!(
        result
            .candidate()
            .point(SketchEntityId(2))
            .unwrap()
            .position
            .map(f64::to_bits),
        anchor.position.map(f64::to_bits)
    );
    assert!(result.conflicting_relations().is_empty());
    let rows = result.residual_rows();
    assert_eq!(rows.len(), 2);
    let operands = &rows[0].operands;
    assert_eq!(operands.len(), 4);
    assert_eq!(
        operands.iter().map(|item| item.point).collect::<Vec<_>>(),
        [
            Some((SketchEntityId(1), 0)),
            Some((SketchEntityId(1), 1)),
            Some((SketchEntityId(2), 0)),
            Some((SketchEntityId(2), 1)),
        ]
    );
    assert_eq!(
        operands
            .iter()
            .map(|item| item.externally_grounded)
            .collect::<Vec<_>>(),
        [true, true, false, false]
    );
    assert_eq!(
        operands.iter().map(|item| item.value).collect::<Vec<_>>(),
        [2.0, -3.0, 2.0, -3.0]
    );
    assert_eq!(rows[0].value, operands[0].value - operands[2].value);
    assert_eq!(rows[1].value, operands[1].value - operands[3].value);
    assert_eq!(rows[0].operands, rows[1].operands);
    let (preferred, _) = sketch
        .solve_with_point_preferences(&parameters, &[anchor], &[], Default::default(), || false)
        .unwrap();
    assert_eq!(preferred.residual_rows(), rows);
}

#[test]
fn conflicting_projected_anchors_remain_a_reported_constraint_failure() {
    use crate::sketch::{PlanarConstraint, SketchConstraintId};
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    sketch.add_point(SketchEntityId(2), [1., 0.]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Coincident {
                points: [SketchEntityId(1), SketchEntityId(2)],
            },
        )
        .unwrap();
    let anchors = [
        SketchPoint2D {
            id: SketchEntityId(1),
            position: [0., 0.],
        },
        SketchPoint2D {
            id: SketchEntityId(2),
            position: [1., 0.],
        },
    ];
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let result = sketch
        .solve_with_grounded_points(&parameters, &anchors, Default::default(), || false)
        .unwrap();
    assert_ne!(
        result.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    assert_eq!(result.conflicting_constraints(), [SketchConstraintId(1)]);
    let rows = result.residual_rows();
    assert_eq!([rows[0].value, rows[1].value], [-1.0, 0.0]);
    assert!(
        rows[0]
            .operands
            .iter()
            .all(|operand| operand.externally_grounded)
    );
    assert_eq!(rows[0].operands[0].value - rows[0].operands[2].value, -1.0);
}
