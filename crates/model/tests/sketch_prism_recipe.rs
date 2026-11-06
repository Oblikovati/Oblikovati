// SPDX-License-Identifier: GPL-2.0-only
// This model suite uses the common authored inputs without loading kernel test helpers.
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
use fixture::*;
use oblikovati_model::{
    feature::{
        FeatureReadKey,
        sketch_prism::{SketchPrismDefinition, SketchPrismRecipe},
        work_geometry::DatumLength,
    },
    history::DocumentSnapshot,
};
fn driven_definition() -> SketchPrismDefinition {
    SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(-0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
fn depth_document() -> DocumentSnapshot {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    document
}
#[test]
fn versioned_recipe_round_trip_contains_only_authored_inputs() {
    let original = driven_definition();
    let json = serde_json::to_string(&original.recipe()).unwrap();
    let recipe: SketchPrismRecipe = serde_json::from_str(&json).unwrap();
    let restored = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&original));
    assert_eq!(restored.sketch(), SKETCH);
    assert_eq!(restored.selection(), &selected_sketch_boundary());
    assert_eq!(restored.displacement()[2], DatumLength::Parameter(3));
    assert!(!json.contains("plane"));
    assert!(!json.contains("volume"));
    assert!(!json.contains("verification"));
}
#[test]
fn exact_recipe_comparison_keeps_signed_zero_and_authored_traversal() {
    let original = driven_definition();
    let mut recipe = original.recipe();
    recipe.displacement[0] = DatumLength::Constant(0.);
    assert!(!original.same_recipe(&SketchPrismDefinition::from_recipe(&recipe).unwrap()));
    recipe = original.recipe();
    recipe.selection.outer[0].reversed = true;
    assert!(!original.same_recipe(&SketchPrismDefinition::from_recipe(&recipe).unwrap()));
    recipe = original.recipe();
    recipe.displacement[2] = DatumLength::Parameter(4);
    assert!(!original.same_recipe(&SketchPrismDefinition::from_recipe(&recipe).unwrap()));
}
#[test]
fn reads_attribute_the_current_sketch_dimensions_and_displacement() {
    let document = depth_document();
    let reads = driven_definition().reads(&document).unwrap();
    assert_eq!(
        reads.keys().iter().copied().collect::<Vec<_>>(),
        vec![
            FeatureReadKey::Parameter(1),
            FeatureReadKey::Parameter(2),
            FeatureReadKey::Parameter(3),
            FeatureReadKey::Sketch(1)
        ]
    );
    assert!(!reads.has_unattributed_reads());
}
#[test]
fn displacement_resolves_current_length_units_without_rewriting_sources() {
    let mut document = depth_document();
    let definition = driven_definition();
    let before = definition.recipe();
    assert_eq!(
        definition.resolve(&document).unwrap().displacement,
        [0., 0., 2.]
    );
    document.parameters.set_expression(3, "25 mm").unwrap();
    assert_eq!(
        definition.resolve(&document).unwrap().displacement,
        [0., 0., 2.5]
    );
    document.parameters.set_expression(3, "-3 cm").unwrap();
    assert_eq!(
        definition.resolve(&document).unwrap().displacement,
        [0., 0., -3.]
    );
    assert_eq!(definition.recipe(), before);
}
#[path = "sketch_prism_recipe/placement.rs"]
mod placement;
#[path = "sketch_prism_recipe/refusals.rs"]
mod refusals;
