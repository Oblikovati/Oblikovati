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
fn legacy_line_boundaries_restore_as_directed_curves() {
    let original = driven_definition();
    let mut encoded = serde_json::to_value(original.recipe()).unwrap();
    for edge in encoded["selection"]["outer"].as_array_mut().unwrap() {
        let object = edge.as_object_mut().unwrap();
        let curve = object.remove("curve").unwrap();
        object.insert("line".into(), curve);
    }
    let recipe: SketchPrismRecipe = serde_json::from_value(encoded).unwrap();
    assert!(
        SketchPrismDefinition::from_recipe(&recipe)
            .unwrap()
            .same_recipe(&original)
    );
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

#[test]
fn cut_recipe_retains_a_body_name_and_tracks_its_producer() {
    use oblikovati_model::{
        feature::FeatureId,
        identity::{SketchPrismReferenceRole, sketch_prism_reference_name},
    };
    let document = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let source = FeatureId(42);
    let body =
        sketch_prism_reference_name(document, source, SKETCH, SketchPrismReferenceRole::Body)
            .unwrap();
    let cap =
        sketch_prism_reference_name(document, source, SKETCH, SketchPrismReferenceRole::EndCap)
            .unwrap();
    assert!(driven_definition().with_cut_target(cap).is_err());
    let authored = driven_definition().with_cut_target(body.clone()).unwrap();
    let encoded = serde_json::to_vec(&authored.recipe()).unwrap();
    let restored =
        SketchPrismDefinition::from_recipe(&serde_json::from_slice(&encoded).unwrap()).unwrap();
    assert!(restored.same_recipe(&authored));
    let target = restored.cut_target().unwrap();
    assert_eq!(target.key(), body);
    assert_eq!(target.source(), source);
    assert!(
        restored
            .source_reads()
            .keys()
            .contains(&FeatureReadKey::Feature(source))
    );
}

#[test]
fn through_all_recipe_requires_the_body_its_extent_depends_on() {
    use oblikovati_model::{
        feature::{
            FeatureId,
            sketch_prism::{SketchPrismExtent, ThroughAllDirection},
        },
        identity::{SketchPrismReferenceRole, sketch_prism_reference_name},
    };
    let document = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let key = sketch_prism_reference_name(
        document,
        FeatureId(42),
        SKETCH,
        SketchPrismReferenceRole::Body,
    )
    .unwrap();
    let extent = SketchPrismExtent::ThroughAll {
        direction: ThroughAllDirection::Negative,
    };
    let definition =
        SketchPrismDefinition::between_planes(SKETCH, selected_sketch_boundary(), extent.clone())
            .unwrap();
    assert!(SketchPrismDefinition::from_recipe(&definition.recipe()).is_err());
    let definition = definition.with_cut_target(key.clone()).unwrap();
    let bytes = serde_json::to_vec(&definition.recipe()).unwrap();
    let restored =
        SketchPrismDefinition::from_recipe(&serde_json::from_slice(&bytes).unwrap()).unwrap();
    assert_eq!(restored.extent(), Some(&extent));
    assert_eq!(restored.cut_target().unwrap().key(), key);
}

#[test]
fn join_recipe_preserves_the_target_and_refuses_conflicting_operations() {
    use oblikovati_model::{
        feature::{FeatureId, sketch_prism::PrismBoolean},
        identity::{SketchPrismReferenceRole, sketch_prism_reference_name},
    };
    let document = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let key = sketch_prism_reference_name(
        document,
        FeatureId(42),
        SKETCH,
        SketchPrismReferenceRole::Body,
    )
    .unwrap();
    let original = driven_definition()
        .with_boolean_target(PrismBoolean::Join, key.clone())
        .unwrap();
    let mut recipe = original.recipe();
    assert_eq!(recipe.join_target.as_deref(), Some(key.as_slice()));
    assert!(recipe.cut_target.is_none());
    let restored = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    assert!(original.same_recipe(&restored));
    assert_eq!(
        restored.boolean_target().unwrap().operation(),
        PrismBoolean::Join
    );
    recipe.cut_target = Some(key);
    assert!(SketchPrismDefinition::from_recipe(&recipe).is_err());
}
