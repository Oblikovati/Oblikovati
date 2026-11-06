// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::parameter::ParameterCollection;
#[test]
fn every_dimensional_form_can_measure_without_reading_or_constraining_its_target() {
    for (relation, expected) in base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
    {
        assert_reference_form(relation, expected);
    }
}
fn assert_reference_form(relation: PlanarConstraint, expected: Quantity) {
    let mut geometry = fixture();
    let id = SketchConstraintId(1);
    geometry.add_constraint(id, relation).unwrap();
    let parameters = ParameterCollection::new(Default::default());
    assert_driving_source(&geometry, &parameters);
    geometry.set_dimension_reference(id, true).unwrap();
    assert_reference_measurement(&geometry, &parameters, expected);
    geometry.set_dimension_reference(id, false).unwrap();
    assert_driving_source(&geometry, &parameters);
}
fn assert_reference_measurement(
    geometry: &Sketch2D,
    parameters: &ParameterCollection<u64>,
    expected: Quantity,
) {
    assert!(geometry.parameter_dependencies().is_empty());
    let result = geometry
        .solve(parameters, Default::default(), || false)
        .unwrap();
    assert!(result.parameter_dependencies().is_empty());
    assert_eq!(
        geometry.measure_dimension(SketchConstraintId(1)).unwrap(),
        expected
    );
    assert!(geometry.same_recipe(result.candidate()));
}

fn assert_driving_source(geometry: &Sketch2D, parameters: &ParameterCollection<u64>) {
    assert_eq!(geometry.parameter_dependencies(), [99]);
    let result = geometry.solve(parameters, Default::default(), || false);
    assert!(result.is_err());
}
