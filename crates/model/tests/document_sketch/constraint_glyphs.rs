// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn hidden_marks_round_trip_and_show_all_clears_overrides() {
    let mut sketches = document();
    let id = DocumentSketchId(1);
    let relation = SketchConstraintId(3);
    sketches.set_constraints_shown(id, true).unwrap();
    let shown = sketches.clone();
    sketches
        .set_constraint_glyph_visible(id, relation, false)
        .unwrap();
    assert!(
        sketches
            .get(id)
            .unwrap()
            .hidden_constraint_glyphs()
            .contains(&relation)
    );
    assert_ne!(sketches, shown);
    let recipe = sketches.recipe();
    assert_eq!(recipe.version, 17);
    let mut restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&sketches));
    restored.set_constraints_shown(id, true).unwrap();
    assert!(
        restored
            .get(id)
            .unwrap()
            .hidden_constraint_glyphs()
            .is_empty()
    );
    assert_eq!(
        restored.get(id).unwrap().geometry(),
        sketches.get(id).unwrap().geometry()
    );
}

#[test]
fn hidden_marks_reject_invalid_relations_and_invalid_recipes() {
    let mut sketches = document();
    let id = DocumentSketchId(1);
    let unchanged = sketches.clone();
    for invalid in [SketchConstraintId(4), SketchConstraintId(999)] {
        assert!(
            sketches
                .set_constraint_glyph_visible(id, invalid, false)
                .is_err()
        );
        assert!(sketches.same_recipe(&unchanged));
    }
    sketches
        .set_constraint_glyph_visible(id, SketchConstraintId(3), false)
        .unwrap();
    let mut recipe = sketches.recipe();
    recipe.version = 16;
    assert!(
        DocumentSketches::from_recipe(&recipe)
            .unwrap_err()
            .to_string()
            .contains("version 17")
    );
    recipe.version = 17;
    recipe.sketches[0]
        .hidden_constraint_glyphs
        .push(SketchConstraintId(3));
    assert!(
        DocumentSketches::from_recipe(&recipe)
            .unwrap_err()
            .to_string()
            .contains("duplicate")
    );
    recipe.sketches[0].hidden_constraint_glyphs = vec![SketchConstraintId(999)];
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
}

#[test]
fn replacing_geometry_drops_only_deleted_glyph_ids() {
    let mut sketches = document();
    let id = DocumentSketchId(1);
    sketches
        .set_constraint_glyph_visible(id, SketchConstraintId(3), false)
        .unwrap();
    let mut geometry = sketches.get(id).unwrap().geometry().clone();
    geometry
        .remove_geometric_constraint(SketchConstraintId(3))
        .unwrap();
    sketches.replace_geometry(id, geometry).unwrap();
    assert!(
        sketches
            .get(id)
            .unwrap()
            .hidden_constraint_glyphs()
            .is_empty()
    );
    assert!(DocumentSketches::from_recipe(&sketches.recipe()).is_ok());
}

#[test]
fn scalar_contract_reports_global_and_individual_glyph_visibility() {
    use oblikovati_api::contract::ConstraintGlyphVisibility;
    fn accepts_contract<T: ConstraintGlyphVisibility>() {}
    accepts_contract::<oblikovati_model::document_sketch::PlacedSketch>();
    let mut sketches = document();
    let id = DocumentSketchId(1);
    assert!(
        !sketches
            .get(id)
            .unwrap()
            .constraint_glyph_visible(3)
            .unwrap()
    );
    sketches.set_constraints_shown(id, true).unwrap();
    assert!(
        sketches
            .get(id)
            .unwrap()
            .constraint_glyph_visible(3)
            .unwrap()
    );
    sketches
        .set_constraint_glyph_visible(id, SketchConstraintId(3), false)
        .unwrap();
    assert!(
        !sketches
            .get(id)
            .unwrap()
            .constraint_glyph_visible(3)
            .unwrap()
    );
    assert!(
        sketches
            .get(id)
            .unwrap()
            .constraint_glyph_visible(4)
            .is_err()
    );
}
