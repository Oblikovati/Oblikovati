// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn saved_plane() -> (Host, SequenceGuids, u64, FeatureId) {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let document = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let plane = create_plane(&mut host, document);
    host.save(document, "alpha", &mut source).unwrap();
    (host, source, document, plane)
}
fn flag(host: &Host, document: u64, plane: FeatureId) -> bool {
    snapshot(host, document)
        .work_planes
        .get(plane)
        .unwrap()
        .is_construction_datum()
}
fn change_flag(host: &mut Host, document: u64, plane: FeatureId, construction: bool) {
    host.commands(document)
        .unwrap()
        .run("Construction plane", |snapshot, _| {
            snapshot
                .work_planes
                .set_work_plane_construction(plane, construction)
        })
        .unwrap();
}

#[test]
fn construction_edits_are_dirty_undoable_and_keep_the_same_frame() {
    let (mut host, _, document, plane) = saved_plane();
    change_flag(&mut host, document, plane, true);
    assert!(flag(&host, document, plane));
    assert!(host.workspace().needs_save(document).unwrap());
    assert_plane_height(&host, document, plane, 1.0);
    host.commands(document).unwrap().undo().unwrap();
    assert!(!flag(&host, document, plane));
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    assert!(flag(&host, document, plane));
    assert_plane_height(&host, document, plane, 1.0);
}

#[test]
fn grouped_construction_abort_restores_saved_state() {
    let (mut host, _, document, plane) = saved_plane();
    host.commands(document)
        .unwrap()
        .begin("Abort construction")
        .unwrap();
    change_flag(&mut host, document, plane, true);
    host.commands(document).unwrap().abort().unwrap();
    assert!(!flag(&host, document, plane));
    assert!(!host.workspace().needs_save(document).unwrap());
    assert_plane_height(&host, document, plane, 1.0);
}

#[test]
fn unchanged_flag_and_missing_identity_do_not_dirty_or_add_history() {
    let (mut host, _, document, plane) = saved_plane();
    let labels = undo_count(&host, document);
    change_flag(&mut host, document, plane, false);
    let mut commands = host.commands(document).unwrap();
    let error = commands.run("Missing", |snapshot, _| {
        snapshot
            .work_planes
            .set_work_plane_construction(FeatureId(999), true)
    });
    assert!(error.unwrap_err().to_string().contains("999"));
    assert!(!host.workspace().needs_save(document).unwrap());
    assert_eq!(undo_count(&host, document), labels);
}

#[test]
fn construction_state_survives_save_reopen_and_copy_with_its_stable_feature() {
    let (mut host, mut source, document, plane) = saved_plane();
    change_flag(&mut host, document, plane, true);
    host.save(document, "alpha", &mut source).unwrap();
    let document = reopen(&mut host, document, &mut source);
    assert!(flag(&host, document, plane));
    assert!(!host.workspace().needs_save(document).unwrap());
    let copied = copy_construction_document(&mut host, document, &mut source);
    assert!(flag(&host, copied, plane));
    assert_plane_height(&host, copied, plane, 1.0);
    assert!(!host.workspace().needs_save(copied).unwrap());
}

fn undo_count(host: &Host, document: u64) -> usize {
    host.workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len()
}
fn copy_construction_document(host: &mut Host, document: u64, source: &mut SequenceGuids) -> u64 {
    let copy_path = path("construction-copy.obk");
    host.save_copy(document, &copy_path, None, "alpha", source)
        .unwrap();
    open_file(
        host,
        "construction-copy.obk",
        source,
        OpenActivation::Background,
    )
}
