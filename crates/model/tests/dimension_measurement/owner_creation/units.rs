// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
fn scaled_owner(scale: f64, display: &str) -> DocumentEditHistory {
    let mut snapshot = snapshot();
    let mut units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
    units.set_preferred(Dimension::Length, display).unwrap();
    snapshot.parameters = ParameterCollection::new(units);
    history(snapshot)
}
fn radius_request(source: &str) -> SketchDimensionCreation {
    let mut input = distance_request(source, false);
    input.operands =
        SketchDimensionalConstraintCreation::Measured(MeasuredDimensionOperands::Radius {
            curve: entity(6),
        });
    input
}
#[test]
fn bare_and_named_length_creation_use_document_display_and_working_units() {
    for scale in [0.01, 1.0, 100.0] {
        for (display, source, canonical) in
            [("mm", "10", 1.0), ("in", "1", 2.54), ("in", "10 mm", 1.0)]
        {
            assert_length_scale(scale, display, source, canonical);
        }
    }
}
fn assert_length_scale(scale: f64, display: &str, source: &str, canonical: f64) {
    let mut history = scaled_owner(scale, display);
    let report = create(&mut history, &radius_request(source)).unwrap();
    assert!(
        (report.measured.value * scale - canonical).abs() < 1e-8,
        "{scale} {display} {source}: {report:?}"
    );
    let parameter = history.snapshot().parameters.get(report.parameter).unwrap();
    assert_eq!(parameter.source, source);
    let target = history
        .snapshot()
        .parameters
        .model_value(report.parameter)
        .unwrap();
    assert_eq!(target.dimension, Dimension::Length);
}

#[test]
fn angular_bare_creation_captures_degrees_without_length_scale_conversion() {
    for scale in [0.01, 1.0, 100.0] {
        let mut history = scaled_owner(scale, "in");
        let mut input = request(
            angular_relation(false),
            Quantity::new(PI / 2.0, Dimension::Angle),
        );
        input.expression = "90".into();
        let report = create(&mut history, &input).unwrap();
        assert_eq!(report.measured.dimension, Dimension::Angle);
        assert!(
            (report.measured.value - PI / 2.0).abs() < 1e-8,
            "{report:?}"
        );
        assert_angular_target(&history, report.parameter);
    }
}

fn assert_angular_target(history: &DocumentEditHistory, parameter: u64) {
    let target = history
        .snapshot()
        .parameters
        .model_value(parameter)
        .unwrap();
    assert_eq!(target, Quantity::new(PI / 2.0, Dimension::Angle));
}
