// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::{
        DocumentSketchId, DocumentSketchReferenceIndex, DocumentSketches, SketchReferenceTarget,
    },
    identity::DocumentGuid,
    parameter::ParameterCollection,
    sketch::{Sketch2D, SketchEntityId, SketchPlane, SketchRecipe},
};
use oblikovati_solve::nonlinear::SolveOptions;
fn mixed_geometry() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(9), [0.0; 2]).unwrap();
    sketch.add_point(SketchEntityId(3), [2.0, 0.0]).unwrap();
    sketch
        .add_line(SketchEntityId(11), [SketchEntityId(9), SketchEntityId(3)])
        .unwrap();
    sketch
        .add_circle(SketchEntityId(7), SketchEntityId(9), 2.0)
        .unwrap();
    sketch.add_point(SketchEntityId(1), [0.0, 2.0]).unwrap();
    add_quarter_arc(&mut sketch, 1).unwrap();
    sketch
}
fn add_quarter_arc(
    sketch: &mut Sketch2D,
    end: u64,
) -> Result<(), oblikovati_model::sketch::SketchError> {
    sketch.add_arc(
        SketchEntityId(15),
        SketchEntityId(9),
        [SketchEntityId(3), SketchEntityId(end)],
        true,
    )
}
fn identifiers(geometry: &Sketch2D) -> Vec<u64> {
    geometry
        .entity_ids()
        .iter()
        .map(|entity| entity.0)
        .collect()
}
#[test]
fn mixed_order_is_not_sorted_by_kind_or_identity_and_survives_version_two_restore() {
    let sketch = mixed_geometry();
    assert_eq!(identifiers(&sketch), [9, 3, 11, 7, 1, 15]);
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 2);
    assert_eq!(recipe.entity_order.as_ref().unwrap(), sketch.entity_ids());
    let encoded = serde_json::to_vec(&recipe).unwrap();
    let restored: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    let restored = Sketch2D::from_recipe(&restored).unwrap();
    assert!(sketch.same_recipe(&restored));
    assert_eq!(identifiers(&restored), [9, 3, 11, 7, 1, 15]);
}
#[test]
fn legacy_version_one_restores_the_original_grouped_order_without_new_content() {
    let mut recipe = mixed_geometry().recipe();
    recipe.version = 1;
    recipe.entity_order = None;
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert_eq!(identifiers(&restored), [9, 3, 1, 11, 7, 15]);
    assert_eq!(restored.recipe().version, 1);
    assert!(
        !String::from_utf8(serde_json::to_vec(&restored.recipe()).unwrap())
            .unwrap()
            .contains("entity_order")
    );
}
#[test]
fn order_is_authored_content_and_not_recovered_from_numerical_coordinates() {
    let first = mixed_geometry();
    let mut recipe = first.recipe();
    recipe.entity_order.as_mut().unwrap().reverse();
    let second = Sketch2D::from_recipe(&recipe).unwrap();
    assert_ne!(first, second);
    assert!(!first.same_recipe(&second));
    assert_eq!(first.points(), second.points());
    assert_eq!(first.lines(), second.lines());
}
#[test]
fn malformed_permutations_are_declined_before_restore() {
    let original = mixed_geometry().recipe();
    for order in [
        vec![],
        vec![SketchEntityId(9); 6],
        vec![SketchEntityId(99); 6],
        vec![SketchEntityId(9)],
    ] {
        let mut recipe = original.clone();
        recipe.entity_order = Some(order);
        assert!(
            Sketch2D::from_recipe(&recipe)
                .unwrap_err()
                .to_string()
                .contains("exactly once")
        );
    }
}
#[test]
fn cross_version_order_shapes_are_declined_before_restore() {
    let original = mixed_geometry().recipe();
    for version in [0, 1, 3] {
        let mut recipe = original.clone();
        recipe.version = version;
        assert!(Sketch2D::from_recipe(&recipe).is_err());
    }
    let mut missing = original;
    missing.entity_order = None;
    assert!(
        Sketch2D::from_recipe(&missing)
            .unwrap_err()
            .to_string()
            .contains("version 2")
    );
}
#[test]
fn failed_additions_do_not_append_missing_or_duplicate_entities_to_the_order() {
    let mut sketch = mixed_geometry();
    let before = sketch.clone();
    assert!(sketch.add_point(SketchEntityId(9), [1.0; 2]).is_err());
    assert!(sketch.add_point(SketchEntityId(99), [f64::NAN; 2]).is_err());
    assert!(
        sketch
            .add_line(SketchEntityId(99), [SketchEntityId(9), SketchEntityId(88)])
            .is_err()
    );
    assert!(
        sketch
            .add_circle(SketchEntityId(99), SketchEntityId(9), 0.0)
            .is_err()
    );
    assert!(add_quarter_arc(&mut sketch, 88).is_err());
    assert_eq!(sketch, before);
}
#[test]
fn solve_candidates_keep_the_same_cross_kind_identity_order() {
    let sketch = mixed_geometry();
    let parameters = ParameterCollection::new(Default::default());
    let outcome = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(outcome.candidate().entity_ids(), sketch.entity_ids());
    assert!(sketch.same_recipe(outcome.candidate()));
}
fn collection() -> DocumentSketches {
    let mut sketches = DocumentSketches::default();
    sketches
        .add(
            DocumentSketchId(42),
            "Mixed",
            SketchPlane::xy(),
            mixed_geometry(),
        )
        .unwrap();
    sketches
        .add(
            DocumentSketchId(12),
            "Empty",
            SketchPlane::xz(),
            Sketch2D::default(),
        )
        .unwrap();
    sketches
}
fn guid() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}
#[test]
fn the_current_reference_catalogue_round_trips_every_kind_in_its_own_namespace() {
    let sketches = collection();
    let names = DocumentSketchReferenceIndex::new(&sketches, guid()).unwrap();
    for sketch in sketches.sketches() {
        let key = names.sketch_key(sketch.id()).unwrap();
        assert_eq!(
            names.resolve(&key),
            Some(SketchReferenceTarget {
                sketch: sketch.id(),
                entity: None
            })
        );
    }
}
#[test]
fn current_entity_names_round_trip_in_a_namespace_distinct_from_sketches() {
    let names = DocumentSketchReferenceIndex::new(&collection(), guid()).unwrap();
    for entity in mixed_geometry().entity_ids() {
        let key = names.entity_key(DocumentSketchId(42), *entity).unwrap();
        assert_eq!(key, guid().sketch_entity_key(entity.0).to_string());
        assert_eq!(
            names.resolve(&key),
            Some(SketchReferenceTarget {
                sketch: DocumentSketchId(42),
                entity: Some(*entity)
            })
        );
        assert_ne!(key, guid().sketch_key(entity.0).to_string());
    }
}
#[test]
fn keys_for_missing_or_foreign_owner_targets_are_refused_and_unknown_names_miss() {
    let names = DocumentSketchReferenceIndex::new(&collection(), guid()).unwrap();
    assert!(
        names
            .sketch_key(DocumentSketchId(99))
            .unwrap_err()
            .to_string()
            .contains("DocumentSketchId(99)")
    );
    assert!(
        names
            .entity_key(DocumentSketchId(12), SketchEntityId(9))
            .is_err()
    );
    assert!(
        names
            .entity_key(DocumentSketchId(42), SketchEntityId(99))
            .is_err()
    );
}
#[test]
fn foreign_and_unknown_names_do_not_resolve_in_the_current_owner() {
    let names = DocumentSketchReferenceIndex::new(&collection(), guid()).unwrap();
    for key in ["", "invalid", "11111111-2222-3333-4444-555555555555"] {
        assert_eq!(names.resolve(key), None);
    }
    let foreign: DocumentGuid = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap();
    assert_eq!(names.resolve(&foreign.sketch_key(42).to_string()), None);
    assert_eq!(collection().entity_ids(), [9, 3, 11, 7, 1, 15]);
}

#[test]
fn catalogue_sketch_keys_match_the_saved_go_baseline() {
    let names = DocumentSketchReferenceIndex::new(&collection(), guid()).unwrap();
    // Saved from Go aa8297d1; see fixtures/go-sketch-identity.tsv.
    let key = "d30e74d4-4665-5cf7-b36d-7b983fb8dc8c";
    assert_eq!(names.sketch_key(DocumentSketchId(42)).unwrap(), key);
    assert_eq!(
        names.resolve(key),
        Some(SketchReferenceTarget {
            sketch: DocumentSketchId(42),
            entity: None,
        })
    );
}

#[path = "sketch_entity_order/construction.rs"]
mod construction;

#[path = "sketch_entity_order/translation.rs"]
mod translation;
