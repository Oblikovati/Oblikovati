// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_api::wire::KeymapExport;
use oblikovati_model::keymap::KeymapPreferenceAccess;
use oblikovati_persistence::keymap_preferences::KeymapPreferenceFile;
use std::{cell::RefCell, collections::BTreeMap, path::Path, rc::Rc};

#[test]
fn saved_delta_reopens_and_failed_write_keeps_loaded_state() {
    let disk = Rc::new(RefCell::new(FakeDisk::default()));
    let load = || {
        KeymapPreferenceFile::load(Path::new("keymap.yaml"), FakeAtomicFiles(disk.clone())).unwrap()
    };
    let mut store = load();
    let delta = KeymapExport {
        chords: Some(BTreeMap::from([("edit.undo".into(), "Ctrl+U".into())])),
        aliases: None,
    };
    store.set_keymap(&delta).unwrap();
    assert_eq!(load().keymap(), delta);
    disk.borrow_mut().fail_at = Some("stage");
    assert!(store.set_keymap(&KeymapExport::default()).is_err());
    assert_eq!(store.keymap(), delta);
    assert_eq!(load().keymap(), delta);
}
