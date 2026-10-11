// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{document_sketch::SketchReadinessCause, sketch::SketchDeletionSelection};
fn selected_scene(host: &Host, document: u64, scene: &DeletionScene) -> SketchDeletionSelection {
    let geometry = owner_snapshot(host, document)
        .sketches
        .get(scene.sketch)
        .unwrap()
        .geometry();
    SketchDeletionSelection {
        entities: vec![scene.circles[0].entity],
        dimensions: scene
            .dimensions
            .iter()
            .map(|dimension| dimension.constraint)
            .collect(),
        geometric_constraints: geometry.geometric_constraints().map(|row| row.id).collect(),
    }
}
fn delete_selected_scene(host: &mut Host, document: u64, scene: &DeletionScene) {
    let selection = selected_scene(host, document, scene);
    let report = host
        .commands(document)
        .unwrap()
        .delete_sketch_selection(scene.sketch, &selection)
        .unwrap();
    assert_eq!(report.entities, vec![scene.circles[0].entity]);
    assert_eq!(report.constraints.len(), 4);
}
fn assert_unconstrained_survivors(host: &Host, document: u64, scene: &DeletionScene) {
    let snapshot = owner_snapshot(host, document);
    let placed = snapshot.sketches.get(scene.sketch).unwrap();
    assert_eq!(
        placed.geometry().entity_ids(),
        &[scene.circles[1].entity, scene.marker]
    );
    assert!(placed.geometry().constraints().is_empty());
    for dimension in &scene.dimensions {
        assert_retired_source(snapshot, dimension);
    }
    let decline = placed
        .require_fully_constrained(&snapshot.parameters)
        .unwrap_err();
    assert_eq!(decline.cause, SketchReadinessCause::FreeVariables(5));
}
#[test]
fn mixed_file_selection_saves_and_copies_local_underconstraint_health_for_both_owners() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_mixed_file_selection(kind);
    }
}
fn verify_mixed_file_selection(kind: DocumentType) {
    let (mut host, mut source, document, scene) = deletion_owner(kind);
    delete_selected_scene(&mut host, document, &scene);
    assert_unconstrained_survivors(&host, document, &scene);
    let expected = owner_snapshot(&host, document).sketches.clone();
    let reopened = save_reopen_owner(&mut host, document, &mut source);
    assert_unconstrained_survivors(&host, reopened, &scene);
    assert!(expected.same_recipe(&owner_snapshot(&host, reopened).sketches));
    let copy = save_open_copy(&mut host, reopened, &mut source);
    assert_unconstrained_survivors(&host, copy, &scene);
    assert!(expected.same_recipe(&owner_snapshot(&host, copy).sketches));
    assert_copy_names(&host, reopened, copy, &scene);
}
#[test]
fn borrowed_mixed_reply_failure_preserves_the_clean_saved_owner() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_borrowed_failure(kind);
    }
}
fn verify_borrowed_failure(kind: DocumentType) {
    let (mut host, _, document, scene) = deletion_owner(kind);
    let before = owner_snapshot(&host, document).sketches.clone();
    let selection = selected_scene(&host, document, &scene);
    let result = host
        .commands(document)
        .unwrap()
        .delete_sketch_selection_projected(scene.sketch, &selection, |reply| {
            assert_eq!(reply.unwrap().constraints.len(), 4);
            Err::<(), _>("reply refused")
        });
    assert!(result.is_err());
    assert!(before.same_recipe(&owner_snapshot(&host, document).sketches));
    assert_restored_source(&host, document, &scene);
    assert!(!host.workspace().needs_save(document).unwrap());
}
