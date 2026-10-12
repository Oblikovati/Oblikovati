// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::retain_sketch_prism_definition;
use oblikovati_model::feature::{
    sketch_prism::{SketchPrismDefinition, SketchPrismRecipe},
    work_geometry::DatumLength,
};
fn driven_prism() -> SketchPrismDefinition {
    SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
#[test]
fn restored_parameter_backed_definition_builds_current_kernel_geometry() {
    let mut document = driven_document();
    let encoded = serde_json::to_string(&driven_prism().recipe()).unwrap();
    let recipe: SketchPrismRecipe = serde_json::from_str(&encoded).unwrap();
    let definition = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    let first = retain_sketch_prism_definition(&document, &definition).unwrap();
    assert_volume(first.report().properties.volume, 24.);
    document.parameters.set_expression(3, "30 mm").unwrap();
    let deeper = retain_sketch_prism_definition(&document, &definition).unwrap();
    assert_volume(deeper.report().properties.volume, 36.);
    document.parameters.set_expression(3, "missing").unwrap();
    assert!(retain_sketch_prism_definition(&document, &definition).is_err());
    assert_volume(first.report().properties.volume, 24.);
    assert_eq!(
        serde_json::to_string(&definition.recipe()).unwrap(),
        encoded
    );
}

fn driven_document() -> oblikovati_model::history::DocumentSnapshot {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    document
}
