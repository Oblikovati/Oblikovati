// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{
    native_sketch_entities, verify_copy_readiness_and_undo, verify_native_entity_reference,
};
use super::*;
use oblikovati_model::{history::SketchPlacementFieldLock, sketch::SketchPlacementRecipe};
#[test]
fn native_owner_commits_constrained_slot_recipes_and_reopens_their_locked_fields() {
    for curved in [false, true] {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("placement.opd");
        let mut window = NativeHeadWindow::open().unwrap();
        let mut owner = native_owner(window.wake.clone()).unwrap();
        let document = open_native_part(&mut owner, &path);
        let original = publish_native_scene(&mut owner, &mut window);
        assert!(solid_pixels(&original) > 0);
        verify_native_placement(&mut owner, document, curved);
        verify_copy_readiness_and_undo(
            &mut owner,
            &mut window,
            &original,
            if curved { 4 } else { 3 },
        );
        Client::new(owner.document_host())
            .transactions()
            .redo()
            .unwrap();
        verify_reopened_placement(&mut owner, document, &path);
        assert_eq!(
            solid_pixels(&publish_native_scene(&mut owner, &mut window)),
            0
        );
        owner.finish().unwrap();
        window.finish().unwrap();
    }
}
fn verify_native_placement(owner: &mut NativeDesktopOwner, document: u64, curved: bool) {
    let recipe = if curved {
        SketchPlacementRecipe::arc_slot([0.0; 2], [[10.0, 0.0], [0.0, 10.0]], 2.0, true)
    } else {
        SketchPlacementRecipe::straight_slot([[0.0; 2], [10.0, 0.0]], 2.0)
    }
    .unwrap();
    let locks = [
        Some(SketchPlacementFieldLock {
            expression: "10 cm".into(),
            text_point: Some([2.0, 3.0]),
        }),
        None,
        Some(SketchPlacementFieldLock {
            expression: "2 cm".into(),
            text_point: Some([4.0, 5.0]),
        }),
    ];
    assert!(
        owner
            .commit_sketch_placement(999, fixture::SKETCH, &recipe, &locks)
            .is_err()
    );
    let mut invalid = recipe.clone();
    invalid.points[0][0] = f64::NAN;
    assert!(
        owner
            .commit_sketch_placement(document, fixture::SKETCH, &invalid, &locks)
            .is_err()
    );
    let report = owner
        .commit_sketch_placement(document, fixture::SKETCH, &recipe, &locks)
        .unwrap();
    assert_eq!(report.dimensions.len(), 2);
    let client = Client::new(owner.document_host());
    let rows = native_sketch_entities(&client);
    for id in report.members {
        verify_native_entity_reference(&client, rows.iter().find(|row| row.id == id.0).unwrap());
    }
    assert_eq!(
        rows.iter().filter(|row| row.construction).count(),
        usize::from(!curved)
    );
}
fn verify_reopened_placement(
    owner: &mut NativeDesktopOwner,
    document: u64,
    path: &std::path::Path,
) {
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .clone();
    let client = Client::new(owner.document_host());
    let rows = native_sketch_entities(&client);
    client
        .documents()
        .save(&SaveDocumentArgs { document })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            ..Default::default()
        })
        .unwrap();
    let reopened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    assert_eq!(native_sketch_entities(&client), rows);
    let after = owner
        .documents()
        .workspace()
        .session(reopened)
        .unwrap()
        .snapshot();
    assert!(before.sketches.same_recipe(&after.sketches));
    assert!(before.parameters.same_recipe(&after.parameters));
}
