// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod snapshots;
#[path = "assets/support.rs"]
mod support;
use oblikovati_api::types::{ASSET_BUILTIN, ASSET_DOCUMENT, ASSET_PROJECT};
use oblikovati_model::{
    assets::{AppearanceAsset, AssignmentReference, DocumentAssets, MaterialAsset},
    history::DocumentEditHistory,
    identity::RecipeIdAllocator,
};
#[test]
fn only_nonbuiltin_definitions_are_embedded_as_independent_document_snapshots() {
    let mut assets = DocumentAssets::default();
    let builtin = MaterialAsset::from_info(support::material_record(ASSET_BUILTIN)).unwrap();
    assert!(!assets.embed_material(&builtin).unwrap());
    let builtin_look =
        AppearanceAsset::from_info(support::appearance_record(ASSET_BUILTIN)).unwrap();
    assert!(!assets.embed_appearance(&builtin_look).unwrap());
    assert!(assets.is_empty());
    let mut project = builtin
        .duplicate("physical", "Project", ASSET_PROJECT)
        .unwrap();
    assets.embed_material(&project).unwrap();
    assets
        .embed_appearance(
            &builtin_look
                .duplicate("visual", "Project look", ASSET_PROJECT)
                .unwrap(),
        )
        .unwrap();
    let mut edit = project.info();
    edit.density = 1.;
    project.edit(edit).unwrap();
    assert_eq!(
        assets.materials().get("physical").unwrap().info().density,
        7.85
    );
    assert_eq!(
        assets.materials().get("physical").unwrap().info().source,
        ASSET_DOCUMENT.0
    );
    assert_eq!(assets.appearances().ordered().count(), 1);
}
#[test]
fn assignment_changes_enter_undo_redo_and_are_not_discarded_as_unchanged() {
    let mut history =
        DocumentEditHistory::new(snapshots::empty_snapshot(), RecipeIdAllocator::default())
            .unwrap();
    let token = history.begin("Material").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.assets.assignments.set_part_material("steel")
        })
        .unwrap();
    assert!(!history.discard_unchanged(&token).unwrap());
    history.commit(&token, false).unwrap();
    assert_eq!(
        history.snapshot().assets.assignments.part_material_id(),
        Some("steel")
    );
    history.undo().unwrap();
    assert!(history.snapshot().assets.is_empty());
    history.redo().unwrap();
    assert_eq!(
        history.snapshot().assets.assignments.part_material_id(),
        Some("steel")
    );
}
#[test]
fn invalid_asset_edit_and_aborted_edit_leave_the_committed_snapshot_unchanged() {
    let mut history =
        DocumentEditHistory::new(snapshots::empty_snapshot(), RecipeIdAllocator::default())
            .unwrap();
    let token = history.begin("Material").unwrap();
    assert!(
        history
            .edit(&token, |snapshot, _| snapshot
                .assets
                .assignments
                .set_part_material("bad\0id"))
            .is_err()
    );
    assert!(history.snapshot().assets.is_empty());
    history
        .edit(&token, |snapshot, _| {
            snapshot
                .assets
                .assignments
                .set_body_material(&AssignmentReference::new("body", vec![]).unwrap(), "steel")
        })
        .unwrap();
    history.abort(&token).unwrap();
    assert!(history.snapshot().assets.is_empty());
}

#[test]
fn refresh_metadata_comparison_publishes_asset_changes_once() {
    let mut history =
        DocumentEditHistory::new(snapshots::empty_snapshot(), RecipeIdAllocator::default())
            .unwrap();
    assert!(
        history
            .refresh(|snapshot| snapshot.assets.assignments.set_part_material("steel"))
            .unwrap()
    );
    assert!(
        !history
            .refresh(|snapshot| snapshot.assets.assignments.set_part_material("steel"))
            .unwrap()
    );
    assert_eq!(
        history.snapshot().assets.assignments.part_material_id(),
        Some("steel")
    );
}
