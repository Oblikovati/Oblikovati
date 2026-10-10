// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn batch_files() -> (DiscoveryWorkspace, [u64; 2], QueryGuids) {
    let (mut files, first, mut source) = created_files(false);
    let second = files
        .create(
            Path::new("/fixtures/second.opd"),
            DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap(),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    (files, [first, second], source)
}

#[test]
fn batch_save_continues_after_failed_item_and_preserves_request_order() {
    let (mut files, [first, second], mut source) = batch_files();
    files.commands(first).unwrap().begin("preview").unwrap();
    let response = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .batch_save(&BatchSaveArgs {
            operation: "save".into(),
            items: Some(vec![
                BatchSaveItem {
                    document: first,
                    target_file_name: String::new(),
                },
                BatchSaveItem {
                    document: second,
                    target_file_name: String::new(),
                },
            ]),
        })
        .unwrap();
    assert_eq!(response.saved, 1);
    let results = response.results.unwrap();
    assert_eq!(
        results.iter().map(|item| item.document).collect::<Vec<_>>(),
        [first, second]
    );
    assert!(!results[0].ok && results[1].ok);
    assert!(results[0].error.contains("transaction"));
    assert!(results[1].error.is_empty());
    assert_eq!(results[0].full_document_name, "/fixtures/part.opd");
    assert_eq!(results[1].full_document_name, "/fixtures/second.opd");
    assert_eq!(files.files().packages.borrow().len(), 1);
    assert!(files.workspace().needs_save(first).unwrap());
    assert!(!files.workspace().needs_save(second).unwrap());
    files.commands(first).unwrap().abort().unwrap();
}

#[test]
fn batch_save_as_rebinds_and_copy_preserves_source_state() {
    for operation in ["saveAs", "saveCopyAs"] {
        let (mut files, ids, mut source) = batch_files();
        let before: Vec<_> = ids
            .iter()
            .map(|id| files.workspace().identity(*id).unwrap().clone())
            .collect();
        let response = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .documents()
            .batch_save(&BatchSaveArgs {
                operation: operation.into(),
                items: Some(
                    ids.iter()
                        .enumerate()
                        .map(|(index, id)| BatchSaveItem {
                            document: *id,
                            target_file_name: format!("/fixtures/output{index}.opd"),
                        })
                        .collect(),
                ),
            })
            .unwrap();
        assert_eq!(response.saved, 2);
        assert!(
            response
                .results
                .unwrap()
                .iter()
                .all(|item| item.ok && item.error.is_empty())
        );
        assert_eq!(files.files().packages.borrow().len(), 2);
        for (index, id) in ids.into_iter().enumerate() {
            if operation == "saveAs" {
                assert_eq!(
                    files.path(id).unwrap(),
                    Path::new(&format!("/fixtures/output{index}.opd"))
                );
                assert!(!files.workspace().needs_save(id).unwrap());
                assert_eq!(
                    files.workspace().identity(id).unwrap().internal_name,
                    before[index].internal_name
                );
            } else {
                assert_eq!(files.workspace().identity(id).unwrap(), &before[index]);
                assert!(files.workspace().needs_save(id).unwrap());
                assert!(!files.path(id).unwrap().to_string_lossy().contains("output"));
            }
        }
    }
}

#[test]
fn batch_validation_rejects_entire_request_before_any_file_is_written() {
    let (mut files, [first, second], mut source) = batch_files();
    let before = files.workspace().identity(first).unwrap().clone();
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        for (operation, documents) in [
            ("unknown", [first, second]),
            ("save", [first, 999]),
            ("saveAs", [first, second]),
        ] {
            assert!(
                client
                    .documents()
                    .batch_save(&BatchSaveArgs {
                        operation: operation.into(),
                        items: Some(
                            documents
                                .into_iter()
                                .map(|document| BatchSaveItem {
                                    document,
                                    target_file_name: "/fixtures/same.opd".into()
                                })
                                .collect()
                        ),
                    })
                    .is_err()
            );
        }
        let empty = client
            .documents()
            .batch_save(&BatchSaveArgs {
                operation: "save".into(),
                items: None,
            })
            .unwrap();
        assert_eq!(empty.saved, 0);
        assert_eq!(empty.results, Some(vec![]));
    }
    assert!(files.files().packages.borrow().is_empty());
    assert_eq!(files.workspace().identity(first).unwrap(), &before);
}
