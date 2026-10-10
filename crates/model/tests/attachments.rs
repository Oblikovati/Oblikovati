// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::*;
use oblikovati_model::{
    attachment::{
        AttachmentContents, DocumentAttachments, ExternalAttachmentFiles, ResolvedAttachmentSource,
    },
    identity::{DocumentGuid, GuidSource, IdentityTransitionError},
};
use std::{
    error::Error,
    io,
    time::{Duration, SystemTime},
};

struct AttachmentFiles {
    modified: Option<SystemTime>,
    bytes: Vec<u8>,
    fail: bool,
    stats: usize,
    reads: usize,
}
impl ExternalAttachmentFiles for AttachmentFiles {
    fn inspect_source(&mut self, path: &str) -> io::Result<Option<ResolvedAttachmentSource>> {
        self.stats += 1;
        if self.fail {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "fixture denied",
            ));
        }
        Ok(self.modified.map(|modified| ResolvedAttachmentSource {
            resolved_file_name: path.into(),
            modified,
        }))
    }
    fn read(&mut self, _: &str) -> io::Result<Vec<u8>> {
        self.reads += 1;
        if self.fail {
            return Err(io::Error::new(io::ErrorKind::NotFound, "fixture missing"));
        }
        Ok(self.bytes.clone())
    }
}
struct AttachmentGuids {
    calls: usize,
    fail: bool,
}
impl GuidSource for AttachmentGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.calls += 1;
        if self.fail {
            return Err(IdentityTransitionError::Randomness(
                "fixture exhausted".into(),
            ));
        }
        Ok("00000000-0000-4000-8000-000000000001".parse().unwrap())
    }
}
fn fixtures() -> (DocumentAttachments, AttachmentFiles, AttachmentGuids) {
    (
        DocumentAttachments::default(),
        AttachmentFiles {
            modified: Some(SystemTime::UNIX_EPOCH + Duration::from_secs(100)),
            bytes: vec![0, 255],
            fail: false,
            stats: 0,
            reads: 0,
        },
        AttachmentGuids {
            calls: 0,
            fail: false,
        },
    )
}

#[test]
fn linked_freshness_uses_one_observation_and_does_not_modify_capture_time() {
    let (mut attachments, mut files, mut guids) = fixtures();
    let captured = files.modified;
    attachments
        .attach(
            "loads",
            ATTACHMENT_LINKED,
            "/data/loads.csv",
            Some(&mut files),
            &mut guids,
        )
        .unwrap();
    let entry = attachments.get("loads").unwrap();
    assert_eq!(entry.name(), "loads");
    assert_eq!(entry.kind(), ATTACHMENT_LINKED);
    assert_eq!(entry.full_file_name(), "/data/loads.csv");
    assert_eq!(
        entry.contents(),
        &AttachmentContents::Linked {
            last_known: captured
        }
    );
    for (observed, expected) in [
        (100, REFERENCE_UP_TO_DATE),
        (101, REFERENCE_OUT_OF_DATE),
        (99, REFERENCE_UP_TO_DATE),
    ] {
        files.modified = Some(SystemTime::UNIX_EPOCH + Duration::from_secs(observed));
        let count = files.stats;
        let state = entry.probe(Some(&mut files)).unwrap();
        assert_eq!(files.stats, count + 1);
        assert_eq!(state.status, expected);
        assert_eq!(state.resolved_file_name.as_deref(), Some("/data/loads.csv"));
    }
    files.modified = None;
    let missing = entry.probe(Some(&mut files)).unwrap();
    assert_eq!(missing.status, REFERENCE_MISSING);
    assert!(missing.resolved_file_name.is_none());
    assert_eq!(entry.probe(None).unwrap().status, REFERENCE_UNKNOWN);
    assert_eq!(
        entry.contents(),
        &AttachmentContents::Linked {
            last_known: captured
        }
    );
    assert_eq!((files.reads, guids.calls), (0, 0));
}

#[test]
fn embedded_payload_is_owned_and_remains_current_without_the_source() {
    let (mut attachments, mut files, mut guids) = fixtures();
    attachments
        .attach(
            "table",
            ATTACHMENT_EMBEDDED,
            "loads.csv",
            Some(&mut files),
            &mut guids,
        )
        .unwrap();
    files.bytes.clear();
    files.modified = None;
    files.fail = true;
    let entry = attachments.get("table").unwrap();
    let AttachmentContents::Embedded { resource_id, bytes } = entry.contents() else {
        panic!("expected embedded contents");
    };
    assert_eq!(
        resource_id.to_string(),
        "00000000-0000-4000-8000-000000000001"
    );
    assert_eq!(bytes, &[0, 255]);
    assert_eq!(entry.kind(), ATTACHMENT_EMBEDDED);
    assert_eq!(entry.probe(None).unwrap().status, REFERENCE_UP_TO_DATE);
    assert!(
        entry
            .probe(Some(&mut files))
            .unwrap()
            .resolved_file_name
            .is_none()
    );
    assert_eq!((files.stats, files.reads, guids.calls), (0, 1, 1));
    assert!(entry.browser_visible());
    let copied = attachments.clone();
    attachments.set_browser_visible("table", false).unwrap();
    assert!(!attachments.get("table").unwrap().browser_visible());
    assert!(copied.get("table").unwrap().browser_visible());
    assert!(attachments.remove("table"));
    assert!(!attachments.remove("table"));
    assert_eq!(copied.entries().len(), 1);
}

