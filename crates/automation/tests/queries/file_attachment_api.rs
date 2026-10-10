// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::attachment::{ExternalAttachmentFiles, ResolvedAttachmentSource};
use std::{
    cell::RefCell,
    io,
    rc::Rc,
    time::{Duration, SystemTime},
};

#[derive(Default)]
struct ForeignState {
    missing: bool,
    changed: bool,
    fail_read: bool,
    fail_stat_at: usize,
    reads: usize,
    stats: usize,
}
struct ForeignFiles(Rc<RefCell<ForeignState>>);
impl ExternalAttachmentFiles for ForeignFiles {
    fn inspect_source(&mut self, path: &str) -> io::Result<Option<ResolvedAttachmentSource>> {
        let mut state = self.0.borrow_mut();
        state.stats += 1;
        if state.stats == state.fail_stat_at {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "foreign metadata denied",
            ));
        }
        Ok((!state.missing).then_some(ResolvedAttachmentSource {
            resolved_file_name: path.into(),
            modified: SystemTime::UNIX_EPOCH
                + Duration::new(if state.changed { 101 } else { 100 }, 1),
        }))
    }
    fn read(&mut self, _: &str) -> io::Result<Vec<u8>> {
        let mut state = self.0.borrow_mut();
        state.reads += 1;
        if state.fail_read {
            return Err(io::Error::new(
                io::ErrorKind::NotFound,
                "foreign payload missing",
            ));
        }
        Ok(vec![0, 255])
    }
}
fn add(document: u64, name: &str, kind: AttachmentKind) -> AddAttachmentArgs {
    AddAttachmentArgs {
        document,
        name: name.into(),
        kind,
        full_file_name: "source.bin".into(),
    }
}

#[test]
fn typed_attachment_capture_status_undo_and_file_round_trip() {
    let (mut files, id, mut guids) = created_files(false);
    let state = Rc::new(RefCell::new(ForeignState::default()));
    let mut external = ForeignFiles(state.clone());
    let client = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_attachment_files(&mut external),
    );
    assert_eq!(
        client
            .documents()
            .attachments(&ListAttachmentsArgs { document: id })
            .unwrap()
            .attachments,
        Some(Vec::new())
    );
    let linked = client
        .documents()
        .add_attachment(&add(id, "linked", ATTACHMENT_LINKED))
        .unwrap();
    assert_eq!(
        linked.last_known_file_time,
        "1970-01-01T00:01:40.000000001Z"
    );
    assert_eq!(linked.resolved_file_name, "source.bin");
    assert_eq!(linked.status, REFERENCE_UP_TO_DATE);
    let embedded = client
        .documents()
        .add_attachment(&add(id, "embedded", ATTACHMENT_EMBEDDED))
        .unwrap();
    assert!(!embedded.resource.is_empty());
    assert!(embedded.resolved_file_name.is_empty());
    assert!(embedded.browser_visible);
    state.borrow_mut().changed = true;
    let before = state.borrow().stats;
    let listed = client
        .documents()
        .attachments(&ListAttachmentsArgs { document: id })
        .unwrap()
        .attachments
        .unwrap();
    assert_eq!(
        listed
            .iter()
            .map(|entry| entry.name.as_str())
            .collect::<Vec<_>>(),
        ["linked", "embedded"]
    );
    assert_eq!(listed[0].status, REFERENCE_OUT_OF_DATE);
    assert_eq!(listed[1].status, REFERENCE_UP_TO_DATE);
    assert_eq!(state.borrow().stats, before + 1);
    state.borrow_mut().missing = true;
    let listed = client
        .documents()
        .attachments(&ListAttachmentsArgs { document: id })
        .unwrap()
        .attachments
        .unwrap();
    assert_eq!(listed[0].status, REFERENCE_MISSING);
    assert!(listed[0].resolved_file_name.is_empty());
    let fail_stat_at = state.borrow().stats + 1;
    state.borrow_mut().fail_stat_at = fail_stat_at;
    assert!(
        client
            .documents()
            .attachments(&ListAttachmentsArgs { document: id })
            .is_err()
    );
    state.borrow_mut().fail_stat_at = 0;
    assert!(
        client
            .documents()
            .remove_attachment(&RemoveAttachmentArgs {
                document: id,
                name: "embedded".into()
            })
            .unwrap()
            .ok
    );
    files.commands(id).unwrap().undo().unwrap();
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client
        .documents()
        .save(&SaveDocumentArgs { document: id })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs { id, force: false })
        .unwrap();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 1,
    };
    let client = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
    );
    let opened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/fixtures/part.opd".into(),
            ..Default::default()
        })
        .unwrap();
    let records = client
        .documents()
        .attachments(&ListAttachmentsArgs {
            document: opened.id,
        })
        .unwrap()
        .attachments
        .unwrap();
    assert_eq!(records[0].status, REFERENCE_UNKNOWN);
    assert_eq!(records[1].resource, embedded.resource);
    assert_eq!(state.borrow().reads, 1);
}

