// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::{DocumentObserverFactory, FileHost};

struct ReopenedObserver;
impl DocumentObserverFactory for ReopenedObserver {
    type Observer = QueryObserver;
    fn create(&mut self) -> Result<QueryObserver, String> {
        Ok(QueryObserver::default())
    }
}

fn current_width(files: &DiscoveryWorkspace, id: u64) -> Option<f64> {
    files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .parameters
        .get(1)
        .unwrap()
        .current_value()
        .map(|quantity| quantity.value)
}

#[test]
fn close_marks_readers_unavailable_and_reopen_restores_current_values() {
    let (mut files, source, target, mut guids) = linked_files();
    files.save(source, "test", &mut guids).unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .close(&CloseDocumentArgs {
            id: source,
            force: true,
        })
        .unwrap();
    assert_eq!(current_width(&files, target), None);
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    let reopened = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut ReopenedObserver),
    )
    .documents()
    .open(&OpenDocumentArgs {
        full_document_name: "/source.opd".into(),
        ..Default::default()
    })
    .unwrap();
    assert_ne!(reopened.id, source);
    assert_eq!(current_width(&files, target), Some(2.0));
    assert_eq!(files.workspace().active_document(), Some(reopened.id));
}

#[test]
fn partial_close_all_retains_command_and_refresh_failures() {
    let (mut files, source, target, mut guids) = linked_files();
    files.commands(target).unwrap().begin("Busy").unwrap();
    let error = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .close_all(&CloseAllDocumentsArgs { force: false })
        .unwrap_err()
        .to_string();
    assert!(error.contains("closing"), "{error}");
    assert!(error.contains("derived refresh is incomplete"), "{error}");
    assert!(files.workspace().session(source).is_err());
    assert_eq!(current_width(&files, target), Some(2.0));
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .abort()
        .unwrap();
    assert_eq!(current_width(&files, target), None);
}

#[test]
fn source_rebinding_updates_path_based_links_without_target_undo() {
    let (mut files, source, target, mut guids) = linked_files();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: source,
            new_full_document_name: "/moved.opd".into(),
        })
        .unwrap();
    assert_eq!(current_width(&files, target), None);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: source,
            new_full_document_name: "/source.opd".into(),
        })
        .unwrap();
    assert_eq!(current_width(&files, target), Some(2.0));
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
fn successful_creation_reports_ambiguous_links_without_discarding_new_owner() {
    let (mut files, _, target, mut guids) = linked_files();
    let error = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut ReopenedObserver),
    )
    .documents()
    .create(&CreateDocumentArgs {
        type_: "part".into(),
        name: "/source.opd".into(),
        ..Default::default()
    })
    .unwrap_err()
    .to_string();
    assert!(error.contains("command completed"), "{error}");
    assert!(error.contains("one unique open document"), "{error}");
    assert_eq!(files.workspace().documents().len(), 3);
    assert_eq!(files.workspace().session(target).unwrap().generation(), 0);
}

#[test]
fn partial_close_all_refreshes_remaining_readers_and_preserves_save_error() {
    let (mut files, source, target, mut guids) = linked_files();
    files
        .commands(target)
        .unwrap()
        .run("Unresolved", |snapshot, ids| {
            snapshot.parameters.insert_expression(
                ids.allocate().unwrap(),
                "output",
                USER_PARAM,
                "missing + 1 cm",
            )
        })
        .unwrap();
    let error = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .close_all(&CloseAllDocumentsArgs { force: false })
        .unwrap_err()
        .to_string();
    assert!(error.contains(&format!("closing {target}")), "{error}");
    assert!(!error.contains("derived refresh is incomplete"), "{error}");
    assert!(files.workspace().session(source).is_err());
    assert_eq!(current_width(&files, target), None);
}

#[test]
fn renamed_link_survives_file_save_close_reopen_and_source_edit() {
    let (mut files, source, target, mut guids) = linked_files();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .rename(&ParameterRenameArgs {
            name: "width".into(),
            new_name: "local_width".into(),
        })
        .unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: target,
            new_full_document_name: "/renamed-link.opd".into(),
        })
        .unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .close(&CloseDocumentArgs {
            id: target,
            force: true,
        })
        .unwrap();
    let reopened = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut ReopenedObserver),
    )
    .documents()
    .open(&OpenDocumentArgs {
        full_document_name: "/renamed-link.opd".into(),
        ..Default::default()
    })
    .unwrap();
    files.activate(source).unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "18 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let parameters = &files
        .workspace()
        .session(reopened.id)
        .unwrap()
        .snapshot()
        .parameters;
    let local = parameters.find("local_width").unwrap();
    assert_eq!(
        parameters
            .get(local)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        18.0
    );
    assert_eq!(
        parameters.derived_references(7).unwrap()[0].source_parameter,
        "width"
    );
}
