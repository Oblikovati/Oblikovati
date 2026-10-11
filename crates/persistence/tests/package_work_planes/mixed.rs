// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_axis::WorkAxisLine;

fn mixed_package() -> DocumentPackage {
    let mut program = program();
    program.set_end_of_part(None).unwrap();
    let axis = WorkAxisLine::new([-0.0, 2.0, 3.0], [1.0, 2.0, 3.0]).unwrap();
    program
        .append(AuthoredFeature::work_axis(FeatureId(9), "Spindle", axis))
        .unwrap();
    package().with_work_geometry(&program, 12).unwrap()
}
#[test]
fn mixed_package_keeps_exact_axis_sources_and_one_reservation_mark() {
    let captured = mixed_package();
    assert_eq!(section(&captured)["workPlanes"]["recipe"]["version"], 3u64);
    let restored = captured.restore_work_geometry().unwrap();
    assert_eq!(restored.ids.high_water(), 12);
    let decoded = DocumentPackage::decode(&captured.encode().unwrap()).unwrap();
    let round_trip = decoded.restore_work_geometry().unwrap();
    assert!(
        restored
            .program
            .same_work_geometry_recipe(&round_trip.program)
    );
    assert_saved_origin(&round_trip.program);
}
fn assert_saved_origin(program: &FeatureSequence<WorkPlaneDefinition>) {
    let parameters = oblikovati_model::parameter::ParameterCollection::new(
        oblikovati_model::units::UnitsOfMeasure::default(),
    );
    let evaluated = program.evaluate_work_geometry(&parameters).unwrap();
    assert_eq!(
        evaluated.axis(FeatureId(9)).unwrap().origin()[0].to_bits(),
        (-0.0f64).to_bits()
    );
}
#[test]
fn axes_in_old_program_versions_are_refused_without_changing_bytes() {
    let captured = mixed_package();
    for version in [1u64, 2] {
        let mut model = section(&captured);
        model["workPlanes"]["recipe"]["version"] = version.into();
        let invalid = from_section(&model, 3);
        let before = invalid.encode().unwrap();
        let error = invalid.restore_work_geometry().unwrap_err();
        assert!(error.to_string().contains("axis FeatureId(9)"));
        assert!(error.to_string().contains("expected program version 3"));
        assert_eq!(invalid.encode().unwrap(), before);
    }
}
