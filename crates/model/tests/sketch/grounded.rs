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
