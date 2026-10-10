// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::DocumentObserverFactory;
use oblikovati_automation::FileHost;

struct QueryObserverFactory {
    created: usize,
    limit: usize,
}
impl DocumentObserverFactory for QueryObserverFactory {
    type Observer = QueryObserver;
    fn create(&mut self) -> Result<QueryObserver, String> {
        if self.created == self.limit {
            return Err("observer limit reached".into());
        }
        self.created += 1;
        Ok(QueryObserver::default())
    }
}

fn created_files(refuse_write: bool) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let mut files = DiscoveryWorkspace::new(
        QueryObserver::default(),
        DiscoveryFiles {
            refuse_write,
            ..Default::default()
        },
    );
    let mut source = QueryGuids(0);
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{name: width, kind: user, expression: '1 cm'}]\n").unwrap();
    let id = files
        .create(
            Path::new("/fixtures/part.opd"),
            package,
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    (files, id, source)
}

#[test]
fn file_host_edits_history_and_save_use_the_same_owned_document() {
    let (mut files, id, mut source) = created_files(false);
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "Resize".into(),
            })
            .unwrap();
        client
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "2 cm".into(),
                ..Default::default()
            })
            .unwrap();
        assert!(
            client
                .documents()
                .save(&SaveDocumentArgs { document: id })
                .is_err()
        );
        client.transactions().end().unwrap();
        let result = client
            .documents()
            .save(&SaveDocumentArgs { document: id })
            .unwrap();
        assert_eq!(result.full_document_name, "/fixtures/part.opd");
        assert!(
            client
                .transactions()
                .history(&TransactionHistoryArgs::default())
                .unwrap()
                .entries
                .unwrap()[0]
                .saved
        );
        assert!(!client.documents().list().unwrap().documents.unwrap()[0].dirty);
        client
            .transactions()
            .jump_to(&TransactionJumpToArgs {
                document: id,
                position: 0,
            })
            .unwrap();
        assert!(client.documents().list().unwrap().documents.unwrap()[0].dirty);
        client.transactions().redo().unwrap();
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "2 cm"
        );
        assert_eq!(
            client
                .documents()
                .close(&CloseDocumentArgs { id, force: false })
                .unwrap()
                .closed,
            1
        );
        assert_eq!(client.documents().list().unwrap().documents, Some(vec![]));
    }
    assert_eq!(files.files().packages.borrow().len(), 1);
    let reopened = files
        .open(
            Path::new("/fixtures/part.opd"),
            QueryObserver::default(),
            "test",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    assert_ne!(reopened, id);
    assert_eq!(
        files
            .workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .source,
        "2 cm"
    );
}

#[test]
fn close_saves_dirty_documents_and_force_close_discards_without_writing() {
    for force in [false, true] {
        let (mut files, id, mut source) = created_files(false);
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        assert_eq!(
            client
                .documents()
                .close(&CloseDocumentArgs { id, force })
                .unwrap()
                .closed,
            1
        );
        assert_eq!(files.files().packages.borrow().is_empty(), force);
        assert!(files.workspace().documents().is_empty());
    }
}

#[test]
fn file_failures_keep_document_binding_identity_and_dirty_state() {
    let (mut files, id, mut source) = created_files(true);
    let identity = files.workspace().identity(id).unwrap().clone();
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        assert!(
            client
                .documents()
                .save(&SaveDocumentArgs { document: id })
                .is_err()
        );
        assert!(
            client
                .documents()
                .close(&CloseDocumentArgs { id, force: false })
                .is_err()
        );
        assert!(
            client
                .documents()
                .activate(&ActivateDocumentArgs { id: 999 })
                .is_err()
        );
        assert!(
            client
                .documents()
                .close(&CloseDocumentArgs {
                    id: 999,
                    force: true
                })
                .is_err()
        );
        assert!(
            client
                .documents()
                .save(&SaveDocumentArgs { document: 0 })
                .is_err()
        );
    }
    assert_eq!(files.workspace().identity(id).unwrap(), &identity);
    assert!(files.workspace().needs_save(id).unwrap());
    assert_eq!(files.workspace().active_document(), Some(id));
    assert!(files.files().packages.borrow().is_empty());
}

#[test]
fn activation_changes_the_target_of_shared_parameter_commands() {
    let (mut files, first, mut source) = created_files(false);
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 2\n").unwrap();
    let second = files
        .create(
            Path::new("/fixtures/assembly.oad"),
            package,
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    assert!(
        client
            .documents()
            .activate(&ActivateDocumentArgs { id: first })
            .unwrap()
            .ok
    );
    client
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "3 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        client
            .documents()
            .list()
            .unwrap()
            .documents
            .unwrap()
            .iter()
            .filter(|document| document.active)
            .map(|document| document.id)
            .collect::<Vec<_>>(),
        [first]
    );
    assert!(
        files
            .workspace()
            .session(second)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .is_empty()
    );
}

#[cfg(unix)]
#[test]
fn non_utf8_response_path_is_refused_before_saving() {
    use std::os::unix::ffi::OsStringExt;
    let (mut files, id, mut source) = created_files(false);
    let path = PathBuf::from(std::ffi::OsString::from_vec(
        b"/fixtures/non-\xff.opd".to_vec(),
    ));
    files.save_as(id, &path, "test", &mut source).unwrap();
    let counter = files.workspace().identity(id).unwrap().save_counter;
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let error = client
        .documents()
        .save(&SaveDocumentArgs { document: id })
        .unwrap_err();
    assert!(error.to_string().contains("UTF-8"));
    assert_eq!(
        files.workspace().identity(id).unwrap().save_counter,
        counter
    );
}

#[path = "file_copies.rs"]
mod copies;

#[path = "file_batch.rs"]
mod batch;

#[path = "file_open.rs"]
mod opening;

#[path = "file_create.rs"]
mod create;

#[path = "file_close_all.rs"]
mod close_all;

#[path = "file_units.rs"]
mod units;

#[path = "file_parameter_ids.rs"]
mod parameter_ids;

#[path = "file_property_sets.rs"]
mod property_sets;

#[path = "file_property_format.rs"]
mod property_format;

#[path = "file_attributes.rs"]
mod attributes;

#[path = "file_attribute_api.rs"]
mod attribute_api;

#[path = "file_interests.rs"]
mod interests;

#[path = "file_interest_api.rs"]
mod interest_api;

#[path = "file_resources.rs"]
mod resources;

#[path = "file_attachments.rs"]
mod attachments;
