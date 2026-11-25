// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_api::{
    types,
    wire::{MarkingMenuItem, MarkingMenuView},
};
use oblikovati_model::marking_menu::{MarkingMenuPreferenceAccess, MarkingMenus};
use oblikovati_persistence::marking_menu_preferences::MarkingMenuPreferenceFile;
use std::{cell::RefCell, path::Path, rc::Rc};

#[test]
fn go_yaml_classic_and_unknown_fields_survive_atomic_save_and_reload() {
    let disk = Rc::new(RefCell::new(FakeDisk {
        committed: b"classic: true\nfuture: retained\nmenus: []\n".to_vec(),
        ..Default::default()
    }));
    let load = || {
        MarkingMenuPreferenceFile::load(
            Path::new("markingmenu.yaml"),
            FakeAtomicFiles(disk.clone()),
        )
        .unwrap()
    };
    let mut store = load();
    assert!(store.classic());
    let mut menus = MarkingMenus::default();
    let view = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: Some(vec![MarkingMenuItem {
            quadrant: types::QUADRANT_NORTH,
            command_id: "Sketch.Rectangle".into(),
        }]),
        overflow: None,
    };
    menus.set(view.clone()).unwrap();
    store.save(&menus).unwrap();
    assert!(
        String::from_utf8(disk.borrow().committed.clone())
            .unwrap()
            .contains("future: retained")
    );
    let reopened = load();
    assert!(reopened.classic());
    assert_eq!(
        reopened
            .menus()
            .into_iter()
            .find(|menu| menu.environment == types::SKETCH_ENVIRONMENT)
            .unwrap()
            .quadrants,
        view.quadrants
    );
    disk.borrow_mut().fail_at = Some("stage");
    assert!(
        store
            .save_with_classic(&MarkingMenus::default(), false)
            .is_err()
    );
    assert!(store.classic());
    let after_failed_save = load();
    assert!(after_failed_save.classic());
    assert_eq!(after_failed_save.menus(), reopened.menus());

    disk.borrow_mut().fail_at = None;
    store.save_with_classic(&menus, false).unwrap();
    let after_successful_save = load();
    assert!(!after_successful_save.classic());
    assert_eq!(after_successful_save.menus(), reopened.menus());
}

#[test]
fn duplicate_slots_and_environment_are_rejected_on_load() {
    for bytes in [
        "menus: [{environment: 1, quadrants: [{quadrant: 1, command: x}, {quadrant: 1, command: y}]}]",
        "menus: [{environment: 1}, {environment: 1}]",
    ] {
        let disk = Rc::new(RefCell::new(FakeDisk {
            committed: bytes.as_bytes().to_vec(),
            ..Default::default()
        }));
        assert!(
            MarkingMenuPreferenceFile::load(Path::new("markingmenu.yaml"), FakeAtomicFiles(disk))
                .is_err()
        );
    }
}
