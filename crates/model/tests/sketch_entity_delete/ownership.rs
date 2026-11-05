// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::{DocumentSketchId, DocumentSketches};
use oblikovati_model::{document_sketch::DocumentSketchReferenceIndex, identity::DocumentGuid};
fn hidden_line() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_defining_point(id(100), [0.0, -0.0]).unwrap();
    sketch.add_defining_point(id(200), [2.0, 0.0]).unwrap();
    sketch.add_line(id(10), [id(100), id(200)]).unwrap();
    sketch
}
#[test]
fn invalid_defining_point_adoption_does_not_change_roles_order_or_existing_coordinates() {
    let mut sketch = hidden_line();
    let before = sketch.clone();
    for (value, position) in [
        (0, [0.0, 0.0]),
        (10, [0.0, 0.0]),
        (100, [1.0, 2.0]),
        (300, [f64::NAN, 0.0]),
    ] {
        assert!(sketch.add_defining_point(id(value), position).is_err());
        assert!(sketch.same_recipe(&before));
    }
}
#[test]
fn defining_points_are_solver_variables_without_standalone_entity_rows() {
    let sketch = hidden_line();
    assert_eq!(sketch.entity_ids(), &[id(10)]);
    assert_eq!(sketch.owned_entity_ids(), vec![id(10), id(100), id(200)]);
    let parameters = ParameterCollection::new(Default::default());
    let structure = sketch.analyze(&parameters, 1e-9).unwrap();
    assert_eq!((structure.variables, structure.degrees_of_freedom), (4, 4));
    let rows = sketch
        .inspect_entities(&parameters, &[], Default::default())
        .unwrap();
    assert_eq!(rows.len(), 1);
    assert_eq!(rows[0].id, id(10));
    assert_eq!(rows[0].points, vec![[0.0, -0.0], [2.0, 0.0]]);
}
#[test]
fn defining_point_ids_remain_in_the_document_identity_namespace() {
    let mut document = DocumentSketches::default();
    add_owned_sketch(&mut document, 1, hidden_line()).unwrap();
    assert_eq!(document.entity_ids(), vec![10, 100, 200]);
    let mut duplicate = Sketch2D::default();
    duplicate.add_point(id(200), [9.0, 9.0]).unwrap();
    assert!(add_owned_sketch(&mut document, 2, duplicate).is_err());
    assert_eq!(document.sketch_ids(), vec![1]);
}
fn add_owned_sketch(
    document: &mut DocumentSketches,
    value: u64,
    geometry: Sketch2D,
) -> Result<(), SketchError> {
    document.add(
        DocumentSketchId(value),
        &format!("Owner {value}"),
        SketchPlane::xy(),
        geometry,
    )
}
#[test]
fn schema_twelve_preserves_point_roles_signed_zero_and_visible_order() {
    let sketch = hidden_line();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 12);
    assert_eq!(recipe.defining_points, Some(vec![id(100), id(200)]));
    assert_eq!(recipe.entity_order, Some(vec![id(10)]));
    let encoded = serde_json::to_vec(&recipe).unwrap();
    let decoded: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    let restored = Sketch2D::from_recipe(&decoded).unwrap();
    assert!(sketch.same_recipe(&restored));
    assert_eq!(
        restored.point(id(100)).unwrap().position[1].to_bits(),
        (-0.0_f64).to_bits()
    );
    assert_eq!(serde_json::to_vec(&restored.recipe()).unwrap(), encoded);
}
#[test]
fn older_schemas_cannot_silently_acquire_hidden_point_ownership() {
    let recipe = hidden_line().recipe();
    for version in 1..=11 {
        let mut legacy = recipe.clone();
        legacy.version = version;
        assert_point_schema_error(&legacy, "point ownership recipe");
    }
    let mut missing = recipe;
    missing.defining_points = None;
    assert_point_schema_error(&missing, "explicit list in version 12");
}
fn assert_point_schema_error(recipe: &SketchRecipe, expected: &str) {
    let error = Sketch2D::from_recipe(recipe).unwrap_err();
    assert!(
        error.to_string().contains(expected),
        "{error}: expected {expected}"
    );
}
#[test]
fn defining_point_metadata_refuses_duplicate_missing_curve_and_reordered_ids() {
    for ids in [
        vec![id(100), id(100)],
        vec![id(999)],
        vec![id(10)],
        vec![id(200), id(100)],
    ] {
        let mut recipe = hidden_line().recipe();
        recipe.defining_points = Some(ids);
        assert!(
            Sketch2D::from_recipe(&recipe)
                .unwrap_err()
                .to_string()
                .contains("defining points")
        );
    }
}
#[test]
fn legacy_point_roles_are_not_guessed_from_curve_references() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(id(1), [0.0, 0.0]).unwrap();
    sketch.add_point(id(2), [1.0, 0.0]).unwrap();
    sketch.add_line(id(3), [id(1), id(2)]).unwrap();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 1);
    assert!(recipe.defining_points.is_none());
    let mut restored = Sketch2D::from_recipe(&recipe).unwrap();
    restored.delete_entities(&[id(3)]).unwrap();
    assert_eq!(restored.entity_ids(), &[id(1), id(2)]);
}
#[test]
fn removing_the_last_curve_prunes_hidden_endpoints_and_their_bound_relations() {
    let mut sketch = hidden_line();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Horizontal {
                points: [id(100), id(200)],
            },
        )
        .unwrap();
    let retired = sketch.delete_entities(&[id(10)]).unwrap();
    assert_eq!(retired.entities, vec![id(10), id(100), id(200)]);
    assert_eq!(retired.constraints.len(), 1);
    assert!(sketch.points().is_empty());
    assert!(sketch.entity_ids().is_empty());
    assert_valid_candidate(&sketch);
}
#[test]
fn defining_points_keep_round_trip_reference_keys_until_their_geometry_is_retired() {
    let mut document = DocumentSketches::default();
    add_owned_sketch(&mut document, 1, hidden_line()).unwrap();
    let guid: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let names = DocumentSketchReferenceIndex::new(&document, guid).unwrap();
    let key = names.entity_key(DocumentSketchId(1), id(100)).unwrap();
    assert_eq!(names.resolve(&key).unwrap().entity, Some(id(100)));
    assert_restored_point_key(&document, guid, &key);
    retire_hidden_line(&mut document);
    assert!(
        DocumentSketchReferenceIndex::new(&document, guid)
            .unwrap()
            .resolve(&key)
            .is_none()
    );
}
fn assert_restored_point_key(document: &DocumentSketches, guid: DocumentGuid, key: &str) {
    let restored = DocumentSketches::from_recipe(&document.recipe()).unwrap();
    let names = DocumentSketchReferenceIndex::new(&restored, guid).unwrap();
    assert_eq!(names.entity_key(DocumentSketchId(1), id(100)).unwrap(), key);
}
fn retire_hidden_line(document: &mut DocumentSketches) {
    let mut geometry = hidden_line();
    geometry.delete_entities(&[id(10)]).unwrap();
    document
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
}