#[test]
fn attachment_errors_leave_state_and_history_unchanged() {
    for (kind, fail_read, fail_stat_at) in [
        (ATTACHMENT_EMBEDDED, true, 0),
        (ATTACHMENT_LINKED, false, 2),
    ] {
        let (mut files, id, mut guids) = created_files(false);
        let state = Rc::new(RefCell::new(ForeignState {
            fail_read,
            fail_stat_at,
            ..Default::default()
        }));
        let mut external = ForeignFiles(state);
        let client = Client::new(
            FileHost::new(&mut files, &mut guids, "test").with_attachment_files(&mut external),
        );
        let error = client
            .documents()
            .add_attachment(&add(id, "failed", kind))
            .unwrap_err();
        assert!(error.to_string().contains("foreign"));
        assert!(
            files
                .workspace()
                .session(id)
                .unwrap()
                .snapshot()
                .attachments
                .entries()
                .is_empty()
        );
        assert!(
            files
                .workspace()
                .session(id)
                .unwrap()
                .undo_labels()
                .is_empty()
        );
    }
}

#[test]
fn attachment_requests_validate_document_names_and_provider_without_changing_focus() {
    let (mut files, id, mut guids) = created_files(false);
    let focus = files.workspace().active_document();
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    for document in [0, u64::MAX] {
        assert!(
            client
                .documents()
                .attachments(&ListAttachmentsArgs { document })
                .is_err()
        );
        assert!(
            client
                .documents()
                .add_attachment(&add(document, "x", ATTACHMENT_GENERIC))
                .is_err()
        );
        assert!(
            client
                .documents()
                .remove_attachment(&RemoveAttachmentArgs {
                    document,
                    name: "x".into()
                })
                .is_err()
        );
    }
    assert!(
        client
            .documents()
            .add_attachment(&add(id, "embedded", ATTACHMENT_EMBEDDED))
            .unwrap_err()
            .to_string()
            .contains("provider")
    );
    assert!(
        client
            .documents()
            .add_attachment(&add(id, "", ATTACHMENT_GENERIC))
            .is_err()
    );
    let generic = client
        .documents()
        .add_attachment(&add(id, "generic", ATTACHMENT_GENERIC))
        .unwrap();
    assert_eq!(generic.status, REFERENCE_UNKNOWN);
    assert!(
        client
            .documents()
            .add_attachment(&add(id, "generic", ATTACHMENT_GENERIC))
            .is_err()
    );
    assert!(
        client
            .documents()
            .remove_attachment(&RemoveAttachmentArgs {
                document: id,
                name: "missing".into()
            })
            .unwrap_err()
            .to_string()
            .contains("missing")
    );
    assert_eq!(files.workspace().active_document(), focus);
    let client = Client::new(SessionQueryHost::new(files.workspace()));
    assert_eq!(
        client
            .documents()
            .attachments(&ListAttachmentsArgs { document: id })
            .unwrap()
            .attachments
            .unwrap()[0],
        generic
    );
}

#[test]
fn vetoed_attachment_command_discards_the_captured_candidate() {
    let (mut files, _, mut guids) = created_files(false);
    let id = files
        .create(
            Path::new("/fixtures/locked.opd"),
            DocumentPackage::empty(DOCUMENT_PART, "Locked"),
            QueryObserver {
                veto: true,
                ..Default::default()
            },
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    let state = Rc::new(RefCell::new(ForeignState::default()));
    let mut external = ForeignFiles(state.clone());
    let client = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_attachment_files(&mut external),
    );
    let error = client
        .documents()
        .add_attachment(&add(id, "embedded", ATTACHMENT_EMBEDDED))
        .unwrap_err();
    assert!(error.to_string().contains("settings are locked"));
    // Commit observers inspect the candidate after capture; veto rolls back model state.
    assert_eq!(state.borrow().reads, 1);
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .attachments
            .entries()
            .is_empty()
    );
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}

#[test]
fn project_attachment_api_preserves_reference_and_reports_resolved_source() {
    use oblikovati_persistence::project::{ProjectAttachmentFiles, ProjectFileLookup};
    let (mut files, id, mut guids) = created_files(false);
    let lookup = ProjectFileLookup::new(Some("/project".into()), vec![], None).unwrap();
    let mut external = ForeignFiles(Rc::new(RefCell::new(ForeignState::default())));
    let mut project_files = ProjectAttachmentFiles::new(&lookup, &mut external);
    let client = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_attachment_files(&mut project_files),
    );
    let result = client
        .documents()
        .add_attachment(&add(id, "linked", ATTACHMENT_LINKED))
        .unwrap();
    assert_eq!(result.full_file_name, "source.bin");
    assert_eq!(result.resolved_file_name, "/project/source.bin");
    let listed = client
        .documents()
        .attachments(&ListAttachmentsArgs { document: id })
        .unwrap()
        .attachments
        .unwrap();
    assert_eq!(listed, [result]);
}
