// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::FileHost;

fn width(files: &DiscoveryWorkspace, id: u64) -> f64 {
    files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .parameters
        .get(1)
        .unwrap()
        .current_value()
        .unwrap()
        .value
}

fn set_width(
    files: &mut DiscoveryWorkspace,
    guids: &mut QueryGuids,
    expression: &str,
) -> Result<(), oblikovati_api::Error> {
    Client::new(FileHost::new(files, guids, "test"))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: expression.into(),
            ..Default::default()
        })
        .map(|_| ())
}

#[test]
fn file_api_edits_undo_redo_and_jump_refresh_readers() {
    let (mut files, source, target, mut guids) = linked_files();
    files.activate(source).unwrap();
    set_width(&mut files, &mut guids, "8 cm").unwrap();
    assert_eq!(width(&files, target), 8.0);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(width(&files, target), 2.0);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(width(&files, target), 8.0);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document: source,
            position: 0,
        })
        .unwrap();
    assert_eq!(width(&files, target), 2.0);
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    assert_eq!(files.workspace().active_document(), Some(source));
}

#[test]
fn nested_transactions_defer_refresh_until_outer_end_and_abort_discards_preview() {
    let (mut files, source, target, mut guids) = linked_files();
    files.activate(source).unwrap();
    for label in ["Outer", "Inner"] {
        Client::new(FileHost::new(&mut files, &mut guids, "test"))
            .transactions()
            .begin(&TransactionBeginArgs {
                label: label.into(),
            })
            .unwrap();
    }
    set_width(&mut files, &mut guids, "9 cm").unwrap();
    assert_eq!(width(&files, target), 2.0);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .end()
        .unwrap();
    assert_eq!(width(&files, target), 2.0);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .end()
        .unwrap();
    assert_eq!(width(&files, target), 9.0);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Preview".into(),
        })
        .unwrap();
    set_width(&mut files, &mut guids, "12 cm").unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .abort()
        .unwrap();
    assert_eq!(width(&files, target), 9.0);
    assert_eq!(width(&files, source), 9.0);
}

#[test]
fn refresh_failure_reports_committed_source_and_target_abort_catches_up() {
    let (mut files, source, target, mut guids) = linked_files();
    files.commands(target).unwrap().begin("Busy").unwrap();
    files.activate(source).unwrap();
    let error = set_width(&mut files, &mut guids, "11 cm")
        .unwrap_err()
        .to_string();
    assert!(error.contains("command completed"), "{error}");
    assert!(error.contains("failed tables"), "{error}");
    assert_eq!(width(&files, source), 11.0);
    assert_eq!(width(&files, target), 2.0);
    files.activate(target).unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .abort()
        .unwrap();
    assert_eq!(width(&files, target), 11.0);
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}

#[test]
fn target_undo_restores_recipe_then_refreshes_from_current_source() {
    let (mut files, source, target, mut guids) = linked_files();
    files
        .commands(target)
        .unwrap()
        .run("Rename", |snapshot, _| {
            snapshot.parameters.rename(1, "local_width")
        })
        .unwrap();
    files.activate(source).unwrap();
    set_width(&mut files, &mut guids, "13 cm").unwrap();
    files.activate(target).unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(width(&files, target), 13.0);
    assert_eq!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .name,
        "width"
    );
    assert_eq!(
        files.workspace().session(target).unwrap().redo_labels(),
        ["Rename"]
    );
}

#[test]
fn graph_preflight_failure_reports_source_commit_without_refreshing_target() {
    let (mut files, source, target, mut guids) = linked_files();
    let _duplicate = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_PART, "/source.opd"),
            QueryObserver::default(),
            "test",
            &mut guids,
        )
        .unwrap();
    files.activate(source).unwrap();
    let error = set_width(&mut files, &mut guids, "14 cm")
        .unwrap_err()
        .to_string();
    assert!(error.contains("command completed"), "{error}");
    assert!(error.contains("one unique open document"), "{error}");
    assert_eq!(width(&files, source), 14.0);
    assert_eq!(width(&files, target), 2.0);
    assert_eq!(files.workspace().session(target).unwrap().generation(), 0);
}
