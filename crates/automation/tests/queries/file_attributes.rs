// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::attribute::AttributeTarget;

#[test]
fn attribute_only_edits_save_copy_and_reopen_as_owned_snapshot_state() {
    for copy in [false, true] {
        let (mut files, id, mut guids) = created_files(false);
        files.save(id, "test", &mut guids).unwrap();
        let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
        files
            .commands(id)
            .unwrap()
            .run("Tag", |snapshot, _| {
                snapshot
                    .attributes
                    .put(&target, "AddIn", "tag", VariantValue::Double(-0.0))
            })
            .unwrap();
        assert!(files.workspace().needs_save(id).unwrap());
        files.commands(id).unwrap().undo().unwrap();
        assert!(!files.workspace().needs_save(id).unwrap());
        files.commands(id).unwrap().redo().unwrap();
        let expected = files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .attributes
            .clone();
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
        assert!(
            files
                .workspace()
                .session(reopened.id)
                .unwrap()
                .snapshot()
                .attributes
                .same_recipe(&expected)
        );
        assert!(!files.workspace().needs_save(reopened.id).unwrap());
    }
}

#[test]
fn failed_attribute_save_preserves_the_live_edit_and_dirty_state() {
    let (mut files, id, mut guids) = created_files(true);
    let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
    files
        .commands(id)
        .unwrap()
        .run("Tag", |snapshot, _| {
            snapshot
                .attributes
                .put(&target, "AddIn", "tag", VariantValue::Integer(8))
        })
        .unwrap();
    let before = files.workspace().identity(id).unwrap().clone();
    assert!(files.save(id, "test", &mut guids).is_err());
    assert_eq!(files.workspace().identity(id).unwrap(), &before);
    assert!(files.workspace().needs_save(id).unwrap());
    assert_eq!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .attributes
            .get(&target, "AddIn", "tag")
            .unwrap()
            .value,
        VariantValue::Integer(8)
    );
}

#[test]
fn new_document_restores_template_attributes_without_sharing_ownership() {
    let (mut files, _, mut guids) = created_files(false);
    let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
    let mut attributes = oblikovati_model::attribute::DocumentAttributes::default();
    attributes
        .put(&target, "AddIn", "tag", VariantValue::Integer(4))
        .unwrap();
    let package = DocumentPackage::empty(DOCUMENT_PART, "Template")
        .with_attributes(&attributes)
        .unwrap();
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
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .attributes
            .same_recipe(&attributes)
    );
    files
        .commands(id)
        .unwrap()
        .run("Change tag", |snapshot, _| {
            snapshot
                .attributes
                .put(&target, "AddIn", "tag", VariantValue::Integer(5))
        })
        .unwrap();
    assert!(
        package
            .restore_attributes()
            .unwrap()
            .same_recipe(&attributes)
    );
}
