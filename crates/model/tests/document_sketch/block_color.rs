// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE};
use oblikovati_model::document_sketch::SketchBlockInstance;

fn color(r: u8, g: u8, b: u8) -> Color {
    Color {
        r,
        g,
        b,
        opacity: 1.0,
        source: OVERRIDE_COLOR_SOURCE,
    }
}
fn placed_block() -> DocumentSketches {
    let mut sketches = document();
    let sketch = DocumentSketchId(1);
    sketches
        .capture_block_definition("Rib", sketch, &[SketchEntityId(4)])
        .unwrap();
    let instance = SketchBlockInstance::new(SketchEntityId(20), "Rib", [1.0; 2], 0.0, 1.0).unwrap();
    sketches.insert_block_instance(sketch, instance).unwrap();
    sketches
}
#[test]
fn independent_colors_round_trip_and_clear_without_changing_old_recipes() {
    let mut sketches = placed_block();
    let original = sketches.clone();
    let cyan = color(0, 255, 255);
    let red = color(255, 0, 0);
    sketches
        .set_block_definition_color("Rib", Some(cyan.clone()))
        .unwrap();
    sketches
        .set_block_instance_color(DocumentSketchId(1), SketchEntityId(20), Some(red.clone()))
        .unwrap();
    let recipe = sketches.recipe();
    assert_eq!(recipe.version, 16);
    let bytes = serde_json::to_vec(&recipe).unwrap();
    let restored = DocumentSketches::from_recipe(&serde_json::from_slice(&bytes).unwrap()).unwrap();
    assert!(restored.same_recipe(&sketches));
    assert_eq!(restored.block_definitions()[0].color(), Some(&cyan));
    assert_eq!(
        restored.get(DocumentSketchId(1)).unwrap().block_instances()[0].color(),
        Some(&red)
    );
    let mut legacy = recipe;
    legacy.version = 15;
    assert!(DocumentSketches::from_recipe(&legacy).is_err());
    sketches.set_block_definition_color("Rib", None).unwrap();
    sketches
        .set_block_instance_color(DocumentSketchId(1), SketchEntityId(20), None)
        .unwrap();
    assert!(sketches.same_recipe(&original));
    assert_eq!(sketches.recipe().version, 11);
}
#[test]
fn invalid_color_or_identity_never_changes_the_document() {
    let mut sketches = placed_block();
    let before = sketches.clone();
    let mut bad = color(0, 255, 255);
    bad.opacity = f64::NAN;
    assert!(
        sketches
            .set_block_definition_color("Rib", Some(bad))
            .is_err()
    );
    assert!(
        sketches
            .set_block_definition_color("missing", None)
            .is_err()
    );
    assert!(
        sketches
            .set_block_instance_color(DocumentSketchId(1), SketchEntityId(99), None)
            .is_err()
    );
    assert!(sketches.same_recipe(&before));
}
