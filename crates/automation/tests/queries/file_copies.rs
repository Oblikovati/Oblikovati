// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn save_as_rebinds_the_owner_and_returns_the_resolved_destination() {
    let (mut files, id, mut source) = created_files(false);
    files.save(id, "test", &mut source).unwrap();
    let original = files.files().packages.borrow()[Path::new("/fixtures/part.opd")].clone();
    let identity = files.workspace().identity(id).unwrap().internal_name;
    files
        .files()
        .save_aliases
        .borrow_mut()
        .insert("/fixtures/alias.opd".into(), "/fixtures/renamed.opd".into());
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        client
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "3 cm".into(),
                ..Default::default()
            })
            .unwrap();
        let result = client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: id,
                new_full_document_name: "/fixtures/alias.opd".into(),
            })
            .unwrap();
        assert_eq!(result.full_document_name, "/fixtures/renamed.opd");
        let document = client
            .documents()
            .list()
            .unwrap()
            .documents
            .unwrap()
            .remove(0);
        assert_eq!(document.name, "renamed");
        assert!(!document.dirty);
    }
    assert_eq!(files.path(id).unwrap(), Path::new("/fixtures/renamed.opd"));
    assert_eq!(
        files.workspace().identity(id).unwrap().internal_name,
        identity
    );
    assert_eq!(
        files.files().packages.borrow()[Path::new("/fixtures/part.opd")],
        original
    );
    let bytes = files.files().packages.borrow()[Path::new("/fixtures/renamed.opd")].clone();
    let package = DocumentPackage::decode(&bytes).unwrap();
    assert_eq!(
        package
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1)
            .unwrap()
            .source,
        "3 cm"
    );
}

#[test]
fn copy_metadata_changes_only_the_new_file_and_mints_an_independent_identity() {
    let (mut files, id, mut source) = created_files(false);
    let identity = files.workspace().identity(id).unwrap().clone();
    let path = files.path(id).unwrap().to_owned();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .save_copy_as(&SaveCopyAsArgs {
            document: id,
            target_file_name: "/fixtures/copy.opd".into(),
            metadata: Some(std::boxed::Box::new(NewFileMetadata {
                display_name: "Archive Ω".into(),
                sub_type: "flavor".into(),
                file_name: "ignored.opd".into(),
                ..Default::default()
            })),
        })
        .unwrap();
    assert_eq!(result.full_document_name, "/fixtures/copy.opd");
    assert_eq!(files.path(id).unwrap(), path);
    assert_eq!(files.workspace().identity(id).unwrap(), &identity);
    assert!(files.workspace().needs_save(id).unwrap());
    assert_eq!(files.document_info(id).unwrap().name, "part");
    let package =
        DocumentPackage::decode(&files.files().packages.borrow()[Path::new("/fixtures/copy.opd")])
            .unwrap();
    assert_eq!(package.manifest().display_name, "Archive Ω");
    assert_eq!(package.manifest().sub_type, "flavor");
    assert_ne!(
        package.identity_record().unwrap().unwrap().internal_name,
        identity.internal_name.to_string()
    );
    assert_eq!(files.workspace().documents(), [id]);
}

#[test]
fn failed_save_as_and_copy_keep_binding_and_identity_unchanged() {
    let (mut files, id, mut source) = created_files(true);
    let identity = files.workspace().identity(id).unwrap().clone();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    assert!(
        client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: id,
                new_full_document_name: "/fixtures/new.opd".into()
            })
            .is_err()
    );
    assert!(
        client
            .documents()
            .save_copy_as(&SaveCopyAsArgs {
                document: id,
                target_file_name: "/fixtures/copy.opd".into(),
                ..Default::default()
            })
            .is_err()
    );
    assert!(
        client
            .documents()
            .save_copy_as(&SaveCopyAsArgs {
                document: id,
                target_file_name: "/fixtures/part.opd".into(),
                ..Default::default()
            })
            .is_err()
    );
    assert_eq!(files.workspace().identity(id).unwrap(), &identity);
    assert_eq!(files.path(id).unwrap(), Path::new("/fixtures/part.opd"));
    assert!(files.files().packages.borrow().is_empty());
    assert!(files.workspace().needs_save(id).unwrap());
}

#[cfg(unix)]
#[test]
fn non_utf8_resolved_save_as_is_refused_before_storage() {
    use std::os::unix::ffi::OsStringExt;
    let (mut files, id, mut source) = created_files(false);
    let destination = PathBuf::from(std::ffi::OsString::from_vec(
        b"/fixtures/non-\xff.opd".to_vec(),
    ));
    files
        .files()
        .save_aliases
        .borrow_mut()
        .insert("/fixtures/alias.opd".into(), destination);
    let identity = files.workspace().identity(id).unwrap().clone();
    let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: id,
            new_full_document_name: "/fixtures/alias.opd".into(),
        })
        .unwrap_err();
    assert!(error.to_string().contains("UTF-8"));
    assert!(files.files().packages.borrow().is_empty());
    assert_eq!(files.workspace().identity(id).unwrap(), &identity);
    assert_eq!(files.path(id).unwrap(), Path::new("/fixtures/part.opd"));
}
