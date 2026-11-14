// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assets::AssignmentReference;
#[test]
fn native_file_owner_create_save_reopen_and_history_retain_assignment_intent() {
    let mut host = Host::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(0);
    let destination = path("assets.opd");
    let package = asset_template()
        .with_assets(&portable_asset_definitions())
        .unwrap();
    let id = host
        .create(
            &destination,
            package,
            RecordingObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    host.commands(id)
        .unwrap()
        .run("Set material", |snapshot, _| {
            snapshot.assets.assignments.set_part_material("steel")
        })
        .unwrap();
    host.save(id, "test", &mut source).unwrap();
    host.close_discarding_changes(id).unwrap();
    let reopened = host
        .open(
            &destination,
            RecordingObserver::default(),
            "test",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    assert_eq!(
        host.workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .assets
            .assignments
            .part_material_id(),
        Some("steel")
    );
    assert_eq!(
        host.workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .assets
            .materials()
            .get("portable")
            .unwrap()
            .info()
            .display_name,
        "Portable steel"
    );
    change_asset_override(&mut host, reopened);
    host.commands(reopened).unwrap().undo().unwrap();
    assert_eq!(
        host.workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .assets
            .assignments
            .part_appearance_id(),
        None
    );
    host.commands(reopened).unwrap().redo().unwrap();
    host.save(reopened, "test", &mut source).unwrap();
    let bytes = host
        .files()
        .contents
        .borrow()
        .get(&destination)
        .unwrap()
        .clone();
    assert_eq!(
        DocumentPackage::decode(&bytes)
            .unwrap()
            .restore_assets()
            .unwrap()
            .assignments
            .part_appearance_id(),
        Some("oak")
    );
}
fn change_asset_override(host: &mut Host, id: u64) {
    host.commands(id)
        .unwrap()
        .run("Set appearance", |snapshot, _| {
            snapshot.assets.assignments.set_part_appearance("oak")?;
            snapshot.assets.assignments.set_face_appearance(
                &AssignmentReference::new("face", vec!["left".into()])?,
                "abs-black",
            )
        })
        .unwrap();
}
#[test]
fn failed_asset_save_keeps_committed_bytes_and_dirty_identity() {
    let mut host = Host::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(0);
    let destination = path("assets-failure.opd");
    let id = host
        .create(
            &destination,
            asset_template(),
            RecordingObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    host.save(id, "test", &mut source).unwrap();
    let before = host
        .files()
        .contents
        .borrow()
        .get(&destination)
        .unwrap()
        .clone();
    change_asset_override(&mut host, id);
    host.files().write_control.set(true);
    assert!(host.save(id, "test", &mut source).is_err());
    assert_eq!(
        host.files().contents.borrow().get(&destination).unwrap(),
        &before
    );
    assert!(host.workspace().needs_save(id).unwrap());
    host.files().write_control.set(false);
    host.save(id, "test", &mut source).unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
}

fn asset_template() -> DocumentPackage {
    // The named recording observer samples width on every event; its document must carry that parameter.
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{name: width, kind: user, expression: '1 cm'}]\n").unwrap()
}

fn portable_asset_definitions() -> oblikovati_model::assets::DocumentAssets {
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let material = catalog
        .materials()
        .get("steel")
        .unwrap()
        .duplicate(
            "portable",
            "Portable steel",
            oblikovati_api::types::ASSET_PROJECT,
        )
        .unwrap();
    let mut assets = oblikovati_model::assets::DocumentAssets::default();
    assets.embed_material(&material).unwrap();
    assets
}
