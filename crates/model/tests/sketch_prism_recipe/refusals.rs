// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{feature::sketch_prism::SketchPrismCause, sketch::SketchEntityId};
use std::error::Error;
#[test]
fn invalid_versions_identities_and_scalar_sources_fail_before_adoption() {
    let baseline = driven_definition().recipe();
    let mut invalid = baseline.clone();
    invalid.sketch.0 = 0;
    assert!(SketchPrismDefinition::from_recipe(&invalid).is_err());
    invalid = baseline.clone();
    invalid.selection.outer[0].curve = SketchEntityId(0);
    assert!(SketchPrismDefinition::from_recipe(&invalid).is_err());
    invalid = baseline.clone();
    invalid
        .selection
        .holes
        .push(invalid.selection.outer.clone());
    assert!(SketchPrismDefinition::from_recipe(&invalid).is_err());
    verify_invalid_scalars(&baseline);
}
fn verify_invalid_scalars(baseline: &SketchPrismRecipe) {
    for source in [
        DatumLength::Parameter(0),
        DatumLength::Constant(f64::NAN),
        DatumLength::Constant(f64::INFINITY),
        DatumLength::Constant(0.),
    ] {
        let mut invalid = baseline.clone();
        invalid.displacement[2] = source;
        let error = SketchPrismDefinition::from_recipe(&invalid).unwrap_err();
        assert_eq!(error.sketch, SKETCH);
        assert!(error.to_string().contains("expected"));
    }
}
#[test]
fn failed_wrong_unit_and_zero_current_values_never_use_last_good_depth() {
    let mut document = depth_document();
    let definition = driven_definition();
    assert_eq!(definition.resolve(&document).unwrap().displacement[2], 2.);
    document.parameters.set_expression(3, "missing").unwrap();
    let failed = definition.resolve(&document).unwrap_err();
    assert!(matches!(
        failed.cause.as_ref(),
        SketchPrismCause::Length { axis: 2, .. }
    ));
    assert!(failed.source().unwrap().source().is_some());
}
#[test]
fn missing_sketch_keeps_typed_owner_declines() {
    let document = depth_document();
    let mut recipe = driven_definition().recipe();
    recipe.sketch.0 = 99;
    let missing = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    assert!(matches!(
        missing.reads(&document).unwrap_err().cause.as_ref(),
        SketchPrismCause::Sketch(_)
    ));
    assert!(matches!(
        missing.resolve(&document).unwrap_err().cause.as_ref(),
        SketchPrismCause::Placement(_)
    ));
}
#[test]
fn unknown_recipe_fields_are_not_silently_accepted() {
    let mut encoded = serde_json::to_value(driven_definition().recipe()).unwrap();
    encoded["cached_volume"] = serde_json::json!(24.);
    assert!(serde_json::from_value::<SketchPrismRecipe>(encoded).is_err());
}

#[test]
fn unsupported_recipe_version_is_a_named_refusal() {
    let baseline = driven_definition().recipe();
    let mut invalid = baseline.clone();
    invalid.version = 2;
    assert!(
        SketchPrismDefinition::from_recipe(&invalid)
            .unwrap_err()
            .to_string()
            .contains("expected version 1")
    );
}

#[test]
fn wrong_unit_and_zero_current_depth_are_named_refusals() {
    let mut document = depth_document();
    let definition = driven_definition();
    document.parameters.set_expression(3, "3 deg").unwrap();
    assert!(
        definition
            .resolve(&document)
            .unwrap_err()
            .to_string()
            .contains("finite Length")
    );
    document.parameters.set_expression(3, "0 cm").unwrap();
    assert!(
        definition
            .resolve(&document)
            .unwrap_err()
            .to_string()
            .contains("nonzero profile-normal")
    );
}

#[test]
fn stale_geometry_keeps_the_typed_owner_decline() {
    let mut document = depth_document();
    document.parameters.set_expression(1, "6 cm").unwrap();
    assert!(matches!(
        driven_definition()
            .resolve(&document)
            .unwrap_err()
            .cause
            .as_ref(),
        SketchPrismCause::Placement(_)
    ));
}
