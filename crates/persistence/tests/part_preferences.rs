// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::application_options::{PartApplicationOptions, PartPreferenceAccess};
use oblikovati_persistence::application_options::ApplicationPreferenceFile;
use std::{cell::RefCell, path::Path, rc::Rc};
type PartDisk = Rc<RefCell<FakeDisk>>;
fn disk(source: &str) -> PartDisk {
    Rc::new(RefCell::new(FakeDisk {
        committed: source.as_bytes().to_vec(),
        ..Default::default()
    }))
}
fn load(disk: &PartDisk) -> ApplicationPreferenceFile<FakeAtomicFiles> {
    ApplicationPreferenceFile::load(Path::new("options.yaml"), FakeAtomicFiles(disk.clone()))
        .unwrap()
}
#[test]
fn missing_part_option_defaults_on_and_explicit_values_survive_reopen() {
    for source in ["", "part: {}", "part: {lookAtSketchPlane: null}"] {
        assert!(load(&disk(source)).part_options().look_at_sketch_plane);
    }
    let disk = disk("part: {futureSetting: [1, true]}\nsketch: {gridVisible: false}\n");
    let mut owner = load(&disk);
    for enabled in [false, true] {
        owner
            .set_part_options(PartApplicationOptions {
                look_at_sketch_plane: enabled,
            })
            .unwrap();
        assert_eq!(load(&disk).part_options().look_at_sketch_plane, enabled);
        let saved: serde_yaml_ng::Value =
            serde_yaml_ng::from_slice(&disk.borrow().committed).unwrap();
        assert_eq!(saved["part"]["futureSetting"][0], 1);
        assert_eq!(saved["sketch"]["gridVisible"], false);
    }
}
#[test]
fn failed_part_save_keeps_the_published_value_and_file() {
    for phase in ["read", "stage", "write", "sync", "commit"] {
        let disk = disk("part: {lookAtSketchPlane: true, futureSetting: 17}\n");
        let mut owner = load(&disk);
        let before = disk.borrow().committed.clone();
        disk.borrow_mut().fail_at = Some(phase);
        assert!(
            owner
                .set_part_options(PartApplicationOptions {
                    look_at_sketch_plane: false
                })
                .is_err()
        );
        assert!(owner.part_options().look_at_sketch_plane);
        assert_eq!(disk.borrow().committed, before);
    }
}
#[test]
fn malformed_part_option_is_refused_before_writing() {
    for source in ["part: []", "part: {lookAtSketchPlane: invalid}"] {
        let disk = disk(source);
        let result = ApplicationPreferenceFile::load(
            Path::new("options.yaml"),
            FakeAtomicFiles(disk.clone()),
        );
        assert!(
            result.is_err(),
            "{source}: expected invalid Part preferences"
        );
        assert_eq!(disk.borrow().committed, source.as_bytes());
    }
}
#[test]
fn part_save_preserves_other_groups_updated_by_another_owner() {
    use oblikovati_model::application_options::{SketchApplicationOptions, SketchPreferenceAccess};
    let disk = disk("part: {futureSetting: 3}\nsketch: {}\n");
    let mut part_owner = load(&disk);
    let mut sketch_owner = load(&disk);
    sketch_owner
        .set_sketch_options(SketchApplicationOptions {
            grid_visible: false,
            ..Default::default()
        })
        .unwrap();
    part_owner
        .set_part_options(PartApplicationOptions {
            look_at_sketch_plane: false,
        })
        .unwrap();
    let reopened = load(&disk);
    assert!(!reopened.part_options().look_at_sketch_plane);
    assert!(!reopened.sketch_options().grid_visible);
    let saved: serde_yaml_ng::Value = serde_yaml_ng::from_slice(&disk.borrow().committed).unwrap();
    assert_eq!(saved["part"]["futureSetting"], 3);
}
