// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::application_options::{SketchApplicationOptions, SketchPreferenceAccess};
use oblikovati_persistence::application_options::SketchPreferenceFile;
use std::{cell::RefCell, io, path::Path, rc::Rc};
type Disk = Rc<RefCell<FakeDisk>>;
fn disk(encoded: &[u8]) -> Disk {
    Rc::new(RefCell::new(FakeDisk {
        committed: encoded.to_vec(),
        ..Default::default()
    }))
}
fn load(disk: &Disk) -> io::Result<SketchPreferenceFile<FakeAtomicFiles>> {
    SketchPreferenceFile::load(Path::new("options.yaml"), FakeAtomicFiles(disk.clone()))
}
#[test]
fn missing_and_empty_files_use_go_defaults_including_the_origin_anchor() {
    for missing in [false, true] {
        let disk = disk(b"");
        if missing {
            disk.borrow_mut().read_error = Some(io::ErrorKind::NotFound);
        }
        let owner = load(&disk).unwrap();
        assert_eq!(owner.sketch_options(), &SketchApplicationOptions::default());
        assert!(owner.sketch_options().auto_project_origin);
        assert!(owner.sketch_options().pointer_input_cartesian);
        assert!(!owner.sketch_options().dimension_input_cartesian);
    }
}
#[test]
fn legacy_partial_group_keeps_defaults_but_explicit_false_survives_reload() {
    let disk =
        disk(b"sketch: {gridSpacingCm: 2, headsUpDisplay: false}\ngeneral: {startupAction: 9}\n");
    let mut owner = load(&disk).unwrap();
    assert!(owner.sketch_options().auto_project_origin);
    assert!(!owner.sketch_options().heads_up_display);
    let mut options = owner.sketch_options().clone();
    options.auto_project_origin = false;
    owner.set_sketch_options(options).unwrap();
    let reopened = load(&disk).unwrap();
    assert!(!reopened.sketch_options().auto_project_origin);
    assert!(!reopened.sketch_options().heads_up_display);
    assert_eq!(reopened.sketch_options().grid_spacing_cm, 2.0);
    assert!(
        String::from_utf8(disk.borrow().committed.clone())
            .unwrap()
            .contains("startupAction: 9")
    );
}
#[test]
fn sketch_save_preserves_the_full_other_group_values() {
    let encoded = b"sketch: {}\nfuture: {nested: [1, true, {value: text}]}\nui: {fontScale: 1.5}\n";
    let disk = disk(encoded);
    let before: serde_yaml_ng::Value = serde_yaml_ng::from_slice(encoded).unwrap();
    let mut owner = load(&disk).unwrap();
    owner
        .save_options(SketchApplicationOptions {
            suppress_format_overrides: true,
            ..Default::default()
        })
        .unwrap();
    let saved: serde_yaml_ng::Value = serde_yaml_ng::from_slice(&disk.borrow().committed).unwrap();
    assert_eq!(saved["future"], before["future"]);
    assert_eq!(saved["ui"], before["ui"]);
    assert_saved_format_override(&disk);
}
#[test]
fn malformed_unknown_and_invalid_sketch_content_declines_before_overwrite() {
    for encoded in [
        "sketch: [",
        "sketch: {futureFlag: true}",
        "sketch: {gridSpacingCm: 0}",
        "sketch: {gridSpacingCm: .nan}",
        "sketch: {gridSpacingCm: .inf}",
        "sketch: {gridSpacingCm: -1}",
    ] {
        let disk = disk(encoded.as_bytes());
        let error = load(&disk).err().unwrap();
        assert!(error.to_string().contains("options.yaml"));
        assert!(error.to_string().contains("expected"));
        assert_eq!(disk.borrow().committed, encoded.as_bytes());
        assert_eq!(disk.borrow().calls, ["read"]);
    }
}
#[test]
fn read_errors_other_than_absence_do_not_reset_the_application_preferences() {
    for kind in [io::ErrorKind::PermissionDenied, io::ErrorKind::Interrupted] {
        let disk = disk(b"sketch: {autoProjectOrigin: false}\n");
        disk.borrow_mut().read_error = Some(kind);
        assert!(
            load(&disk)
                .err()
                .unwrap()
                .to_string()
                .contains("injected read decline")
        );
        assert_eq!(disk.borrow().calls, ["read"]);
    }
}
#[test]
fn failed_atomic_phase_retains_the_old_file_and_current_options() {
    for phase in ["stage", "write", "sync", "commit"] {
        let disk = disk(b"sketch: {autoProjectOrigin: true}\n");
        let before = disk.borrow().committed.clone();
        let mut owner = load(&disk).unwrap();
        disk.borrow_mut().fail_at = Some(phase);
        let options = SketchApplicationOptions {
            auto_project_origin: false,
            ..Default::default()
        };
        let error = owner.save_options(options).unwrap_err();
        assert!(error.to_string().contains(phase));
        assert!(owner.sketch_options().auto_project_origin);
        assert_eq!(disk.borrow().committed, before);
    }
}
#[test]
fn invalid_grid_writes_fail_before_any_staging_or_state_change() {
    let disk = disk(b"sketch: {}\n");
    let mut owner = load(&disk).unwrap();
    for spacing in [0.0, -1.0, f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let options = SketchApplicationOptions {
            grid_spacing_cm: spacing,
            ..Default::default()
        };
        let error = owner.set_sketch_options(options).unwrap_err();
        assert!(error.contains("positive finite"));
        assert_eq!(owner.sketch_options(), &SketchApplicationOptions::default());
    }
    assert_eq!(disk.borrow().calls, ["read"]);
}

fn assert_saved_format_override(disk: &Disk) {
    assert!(
        load(disk)
            .unwrap()
            .sketch_options()
            .suppress_format_overrides
    );
}
