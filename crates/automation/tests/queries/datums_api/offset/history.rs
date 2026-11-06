// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn aborted_creation_retires_both_feature_and_scalar_ids() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files
        .commands(document)
        .unwrap()
        .begin("Offset group")
        .unwrap();
    let retired = create_offset(&mut files, &mut source, "width / 2");
    let retired_scalar = scalar_id(&files, document);
    files.commands(document).unwrap().abort().unwrap();
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert!(snapshot.work_geometry.features().is_empty());
    assert!(snapshot.parameters.get(retired_scalar).is_none());
    let kept = create_offset(&mut files, &mut source, "width / 2");
    assert_ne!(kept.ref_, retired.ref_);
    assert_ne!(scalar_id(&files, document), retired_scalar);
}

#[test]
fn grouped_redefine_abort_restores_formula_and_reference() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    let id = scalar_id(&files, document);
    files
        .commands(document)
        .unwrap()
        .begin("Offset edit")
        .unwrap();
    let args = revision(3, "width * 2", WORK_REF_YZ_PLANE);
    redefine(&mut files, &mut source, &args).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_source(&files, document, id, "width / 2");
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 0.5])
    );
}

#[test]
fn vetoed_redefine_keeps_the_last_committed_source_and_geometry() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    files.save(document, "test", &mut source).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    let args = revision(3, "width * 2", WORK_REF_YZ_PLANE);
    assert_datum_decline(redefine(&mut files, &mut source, &args), "locked");
    assert_source(&files, document, scalar_id(&files, document), "width / 2");
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 0.5])
    );
    assert_eq!(owner_revision(&files, document).1, before.1);
}

#[test]
fn copied_offset_keeps_formula_identity_and_current_driver_dependencies() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create_offset(&mut files, &mut source, "width / 2");
    let id = scalar_id(&files, document);
    let path = Path::new("/fixtures/offset-copy.obk");
    files
        .save_copy(document, path, None, "test", &mut source)
        .unwrap();
    let copied = super::super::history::open_copied_datum(&mut files, &mut source, path);
    assert_eq!(scalar_id(&files, copied), id);
    assert_source(&files, copied, id, "width / 2");
    edit_driver(&mut files, copied, "8 cm");
    assert_eq!(plane_rows(&files, true)[3].ref_, created.ref_);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 4.0])
    );
}
