// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        work_geometry::{DatumLength, WorkGeometryDefinition},
    },
    parameter::ParameterCollection,
    units::UnitsOfMeasure,
};
use std::error::Error;

fn point_program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    let definition = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    program
        .append(AuthoredFeature::datum(FeatureId(1), "Point", definition))
        .unwrap();
    program
}
#[test]
fn visibility_does_not_change_construction_state_or_current_coordinates() {
    let mut program = point_program();
    program
        .set_work_plane_construction(FeatureId(1), true)
        .unwrap();
    let before = program.clone();
    program.set_datum_visible(FeatureId(1), false).unwrap();
    assert!(!program.get(FeatureId(1)).unwrap().is_visible());
    assert!(program.get(FeatureId(1)).unwrap().is_construction_datum());
    assert!(!program.same_work_geometry_recipe(&before));
    assert_visibility_preserves_geometry(&mut program);
}
fn assert_visibility_preserves_geometry(program: &mut FeatureSequence<WorkGeometryDefinition>) {
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    assert_eq!(
        program
            .evaluate_work_geometry(&parameters)
            .unwrap()
            .point(FeatureId(1))
            .unwrap(),
        [0.0; 3]
    );
    assert!(
        program
            .set_datum_visible(FeatureId(99), false)
            .unwrap_err()
            .to_string()
            .contains("99")
    );
}
#[test]
fn replacing_a_hidden_definition_retains_the_visibility_metadata() {
    let mut program = point_program();
    program.set_datum_visible(FeatureId(1), false).unwrap();
    let replacement = WorkGeometryDefinition::Point([DatumLength::Constant(3.0); 3]);
    program
        .replace(FeatureId(1), replacement, replacement.reads())
        .unwrap();
    assert!(!program.get(FeatureId(1)).unwrap().is_visible());
}
#[test]
fn hidden_metadata_uses_version_five_and_legacy_records_keep_visible_defaults() {
    let mut program = point_program();
    let visible = program.work_geometry_recipe().unwrap();
    let bytes = serde_json::to_vec(&visible).unwrap();
    assert!(
        !String::from_utf8(bytes.clone())
            .unwrap()
            .contains("visible")
    );
    let decoded = serde_json::from_slice(&bytes).unwrap();
    let restored = FeatureSequence::from_work_geometry_recipe(&decoded).unwrap();
    assert!(restored.get(FeatureId(1)).unwrap().is_visible());
    program.set_datum_visible(FeatureId(1), false).unwrap();
    assert_hidden_recipe(&program);
}
fn assert_hidden_recipe(program: &FeatureSequence<WorkGeometryDefinition>) {
    let mut recipe = program.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 5);
    let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(program.same_work_geometry_recipe(&restored));
    recipe.version = 4;
    let error = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap_err();
    assert!(
        error
            .to_string()
            .contains("visible=false: expected program version 5")
    );
    assert!(error.source().is_none());
}
