// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::SketchBlockDefinitionRecipe;

#[test]
fn empty_block_definitions_round_trip_in_creation_order() {
    let mut sketches = document();
    let before = sketches.clone();
    assert_eq!(sketches.define_empty_block("Rib").unwrap(), 0);
    assert_eq!(sketches.define_empty_block("Hole pattern").unwrap(), 1);
    assert!(!sketches.same_recipe(&before));
    let recipe = sketches.recipe();
    assert_eq!(recipe.version, 8);
    let encoded = serde_json::to_string(&recipe).unwrap();
    let decoded: DocumentSketchRecipe = serde_json::from_str(&encoded).unwrap();
    let restored = DocumentSketches::from_recipe(&decoded).unwrap();
    assert!(sketches.same_recipe(&restored));
    assert_eq!(
        restored
            .block_definitions()
            .iter()
            .map(|item| item.name())
            .collect::<Vec<_>>(),
        ["Rib", "Hole pattern"]
    );
}
#[test]
fn invalid_names_and_duplicate_definitions_leave_the_registry_unchanged() {
    let mut sketches = document();
    sketches.define_empty_block("Rib").unwrap();
    for name in ["", "   ", "bad\0name", "Rib"] {
        let before = sketches.clone();
        assert!(sketches.define_empty_block(name).is_err());
        assert!(sketches.same_recipe(&before));
    }
    let before = sketches.clone();
    assert!(sketches.delete_empty_block("missing").is_err());
    assert!(sketches.same_recipe(&before));
    sketches.delete_empty_block("Rib").unwrap();
    assert!(sketches.block_definitions().is_empty());
}
#[test]
fn malformed_or_legacy_block_recipes_are_refused() {
    let mut recipe = document().recipe();
    recipe.block_definitions = vec![SketchBlockDefinitionRecipe { name: "Rib".into() }];
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
    recipe.version = 8;
    recipe
        .block_definitions
        .push(recipe.block_definitions[0].clone());
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
}
