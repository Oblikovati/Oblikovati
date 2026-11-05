// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn open_uses_the_canonical_request_without_changing_the_path() {
    let draft = DocumentPathDraft {
        purpose: DocumentPathPurpose::Open,
        path: " /tmp/部品.opd ".into(),
    };
    let GuiShellAction::OpenDocument(request) = draft.action().unwrap() else {
        panic!("expected an OpenDocument action");
    };
    assert_eq!(request.full_document_name, draft.path);
    assert!(request.visible && !request.defer_content);
    assert_eq!(draft.purpose.title(), "Open document");
    assert_eq!(draft.purpose.button(), "Open");
}
#[test]
fn save_as_keeps_requesting_document_identity_in_the_canonical_request() {
    let mut files = GuiDocumentFiles::default();
    assert!(files.request_save_path(0).unwrap_err().contains("nonzero"));
    files.request_save_path(7).unwrap();
    let draft = files.draft.as_mut().unwrap();
    draft.path = "/tmp/part.opd".into();
    let GuiShellAction::SaveDocumentAs(request) = draft.action().unwrap() else {
        panic!("expected a SaveDocumentAs action");
    };
    assert_eq!(request.document, 7);
    assert_eq!(request.new_full_document_name, "/tmp/part.opd");
    assert_eq!(draft.purpose.title(), "Save document as");
    assert_eq!(draft.purpose.button(), "Save");
}
#[test]
fn document_switch_discards_only_a_document_bound_save_draft() {
    let mut files = GuiDocumentFiles::default();
    files.request_save_path(7).unwrap();
    files.synchronize_document(Some(7));
    assert!(files.draft.is_some());
    files.synchronize_document(Some(8));
    assert!(files.draft.is_none());
    files.begin(DocumentPathPurpose::Open);
    files.synchronize_document(None);
    assert!(files.draft.is_some());
}
#[test]
fn invalid_path_is_refused_before_the_text_widget_receives_it() {
    let mut files = GuiDocumentFiles::default();
    files.validate().unwrap();
    files.begin(DocumentPathPurpose::Open);
    files.draft.as_mut().unwrap().path = "bad\0path".into();
    assert!(files.validate().unwrap_err().contains("NUL"));
}
#[test]
fn file_actions_require_live_identity_and_no_open_transaction() {
    let open = DocumentPathDraft {
        purpose: DocumentPathPurpose::Open,
        path: "part.opd".into(),
    }
    .action()
    .unwrap();
    let save = DocumentPathDraft {
        purpose: DocumentPathPurpose::SaveAs(7),
        path: "part.opd".into(),
    }
    .action()
    .unwrap();
    let mut state = GuiCommandState::default();
    let close = GuiShellAction::CloseDocument(oblikovati_api::wire::CloseDocumentArgs {
        id: 7,
        force: false,
    });
    assert!(!close.dispatchable(&state));
    assert!(open.dispatchable(&state) && !save.dispatchable(&state));
    state.document = Some(7);
    assert!(close.dispatchable(&state));
    assert!(save.dispatchable(&state));
    state.transaction_open = true;
    assert!(!close.dispatchable(&state));
    assert!(!open.dispatchable(&state) && !save.dispatchable(&state));
}
