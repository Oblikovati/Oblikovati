// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::{BatchSaveArgs, BatchSaveItem};
use oblikovati_session::files::BatchSaveError;

fn batch_request(operation: &str, items: &[(u64, &Path)]) -> BatchSaveArgs {
    BatchSaveArgs {
        operation: operation.into(),
        items: Some(
            items
                .iter()
                .map(|&(document, target)| BatchSaveItem {
                    document,
                    target_file_name: target.to_str().unwrap().into(),
                })
                .collect(),
        ),
    }
}

#[test]
fn copy_batch_continues_after_conflict_and_keeps_order_and_source_identity() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let first = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let second = open_file(&mut host, "b.obk", &mut source, OpenActivation::Background);
    let identity = host.workspace().identity(first).unwrap().clone();
    let request = batch_request(
        "saveCopyAs",
        &[(first, &path("b.obk")), (second, &path("copy.obk"))],
    );
    let result = host.batch_save(request, "alpha", &mut source).unwrap();
    assert_eq!(result.saved, 1);
    let outcomes = result.results.unwrap();
    assert_eq!(
        outcomes
            .iter()
            .map(|item| item.document)
            .collect::<Vec<_>>(),
        [first, second]
    );
    assert!(!outcomes[0].ok);
    assert!(outcomes[0].error.contains("owned by an open document"));
    assert!(outcomes[1].ok);
    assert!(outcomes[1].error.is_empty());
    assert_eq!(
        outcomes[1].full_document_name,
        path("copy.obk").to_str().unwrap()
    );
    assert_eq!(host.workspace().identity(first).unwrap(), &identity);
    assert_eq!(host.path(second).unwrap(), path("b.obk"));
    assert!(
        host.files()
            .contents
            .borrow()
            .contains_key(&path("copy.obk"))
    );
}

#[test]
fn save_batch_uses_bindings_and_continues_after_policy_veto() {
    let mut host = host(RecordingWorkspace {
        deny_save: Some(1),
        ..Default::default()
    });
    let mut source = SequenceGuids(1000);
    let first = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let second = open_file(&mut host, "b.obk", &mut source, OpenActivation::Background);
    let ignored = path("ignored.obk");
    let result = host
        .batch_save(
            batch_request("save", &[(first, &ignored), (second, &ignored)]),
            "beta",
            &mut source,
        )
        .unwrap();
    let outcomes = result.results.unwrap();
    assert_eq!(result.saved, 1);
    assert!(!outcomes[0].ok);
    assert!(outcomes[0].error.contains("veto"));
    assert_eq!(
        outcomes[1].full_document_name,
        path("b.obk").to_str().unwrap()
    );
    assert_eq!(host.workspace().identity(first).unwrap().save_counter, 0);
    assert_eq!(host.workspace().identity(second).unwrap().save_counter, 1);
    assert!(!host.files().contents.borrow().contains_key(&ignored));
}

#[test]
fn save_as_batch_adopts_only_successful_bindings() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let first = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let second = open_file(&mut host, "b.obk", &mut source, OpenActivation::Background);
    let result = host
        .batch_save(
            batch_request(
                "saveAs",
                &[(first, &path("b.obk")), (second, &path("new.obk"))],
            ),
            "beta",
            &mut source,
        )
        .unwrap();
    assert_eq!(result.saved, 1);
    assert_eq!(host.path(first).unwrap(), path("a.obk"));
    assert_eq!(host.path(second).unwrap(), path("new.obk"));
    assert!(host.files().contents.borrow().contains_key(&path("b.obk")));
}

#[test]
fn invalid_operation_or_missing_source_refuses_every_write() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let count = source.0;
    let error = host
        .batch_save(
            batch_request("wrong", &[(id, &path("copy.obk"))]),
            "beta",
            &mut source,
        )
        .unwrap_err();
    assert!(error.to_string().contains("wrong"));
    assert!(std::error::Error::source(&error).is_none());
    let error = host
        .batch_save(
            batch_request(
                "saveCopyAs",
                &[(id, &path("copy.obk")), (99, &path("other.obk"))],
            ),
            "beta",
            &mut source,
        )
        .unwrap_err();
    assert!(matches!(error, BatchSaveError::Source(_)));
    assert!(error.to_string().contains("99"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(source.0, count);
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("copy.obk"))
    );
}