#[test]
fn generic_and_unobserved_links_have_no_freshness_baseline() {
    let (mut attachments, mut files, mut guids) = fixtures();
    attachments
        .attach(
            "generic",
            ATTACHMENT_GENERIC,
            "file",
            Some(&mut files),
            &mut guids,
        )
        .unwrap();
    attachments
        .attach("linked", ATTACHMENT_LINKED, "file", None, &mut guids)
        .unwrap();
    assert_eq!(files.stats, 0);
    assert_eq!(
        attachments
            .entries()
            .iter()
            .map(|entry| entry.name())
            .collect::<Vec<_>>(),
        ["generic", "linked"]
    );
    assert_eq!(
        attachments.get("generic").unwrap().kind(),
        ATTACHMENT_GENERIC
    );
    for entry in attachments.entries() {
        assert_eq!(
            entry.probe(Some(&mut files)).unwrap().status,
            REFERENCE_UP_TO_DATE
        );
    }
    assert!(attachments.remove("generic"));
    assert_eq!(attachments.entries()[0].name(), "linked");
}

#[test]
fn invalid_names_paths_and_kinds_fail_before_external_work() {
    let (mut attachments, mut files, mut guids) = fixtures();
    attachments
        .attach("existing", ATTACHMENT_GENERIC, "file", None, &mut guids)
        .unwrap();
    let before = attachments.clone();
    for (name, path, kind) in [
        ("", "file", ATTACHMENT_LINKED),
        ("name", "", ATTACHMENT_EMBEDDED),
        ("existing", "file", ATTACHMENT_EMBEDDED),
        ("unknown", "file", AttachmentKind(1)),
    ] {
        let error = attachments
            .attach(name, kind, path, Some(&mut files), &mut guids)
            .unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert!(error.source().is_none());
        assert_eq!(attachments, before);
    }
    assert!(attachments.set_browser_visible("missing", false).is_err());
    assert_eq!((files.stats, files.reads, guids.calls), (0, 0, 0));
    assert!(
        attachments
            .attach("embed", ATTACHMENT_EMBEDDED, "file", None, &mut guids)
            .is_err()
    );
    assert_eq!(attachments, before);
}

#[test]
fn read_probe_and_identity_failures_never_publish_partial_attachments() {
    let (mut attachments, mut files, mut guids) = fixtures();
    files.fail = true;
    for kind in [ATTACHMENT_LINKED, ATTACHMENT_EMBEDDED] {
        let error = attachments
            .attach("failed", kind, "file", Some(&mut files), &mut guids)
            .unwrap_err();
        assert!(error.to_string().contains("file"));
        assert!(error.source().unwrap().is::<io::Error>());
        assert!(attachments.entries().is_empty());
    }
    assert_eq!(guids.calls, 0);
    files.fail = false;
    guids.fail = true;
    let error = attachments
        .attach(
            "failed",
            ATTACHMENT_EMBEDDED,
            "file",
            Some(&mut files),
            &mut guids,
        )
        .unwrap_err();
    assert!(error.source().unwrap().is::<IdentityTransitionError>());
    assert!(error.to_string().contains("fixture exhausted"));
    assert!(attachments.entries().is_empty());
    guids.fail = false;
    attachments
        .attach(
            "first",
            ATTACHMENT_EMBEDDED,
            "file",
            Some(&mut files),
            &mut guids,
        )
        .unwrap();
    let before = attachments.clone();
    assert!(
        attachments
            .attach(
                "collision",
                ATTACHMENT_EMBEDDED,
                "file",
                Some(&mut files),
                &mut guids
            )
            .unwrap_err()
            .to_string()
            .contains("fresh identity")
    );
    assert_eq!(attachments, before);
    attachments
        .attach(
            "link",
            ATTACHMENT_LINKED,
            "file",
            Some(&mut files),
            &mut guids,
        )
        .unwrap();
    files.fail = true;
    let error = attachments
        .get("link")
        .unwrap()
        .probe(Some(&mut files))
        .unwrap_err();
    assert_eq!(
        error
            .source()
            .unwrap()
            .downcast_ref::<io::Error>()
            .unwrap()
            .kind(),
        io::ErrorKind::PermissionDenied
    );
}
