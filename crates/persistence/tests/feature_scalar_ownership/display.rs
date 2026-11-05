// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_geometry::WorkPlaneDisplaySettings;

fn displayed_package() -> DocumentPackage {
    let mut program = sequence();
    let corners = [[1.0, -0.0, 0.0], [3.0, 4.0, 0.0]];
    let settings = WorkPlaneDisplaySettings::default()
        .edited(Some(true), Some(true), Some(corners))
        .unwrap();
    program.set_plane_display(FeatureId(3), settings).unwrap();
    stored().with_work_geometry(&program, 3).unwrap()
}
#[test]
fn reopened_settings_preserve_physical_corners_and_signed_zero() {
    let reopened = DocumentPackage::decode(&displayed_package().encode().unwrap()).unwrap();
    let restored = reopened.restore_work_geometry().unwrap().program;
    let settings = restored.get(FeatureId(3)).unwrap().plane_display_settings();
    assert!(!settings.auto_resize() && settings.grounded());
    let corners = settings
        .current_corners(None, &Default::default())
        .unwrap()
        .unwrap();
    assert_eq!(corners[0][1].to_bits(), (-0.0_f64).to_bits());
    assert_eq!(corners[1], [3.0, 4.0, 0.0]);
    let parameters = reopened.restore_document_parameters().unwrap().parameters;
    assert_eq!(parameters.get(9).unwrap().source, "width / 2");
}
#[test]
fn older_program_versions_and_unknown_display_fields_are_refused() {
    let encoded = String::from_utf8(displayed_package().encode().unwrap()).unwrap();
    for (from, to) in [
        ("version: 9", "version: 8"),
        ("grounded: true", "unknown: true"),
    ] {
        let invalid = encoded.replace(from, to);
        assert_ne!(invalid, encoded);
        let package = DocumentPackage::decode(invalid.as_bytes()).unwrap();
        assert!(package.restore_work_geometry().is_err());
    }
}
#[test]
fn nonfinite_stored_rectangle_is_refused_before_program_adoption() {
    let encoded = String::from_utf8(displayed_package().encode().unwrap()).unwrap();
    let invalid = encoded.replace("- 3.0", "- .inf");
    assert_ne!(invalid, encoded);
    let package = DocumentPackage::decode(invalid.as_bytes()).unwrap();
    let error = package.restore_work_geometry().err().unwrap();
    assert!(error.to_string().contains("finite three-component"));
}
