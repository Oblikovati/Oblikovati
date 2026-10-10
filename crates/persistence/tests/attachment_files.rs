// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    attachment::{AttachmentContents, DocumentAttachments, ExternalAttachmentFiles},
    identity::SystemGuidSource,
};
use oblikovati_persistence::attachment_files::NativeAttachmentFiles;
use std::io;

#[test]
fn native_attachment_capture_keeps_payload_after_source_removal() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("source.bin");
    let path = path.to_str().unwrap();
    std::fs::write(path, [0, 255]).unwrap();
    let mut files = NativeAttachmentFiles;
    let mut attachments = DocumentAttachments::default();
    attachments
        .attach(
            "embedded",
            ATTACHMENT_EMBEDDED,
            path,
            Some(&mut files),
            &mut SystemGuidSource,
        )
        .unwrap();
    attachments
        .attach(
            "linked",
            ATTACHMENT_LINKED,
            path,
            Some(&mut files),
            &mut SystemGuidSource,
        )
        .unwrap();
    assert_eq!(
        attachments
            .get("linked")
            .unwrap()
            .probe(Some(&mut files))
            .unwrap()
            .status,
        REFERENCE_UP_TO_DATE
    );
    std::fs::remove_file(path).unwrap();
    assert_eq!(
        attachments
            .get("linked")
            .unwrap()
            .probe(Some(&mut files))
            .unwrap()
            .status,
        REFERENCE_MISSING
    );
    let embedded = attachments.get("embedded").unwrap();
    assert_eq!(
        embedded.probe(Some(&mut files)).unwrap().status,
        REFERENCE_UP_TO_DATE
    );
    assert!(
        matches!(embedded.contents(), AttachmentContents::Embedded { bytes, .. } if bytes == &[0, 255])
    );
    assert_eq!(
        files.read(path).unwrap_err().kind(),
        io::ErrorKind::NotFound
    );
    assert!(
        files
            .modified(directory.path().to_str().unwrap())
            .unwrap()
            .is_none()
    );
}

#[test]
fn native_attachment_probe_preserves_invalid_path_errors() {
    let mut files = NativeAttachmentFiles;
    assert_eq!(
        files.modified("bad\0path").unwrap_err().kind(),
        io::ErrorKind::InvalidInput
    );
    assert_eq!(
        files.read("bad\0path").unwrap_err().kind(),
        io::ErrorKind::InvalidInput
    );
}
