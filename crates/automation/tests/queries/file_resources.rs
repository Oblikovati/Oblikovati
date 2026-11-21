// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::resource::DocumentResource;
use std::collections::BTreeMap;

fn resources() -> BTreeMap<String, DocumentResource> {
    BTreeMap::from([(
        "font".into(),
        DocumentResource::imported("TrueTypeFont", "font.ttf", &[0, 255]),
    )])
}

#[test]
fn resource_edits_save_copy_and_reopen_with_independent_owned_state() {
    for copy in [false, true] {
        let (mut files, id, mut guids) = created_files(false);
        files.save(id, "test", &mut guids).unwrap();
        files
            .commands(id)
            .unwrap()
            .run("Import Font", |snapshot, _| {
                snapshot.resources = resources();
                Ok::<(), String>(())
            })
            .unwrap();
        assert!(files.workspace().needs_save(id).unwrap());
        files.commands(id).unwrap().undo().unwrap();
        assert!(!files.workspace().needs_save(id).unwrap());
        files.commands(id).unwrap().redo().unwrap();
        let path = if copy {
            "/fixtures/copy.opd"
        } else {
            "/fixtures/part.opd"
        };
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        if copy {
            client
                .documents()
                .save_copy_as(&SaveCopyAsArgs {
                    document: id,
                    target_file_name: path.into(),
                    ..Default::default()
                })
                .unwrap();
        } else {
            client
                .documents()
                .save(&SaveDocumentArgs { document: id })
                .unwrap();
        }
        assert_eq!(files.workspace().needs_save(id).unwrap(), copy);
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        client
            .documents()
            .close(&CloseDocumentArgs { id, force: copy })
            .unwrap();
        let mut factory = QueryObserverFactory {
            created: 0,
            limit: 1,
        };
        let client = Client::new(
            FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
        );
        let reopened = client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: path.into(),
                ..Default::default()
            })
            .unwrap();
        assert_eq!(
            files
                .workspace()
                .session(reopened.id)
                .unwrap()
                .snapshot()
                .resources,
            resources()
        );
        assert!(!files.workspace().needs_save(reopened.id).unwrap());
    }
}

#[test]
fn failed_resource_save_keeps_live_metadata_and_identity() {
    let (mut files, id, mut guids) = created_files(true);
    files
        .commands(id)
        .unwrap()
        .run("Import Font", |snapshot, _| {
            snapshot.resources = resources();
            Ok::<(), String>(())
        })
        .unwrap();
    let identity = files.workspace().identity(id).unwrap().clone();
    assert!(files.save(id, "test", &mut guids).is_err());
    assert_eq!(files.workspace().identity(id).unwrap(), &identity);
    assert_eq!(
        files.workspace().session(id).unwrap().snapshot().resources,
        resources()
    );
    assert!(files.workspace().needs_save(id).unwrap());
}

#[test]
fn template_resource_records_are_restored_and_remain_independent() {
    let (mut files, _, mut guids) = created_files(false);
    let package =
        DocumentPackage::empty(DOCUMENT_PART, "Font Template").with_resources(&resources());
    let id = files
        .create(
            Path::new("/fixtures/template-copy.opd"),
            package.clone(),
            QueryObserver::default(),
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    assert_eq!(
        files.workspace().session(id).unwrap().snapshot().resources,
        resources()
    );
    files
        .commands(id)
        .unwrap()
        .run("Remove Font", |snapshot, _| {
            snapshot.resources.remove("font");
            Ok::<(), String>(())
        })
        .unwrap();
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .resources
            .is_empty()
    );
    assert_eq!(*package.resources(), resources());
}
