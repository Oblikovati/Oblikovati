// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn legacy_3d_sketch_load_refuses_before_document_registration() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let bytes = String::from_utf8(fixture(1))
        .unwrap()
        .replace("model:\n", "model:\n  sketches3D: [{id: 7}]\n")
        .into_bytes();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), bytes.clone());
    let error = host
        .open(
            &path("a.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(error.to_string().contains("sketches3D"));
    assert!(host.workspace().documents().is_empty());
    assert_eq!(source.0, 1000);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], bytes);
}

#[test]
fn model_merge_load_refuses_before_document_registration() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let bytes = String::from_utf8(fixture(1))
        .unwrap()
        .replace("model:\n", "model:\n  <<: {sketches3D: [{id: 7}]}\n")
        .into_bytes();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), bytes.clone());
    let error = host
        .open(
            &path("a.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(error.to_string().contains("model.<<"));
    assert!(host.workspace().documents().is_empty());
    assert_eq!(source.0, 1000);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], bytes);
}
