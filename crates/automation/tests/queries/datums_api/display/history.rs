// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn display_settings_survive_reopen_and_copy_with_the_same_key() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = fixed(&mut files, &mut source, false, true);
    redefine(&mut files, &mut source, &display_args(3)).unwrap();
    files.save(document, "test", &mut source).unwrap();
    let document =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_explicit(&plane_rows(&files, true)[3]);
    let path = Path::new("/fixtures/display-copy.obk");
    files
        .save_copy(document, path, None, "test", &mut source)
        .unwrap();
    super::super::history::open_copied_datum(&mut files, &mut source, path);
    assert_explicit(&plane_rows(&files, true)[3]);
    assert_eq!(plane_rows(&files, true)[3].ref_, created.ref_);
}

#[test]
fn aborted_settings_keep_previous_display_source() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    files
        .commands(document)
        .unwrap()
        .begin("Display group")
        .unwrap();
    redefine(&mut files, &mut source, &display_args(3)).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert!(!plane_rows(&files, true)[3].grounded);
}

#[test]
fn vetoed_settings_keep_previous_display_source() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    files.save(document, "test", &mut source).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    assert_datum_decline(
        redefine(&mut files, &mut source, &display_args(3)),
        "locked",
    );
    assert_eq!(owner_revision(&files, document).1, before.1);
    assert!(!plane_rows(&files, true)[3].grounded);
}