#[test]
fn duplicate_canonical_destinations_refuse_batch_before_writes() {
    let mut files = WorkspaceFiles::default();
    files.aliases.insert(path("alias.obk"), path("copy.obk"));
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    for operation in ["saveAs", "saveCopyAs", "save"] {
        let error = host
            .batch_save(
                batch_request(
                    operation,
                    &[(id, &path("copy.obk")), (id, &path("alias.obk"))],
                ),
                "beta",
                &mut source,
            )
            .unwrap_err();
        assert!(matches!(error, BatchSaveError::DuplicateDestination(_)));
        assert!(error.to_string().contains("one write"));
        assert!(std::error::Error::source(&error).is_none());
    }
    assert_eq!(host.workspace().identity(id).unwrap().save_counter, 0);
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("copy.obk"))
    );
}

#[test]
fn path_refusal_does_not_stop_later_copy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let result = host
        .batch_save(
            batch_request(
                "saveCopyAs",
                &[(id, Path::new("")), (id, &path("copy.obk"))],
            ),
            "beta",
            &mut source,
        )
        .unwrap();
    assert_eq!(result.saved, 1);
    let outcomes = result.results.unwrap();
    assert!(!outcomes[0].ok);
    assert!(outcomes[0].error.contains("absolute canonical path"));
    assert!(outcomes[1].ok);
}

#[test]
fn empty_batch_is_successful_with_an_empty_result_array() {
    let mut host = host(RecordingWorkspace::default());
    for items in [None, Some(Vec::new())] {
        let request = BatchSaveArgs {
            operation: "save".into(),
            items,
        };
        let result = host
            .batch_save(request, "alpha", &mut SequenceGuids(0))
            .unwrap();
        assert_eq!(result.saved, 0);
        assert_eq!(result.results, Some(Vec::new()));
    }
}

#[test]
fn native_batch_copies_continue_after_missing_parent() {
    let directory = tempfile::tempdir().unwrap();
    let original = directory.path().join("source.obk");
    let copy = directory.path().join("copy.obk");
    let missing = directory.path().join("absent/copy.obk");
    std::fs::write(&original, fixture(1)).unwrap();
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), NativeDocumentFiles);
    let mut source = SequenceGuids(1000);
    let id = host
        .open(
            &original,
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    let result = host
        .batch_save(
            batch_request("saveCopyAs", &[(id, &missing), (id, &copy)]),
            "beta",
            &mut source,
        )
        .unwrap();
    assert_eq!(result.saved, 1);
    assert_eq!(std::fs::read(&original).unwrap(), fixture(1));
    let package = DocumentPackage::decode(&std::fs::read(copy).unwrap()).unwrap();
    assert_eq!(
        package
            .restore_document_parameters()
            .unwrap()
            .parameters
            .ids(),
        [1]
    );
}

#[test]
fn failed_writes_report_every_item_without_adopting_revisions() {
    let files = WorkspaceFiles {
        fail_write: true,
        ..Default::default()
    };
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let count = source.0;
    let result = host
        .batch_save(
            batch_request(
                "saveCopyAs",
                &[(id, &path("one.obk")), (id, &path("two.obk"))],
            ),
            "beta",
            &mut source,
        )
        .unwrap();
    assert_eq!(result.saved, 0);
    let outcomes = result.results.unwrap();
    assert_eq!(outcomes.len(), 2);
    assert!(
        outcomes
            .iter()
            .all(|item| !item.ok && item.error.contains("write denied"))
    );
    assert_eq!(source.0, count + 6);
    assert_eq!(host.workspace().identity(id).unwrap().save_counter, 0);
}

#[cfg(unix)]
#[test]
fn non_utf8_bound_path_is_refused_before_a_wire_save_writes() {
    use std::os::unix::ffi::OsStringExt;
    let directory = tempfile::tempdir().unwrap();
    let original = directory
        .path()
        .join(std::ffi::OsString::from_vec(b"source-\xff.obk".to_vec()));
    std::fs::write(&original, fixture(1)).unwrap();
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), NativeDocumentFiles);
    let mut source = SequenceGuids(1000);
    let id = host
        .open(
            &original,
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    let count = source.0;
    let result = host
        .batch_save(
            batch_request("save", &[(id, Path::new(""))]),
            "beta",
            &mut source,
        )
        .unwrap();
    let outcome = &result.results.unwrap()[0];
    assert_eq!(result.saved, 0);
    assert!(!outcome.ok);
    assert!(outcome.error.contains("expected UTF-8"));
    assert_eq!(std::fs::read(original).unwrap(), fixture(1));
    assert_eq!(source.0, count);
}
