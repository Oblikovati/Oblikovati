// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketchRecipe, DocumentSketches},
    identity::{DocumentGuid, RecipeIdAllocator},
    parameter::ParameterCollection,
    sketch::{
        LineContactSide, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchPlane, SketchPlaneOptions,
    },
    units::UnitsOfMeasure,
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
fn geometry(offset: u64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    let id = |value| SketchEntityId(offset + value);
    for (value, xy) in [(1, [0.0, 0.0]), (2, [4.0, 0.0]), (3, [0.0, 0.0])] {
        sketch.add_point(id(value), xy).unwrap();
    }
    sketch.add_line(id(4), [id(1), id(2)]).unwrap();
    for value in [1, 2] {
        sketch
            .add_constraint(
                SketchConstraintId(value),
                PlanarConstraint::Fix {
                    point: id(value),
                    position: if value == 1 { [0.0, 0.0] } else { [4.0, 0.0] },
                },
            )
            .unwrap();
    }
    sketch
        .add_constraint(
            SketchConstraintId(3),
            PlanarConstraint::Vertical {
                points: [id(1), id(3)],
            },
        )
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(4),
            PlanarConstraint::Offset {
                point: id(3),
                line: id(4),
                parameter: 1,
                side: LineContactSide::Left,
                linear_diameter: false,
            },
        )
        .unwrap();
    sketch
}
fn document() -> DocumentSketches {
    let mut sketches = DocumentSketches::default();
    sketches
        .add(
            DocumentSketchId(1),
            "Sketch1",
            SketchPlane::xy(),
            geometry(0),
        )
        .unwrap();
    sketches
        .add(
            DocumentSketchId(2),
            "Sketch2",
            SketchPlane::xz(),
            geometry(10),
        )
        .unwrap();
    sketches
}
#[test]
fn document_recipes_preserve_order_names_frames_global_entities_and_local_constraints() {
    let original = document();
    let encoded = serde_json::to_string(&original.recipe()).unwrap();
    let recipe: DocumentSketchRecipe = serde_json::from_str(&encoded).unwrap();
    let restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert_eq!(restored, original);
    assert_eq!(restored.sketch_ids(), [1, 2]);
    assert_eq!(restored.entity_ids(), [1, 2, 3, 4, 11, 12, 13, 14]);
    assert_eq!(restored.get(DocumentSketchId(2)).unwrap().name(), "Sketch2");
    assert_eq!(
        restored.get(DocumentSketchId(2)).unwrap().plane(),
        SketchPlane::xz()
    );
    assert_eq!(
        restored
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry()
            .constraints()[0]
            .id,
        restored
            .get(DocumentSketchId(2))
            .unwrap()
            .geometry()
            .constraints()[0]
            .id
    );
    assert_eq!(serde_json::to_string(&restored.recipe()).unwrap(), encoded);
}
#[test]
fn entity_collisions_and_unknown_sketch_edits_leave_the_whole_document_unchanged() {
    let mut sketches = document();
    let original = sketches.clone();
    for id in [0, 1, 99] {
        assert!(
            sketches
                .add(
                    DocumentSketchId(id),
                    "Collision",
                    SketchPlane::yz(),
                    geometry(0)
                )
                .is_err()
        );
        assert_eq!(sketches, original);
    }
    assert!(
        sketches
            .replace_geometry(DocumentSketchId(2), geometry(0))
            .unwrap_err()
            .to_string()
            .contains("one owner")
    );
    assert_eq!(sketches, original);
    let missing = DocumentSketchId(999);
    assert!(sketches.get(missing).is_err());
    assert!(sketches.remove(missing).is_err());
    assert!(sketches.rename(missing, "Missing").is_err());
    assert!(sketches.replace_geometry(missing, geometry(20)).is_err());
    assert!(sketches.replace_plane(missing, SketchPlane::yz()).is_err());
    assert_eq!(sketches, original);
}
#[test]
fn named_geometry_and_placement_edits_preserve_keys_and_other_sketch_records() {
    let mut sketches = document();
    let other = sketches.get(DocumentSketchId(2)).unwrap().clone();
    sketches.rename(DocumentSketchId(1), "Sketch2").unwrap(); // Go permits explicitly duplicated names.
    sketches
        .replace_plane(DocumentSketchId(1), SketchPlane::yz())
        .unwrap();
    let mut updated = geometry(0);
    updated.set_position(SketchEntityId(3), [0.0, 7.0]).unwrap();
    sketches
        .replace_geometry(DocumentSketchId(1), updated)
        .unwrap();
    assert_eq!(
        sketches.get(DocumentSketchId(1)).unwrap().id(),
        DocumentSketchId(1)
    );
    assert_eq!(
        sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry()
            .point(SketchEntityId(3))
            .unwrap()
            .position,
        [0.0, 7.0]
    );
    assert_eq!(sketches.get(DocumentSketchId(2)).unwrap(), &other);
    assert_eq!(
        sketches.remove(DocumentSketchId(1)).unwrap().name(),
        "Sketch2"
    );
    assert_eq!(sketches.sketch_ids(), [2]);
    sketches
        .add(DocumentSketchId(3), "", SketchPlane::xy(), geometry(20))
        .unwrap();
    assert_eq!(sketches.sketch_ids(), [2, 3]);
}
#[test]
fn restored_sketch_solves_with_zero_dof_and_keeps_document_placement() {
    let mut sketches = DocumentSketches::from_recipe(&document().recipe()).unwrap();
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "5 cm")
        .unwrap();
    for width in [5.0, 0.0, 8.0] {
        parameters
            .set_expression(1, &format!("{width} cm"))
            .unwrap();
        let placed = sketches.get(DocumentSketchId(2)).unwrap();
        let result = placed
            .geometry()
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        sketches
            .replace_geometry(DocumentSketchId(2), result.into_candidate())
            .unwrap();
        let placed = sketches.get(DocumentSketchId(2)).unwrap();
        let xy = placed
            .geometry()
            .point(SketchEntityId(13))
            .unwrap()
            .position;
        assert!((xy[1] - width).abs() < 1e-8);
        let model = placed.plane().to_model(xy).unwrap();
        assert!((model[2] - width).abs() < 1e-8);
        assert_eq!(sketches.sketch_ids(), [1, 2]);
    }
}
#[test]
fn restoring_refuses_invalid_collection_version_global_ids_geometry_and_frame_options() {
    let original = document().recipe();
    let mut candidates = Vec::new();
    let mut recipe = original.clone();
    recipe.version = 99;
    candidates.push(recipe);
    let mut recipe = original.clone();
    recipe.sketches[1].id = DocumentSketchId(1);
    candidates.push(recipe);
    let mut recipe = original.clone();
    recipe.sketches[1].geometry = recipe.sketches[0].geometry.clone();
    candidates.push(recipe);
    let mut recipe = original.clone();
    recipe.sketches[1].geometry.points[0].position[0] = f64::NAN;
    candidates.push(recipe);
    let mut recipe = original.clone();
    recipe.sketches[1].plane_options.version = 99;
    candidates.push(recipe);
    for recipe in candidates {
        assert!(DocumentSketches::from_recipe(&recipe).is_err());
    }
    let mut json = serde_json::to_value(&original).unwrap();
    json["sketches"][0]["unsupported"] = true.into();
    assert!(serde_json::from_value::<DocumentSketchRecipe>(json).is_err());
    assert_eq!(document().recipe(), original);
}
#[test]
fn custom_frame_options_round_trip_and_identity_allocators_do_not_rewind_on_failure_or_removal() {
    let options = SketchPlaneOptions {
        version: 1,
        unit_deviation: 0.01,
        angular_deviation: 0.01,
    };
    let plane =
        SketchPlane::new([1.0, 2.0, 3.0], [1.001, 0.0, 0.0], [0.0, 1.0, 0.0], options).unwrap();
    let mut sketches = DocumentSketches::default();
    let mut ids = RecipeIdAllocator::default();
    let first = DocumentSketchId(ids.allocate().unwrap());
    sketches.add(first, "Custom", plane, geometry(0)).unwrap();
    let restored = DocumentSketches::from_recipe(&sketches.recipe()).unwrap();
    assert_eq!(restored.get(first).unwrap().plane().options(), options);
    assert_eq!(
        restored.get(first).unwrap().plane().recipe().x_axis,
        [1.001, 0.0, 0.0]
    );
    let refused = DocumentSketchId(ids.allocate().unwrap());
    assert!(
        sketches
            .add(refused, "Collision", plane, geometry(0))
            .is_err()
    );
    sketches.remove(first).unwrap();
    assert_eq!(ids.allocate().unwrap(), 3);
    assert_eq!(ids.high_water(), 3);
}
#[test]
fn saved_document_reference_keys_survive_restore_and_keep_kind_and_document_namespaces() {
    let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let copy: DocumentGuid = "21111111-2222-3333-4444-555555555555".parse().unwrap();
    let before = document();
    let after = DocumentSketches::from_recipe(&before.recipe()).unwrap();
    for id in before.sketch_ids() {
        assert_eq!(
            doc.sketch_key(id),
            doc.sketch_key(after.get(DocumentSketchId(id)).unwrap().id().0)
        );
        assert_ne!(doc.sketch_key(id), copy.sketch_key(id));
    }
    for id in before.entity_ids() {
        assert!(after.entity_ids().contains(&id));
        assert_ne!(doc.sketch_key(id), doc.sketch_entity_key(id));
        assert_ne!(doc.sketch_entity_key(id), copy.sketch_entity_key(id));
    }
}
#[test]
fn exact_recipe_comparison_tracks_signed_zero_geometry_fix_targets_and_collection_shape() {
    let original = document();
    assert!(original.same_recipe(&original.clone()));
    let mut changed = original.clone();
    let mut geometry = geometry(0);
    geometry
        .set_position(SketchEntityId(1), [-0.0, 0.0])
        .unwrap();
    changed
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    assert!(!original.same_recipe(&changed));
    let mut fixed = Sketch2D::default();
    fixed.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    fixed
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point: SketchEntityId(1),
                position: [0.0, 0.0],
            },
        )
        .unwrap();
    let mut recipe = fixed.recipe();
    recipe.constraints[0].relation = PlanarConstraint::Fix {
        point: SketchEntityId(1),
        position: [-0.0, 0.0],
    };
    let other = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(!fixed.same_recipe(&other));
    let mut changed = original.clone();
    changed.remove(DocumentSketchId(2)).unwrap();
    assert!(!original.same_recipe(&changed));
    let mut changed = original.clone();
    changed.rename(DocumentSketchId(1), "Different").unwrap();
    assert!(!original.same_recipe(&changed));
    assert!(!SketchPlane::xy().same_recipe(SketchPlane::xz()));
}

#[path = "document_sketch/recompute.rs"]
mod recompute;

#[path = "document_sketch/health.rs"]
mod health;
