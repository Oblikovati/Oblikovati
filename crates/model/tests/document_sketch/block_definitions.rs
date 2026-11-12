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
    assert!(sketches.delete_block_definition("missing").is_err());
    assert!(sketches.same_recipe(&before));
    sketches.delete_block_definition("Rib").unwrap();
    assert!(sketches.block_definitions().is_empty());
}
#[test]
fn malformed_or_legacy_block_recipes_are_refused() {
    let mut recipe = document().recipe();
    recipe.block_definitions = vec![SketchBlockDefinitionRecipe {
        name: "Rib".into(),
        geometry: None,
    }];
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
    recipe.version = 8;
    recipe
        .block_definitions
        .push(recipe.block_definitions[0].clone());
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
}
#[test]
fn definition_recipes_refuse_constraints_owned_by_a_sketch() {
    let source = document();
    let mut recipe = source.recipe();
    recipe.version = 9;
    recipe.block_definitions.push(SketchBlockDefinitionRecipe {
        name: "Rib".into(),
        geometry: Some(source.get(DocumentSketchId(1)).unwrap().geometry().recipe()),
    });
    assert!(
        DocumentSketches::from_recipe(&recipe)
            .unwrap_err()
            .to_string()
            .contains("constraints")
    );
}

#[test]
fn captured_geometry_keeps_shared_points_and_does_not_change_the_source() {
    let mut sketches = document();
    let before = sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    let ids = [SketchEntityId(4), SketchEntityId(1), SketchEntityId(4)];
    sketches
        .capture_block_definition("Rib", DocumentSketchId(1), &ids)
        .unwrap();
    let definition = &sketches.block_definitions()[0];
    assert_eq!(definition.geometry().lines().len(), 1);
    assert_eq!(definition.geometry().points().len(), 2);
    assert_eq!(
        definition
            .geometry()
            .line(SketchEntityId(4))
            .unwrap()
            .endpoints,
        [SketchEntityId(1), SketchEntityId(2)]
    );
    assert!(definition.geometry().constraints().is_empty());
    assert!(
        sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry()
            .same_recipe(&before)
    );
    let recipe = sketches.recipe();
    assert_eq!(recipe.version, 9);
    let restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(sketches.same_recipe(&restored));
    let mut legacy = recipe;
    legacy.version = 8;
    assert!(DocumentSketches::from_recipe(&legacy).is_err());
}

#[test]
fn invalid_capture_does_not_create_a_definition() {
    let mut sketches = document();
    let before = sketches.clone();
    for ids in [vec![], vec![SketchEntityId(999)]] {
        assert!(
            sketches
                .capture_block_definition("Rib", DocumentSketchId(1), &ids)
                .is_err()
        );
        assert!(sketches.same_recipe(&before));
    }
}

#[test]
fn captured_point_preserves_signed_zero_and_exact_recipe_comparison() {
    let mut sketches = document();
    sketches
        .capture_block_definition("Point", DocumentSketchId(1), &[SketchEntityId(1)])
        .unwrap();
    let mut recipe = sketches.recipe();
    recipe.block_definitions[0]
        .geometry
        .as_mut()
        .unwrap()
        .points[0]
        .position[0] = -0.0;
    let other = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(!sketches.same_recipe(&other));
    let geometry = other.block_definitions()[0].geometry();
    assert_eq!(
        geometry.point(SketchEntityId(1)).unwrap().position[0].to_bits(),
        (-0.0f64).to_bits()
    );
}

#[test]
fn captured_center_point_keeps_its_role() {
    let mut sketches = document();
    let mut geometry = sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry.set_center_point(SketchEntityId(1), true).unwrap();
    sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    sketches
        .capture_block_definition("Point", DocumentSketchId(1), &[SketchEntityId(1)])
        .unwrap();
    assert!(
        sketches.block_definitions()[0]
            .geometry()
            .is_center_point(SketchEntityId(1))
    );
}
