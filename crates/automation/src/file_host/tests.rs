// SPDX-License-Identifier: GPL-2.0-only
//! File host query reentry regression.
use super::*;
use crate::host::tests::RejectUnexpectedCalls;
use oblikovati_model::identity::{DocumentGuid, IdentityTransitionError};
use oblikovati_persistence::file_access::NativeDocumentFiles;

struct RejectGuidRequests;
impl GuidSource for RejectGuidRequests {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        panic!("unexpected GUID request")
    }
}

#[test]
fn file_host_refuses_reentry_and_recovers_for_queries() {
    let mut files = FileWorkspace::<RejectUnexpectedCalls, _, _>::new(
        RejectUnexpectedCalls,
        NativeDocumentFiles,
    );
    let mut source = RejectGuidRequests;
    let host = FileHost::new(&mut files, &mut source, "test");
    {
        let _active = host.context.borrow_mut();
        for method in [
            METHOD_DOCUMENTS_LIST,
            METHOD_DOCUMENTS_LIST_ATTACHMENTS,
            METHOD_DOCUMENTS_ADD_ATTACHMENT,
            METHOD_DOCUMENTS_SAVE,
            METHOD_PARAMETERS_SET,
            METHOD_TRANSACTION_BEGIN,
        ] {
            assert!(
                host.call(method, b"{}")
                    .unwrap_err()
                    .to_string()
                    .contains("reentry is refused")
            );
        }
    }
    let reply: ListDocumentsResult =
        serde_json::from_slice(&host.call(METHOD_DOCUMENTS_LIST, b"").unwrap()).unwrap();
    assert_eq!(reply.documents, Some(vec![]));
}
