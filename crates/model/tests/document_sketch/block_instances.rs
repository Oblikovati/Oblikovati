// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::SketchBlockInstance;

fn instance(id: u64, definition: &str) -> SketchBlockInstance {
    SketchBlockInstance::new(SketchEntityId(id), definition, [1., -2.], 0.5, 2.).unwrap()
}

#[test]
fn placement_coordinates_round_trip_for_rotation_and_signed_scale() {
    for scale in [2., -2.] {
        let block =
            SketchBlockInstance::new(SketchEntityId(20), "Rib", [3., -4.], 0.7, scale).unwrap();
        let point = [2., -1.];
        let restored = block
            .to_definition(block.to_sketch(point).unwrap())
            .unwrap();
        for axis in 0..2 {
            assert!((restored[axis] - point[axis]).abs() < 1e-12);
        }
        assert!(block.to_sketch([f64::MAX, f64::MAX]).is_err());
        assert!(block.to_definition([f64::INFINITY, 0.]).is_err());
    }
}
#[test]
fn instances_round_trip_with_global_identities_and_block_deletion_is_refused() {
    let mut sketches = document();
    sketches.define_empty_block("Rib").unwrap();
    let before = sketches.clone();
    sketches
        .insert_block_instance(DocumentSketchId(1), instance(20, "Rib"))
        .unwrap();
    assert!(!before.same_recipe(&sketches));
    assert!(sketches.entity_ids().contains(&20));
    let error = sketches
        .delete_block_definition("Rib")
        .unwrap_err()
        .to_string();
    assert!(error.contains("Sketch1"), "{error}");
    let recipe = sketches.recipe();
    assert_eq!(recipe.version, 10);
    let restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&sketches));
    let guid: DocumentGuid = "00112233-4455-6677-8899-aabbccddeeff".parse().unwrap();
    let names =
        oblikovati_model::document_sketch::DocumentSketchReferenceIndex::new(&restored, guid)
            .unwrap();
    let key = names
        .entity_key(DocumentSketchId(1), SketchEntityId(20))
        .unwrap();
    assert_eq!(
        names.resolve(&key).unwrap().entity,
        Some(SketchEntityId(20))
    );
    assert_eq!(
        restored.get(DocumentSketchId(1)).unwrap().block_instances(),
        &[instance(20, "Rib")]
    );
    sketches.remove(DocumentSketchId(1)).unwrap();
    sketches.delete_block_definition("Rib").unwrap();
}
#[test]
fn invalid_or_colliding_instances_never_change_the_document() {
    let mut sketches = document();
    sketches.define_empty_block("Rib").unwrap();
    let before = sketches.clone();
    for (sketch, item) in [
        (DocumentSketchId(1), instance(1, "Rib")),
        (DocumentSketchId(1), instance(20, "Missing")),
        (DocumentSketchId(99), instance(20, "Rib")),
    ] {
        assert!(sketches.insert_block_instance(sketch, item).is_err());
        assert!(before.same_recipe(&sketches));
    }
    for (id, position, rotation, scale) in [
        (0, [0., 0.], 0., 1.),
        (20, [f64::NAN, 0.], 0., 1.),
        (20, [0., 0.], f64::INFINITY, 1.),
        (20, [0., 0.], 0., 0.),
    ] {
        assert!(
            SketchBlockInstance::new(SketchEntityId(id), "Rib", position, rotation, scale).is_err()
        );
    }
}
#[test]
fn old_or_malformed_instance_recipes_are_refused() {
    let mut sketches = document();
    sketches.define_empty_block("Rib").unwrap();
    sketches
        .insert_block_instance(DocumentSketchId(1), instance(20, "Rib"))
        .unwrap();
    let mut recipe = sketches.recipe();
    recipe.version = 9;
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
    recipe.version = 10;
    recipe.sketches[1].block_instances.push(instance(20, "Rib"));
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
}

#[test]
fn replacing_geometry_cannot_reuse_an_instance_identity() {
    let mut sketches = document();
    sketches.define_empty_block("Rib").unwrap();
    sketches
        .insert_block_instance(DocumentSketchId(1), instance(20, "Rib"))
        .unwrap();
    let before = sketches.clone();
    let mut geometry = sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry.add_point(SketchEntityId(20), [0., 0.]).unwrap();
    assert!(
        sketches
            .replace_geometry(DocumentSketchId(1), geometry)
            .is_err()
    );
    assert!(before.same_recipe(&sketches));
}
#[test]
fn instance_placement_preserves_signed_zero_in_recipe_comparison() {
    let mut first = document();
    first.define_empty_block("Rib").unwrap();
    let mut second = first.clone();
    for (sketches, x) in [(&mut first, 0.0), (&mut second, -0.0)] {
        let placed = SketchBlockInstance::new(SketchEntityId(20), "Rib", [x, 0.], 0., 1.).unwrap();
        sketches
            .insert_block_instance(DocumentSketchId(1), placed)
            .unwrap();
    }
    assert!(!first.same_recipe(&second));
    let restored = DocumentSketches::from_recipe(&second.recipe()).unwrap();
    assert!(second.same_recipe(&restored));
}
