// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn flipped_package() -> DocumentPackage {
    let mut program = sequence();
    program.flip_plane_normal(FeatureId(3)).unwrap();
    stored().with_work_geometry(&program, 3).unwrap()
}

#[test]
fn reopened_flip_retains_owned_expression_and_recomputes_current_orientation() {
    let reopened = DocumentPackage::decode(&flipped_package().encode().unwrap()).unwrap();
    let program = reopened.restore_work_geometry().unwrap().program;
    let mut parameters = reopened.restore_document_parameters().unwrap().parameters;
    assert!(program.get(FeatureId(3)).unwrap().is_normal_flipped());
    assert_eq!(
        program.get(FeatureId(3)).unwrap().owned_scalars()[0].parameter,
        9
    );
    assert_eq!(parameters.get(9).unwrap().source, "width / 2");
    parameters.set_expression(1, "8 cm").unwrap();
    let evaluated = program.evaluate_work_geometry(&parameters).unwrap();
    let plane = evaluated.plane(FeatureId(3)).unwrap();
    assert_eq!(plane.recipe().origin, [0.0, 0.0, 4.0]);
    assert_eq!(plane.normal(), [0.0, 0.0, -1.0]);
}

#[test]
fn package_rejects_flip_metadata_in_an_older_program_version() {
    let encoded = String::from_utf8(flipped_package().encode().unwrap()).unwrap();
    let invalid = encoded.replace("version: 8", "version: 7");
    assert_ne!(invalid, encoded);
    let package = DocumentPackage::decode(invalid.as_bytes()).unwrap();
    let error = package.restore_work_geometry().err().unwrap();
    assert!(error.to_string().contains("expected program version 8"));
}
