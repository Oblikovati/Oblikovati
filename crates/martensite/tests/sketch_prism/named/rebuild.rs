// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    part::PartFeatureDefinition, sketch_prism::SketchPrismDefinition, work_geometry::DatumLength,
};
use oblikovati_persistence::package::DocumentPackage;
fn holed_document(rotation: usize) -> DocumentSnapshot {
    let mut snapshot = selected_prism_document();
    add_fixed_sketch_hole(&mut snapshot);
    let mut selection = selected_sketch_boundary();
    selection.outer.rotate_left(rotation);
    selection.holes.push(selected_sketch_line_ring(111));
    let definition =
        SketchPrismDefinition::new(SKETCH, selection, [1., 2., -3.].map(DatumLength::Constant))
            .unwrap();
    let reads = definition.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .replace(
            FeatureId(1),
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        )
        .unwrap();
    solve_selected_sketches(&mut snapshot);
    snapshot
}
#[test]
fn hole_and_oblique_negative_sweep_names_ignore_traversal_order() {
    let first = owner(holed_document(0));
    let second = owner(holed_document(2));
    let left = named(&first);
    let right = named(&second);
    let keys = left.reference_keys(&first, document()).unwrap();
    assert_eq!(keys, right.reference_keys(&second, document()).unwrap());
    assert_eq!(keys.len(), 11);
    for key in keys
        .into_iter()
        .filter(|key| key != &super::key("body", &[1]))
    {
        let a = left.face_report(&first, document(), &key).unwrap();
        let b = right.face_report(&second, document(), &key).unwrap();
        assert_eq!(a.role, b.role);
        assert_volume(a.area, b.area);
        assert_eq!(a.loops, b.loops);
    }
    for role in ["start", "end"] {
        let face = left
            .face_report(&first, document(), &key(role, &[1]))
            .unwrap();
        assert_volume(face.area, 10.);
        assert_eq!(face.loops, 2);
        let expected_z = if role == "start" { 0. } else { -3. };
        assert_centroid([0., 0., face.centroid[2]], [0., 0., expected_z]);
    }
}
fn package(snapshot: &DocumentSnapshot) -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, snapshot.parameters.units().clone())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
}
#[test]
fn saved_recipes_rebuild_the_same_keys_under_a_new_history_owner() {
    let original = owner(selected_prism_document());
    let before = named(&original);
    let keys = before.reference_keys(&original, document()).unwrap();
    let bytes = package(original.snapshot()).encode().unwrap();
    let reopened = DocumentPackage::decode(&bytes).unwrap();
    let mut snapshot = DocumentSnapshot {
        parameters: reopened.restore_document_parameters().unwrap().parameters,
        sketches: reopened.restore_sketches().unwrap().sketches,
        work_geometry: reopened.restore_work_geometry().unwrap().program,
        assets: Default::default(),
        body_names: Default::default(),
        sketch_settings: None,
        properties: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
    };
    solve_selected_sketches(&mut snapshot);
    let restored = owner(snapshot);
    let after = named(&restored);
    assert_eq!(keys, after.reference_keys(&restored, document()).unwrap());
    assert_faces(&after, &restored, 2.);
    assert!(before.reference_keys(&restored, document()).is_err());
}
