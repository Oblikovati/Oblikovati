// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::{
    application_options::{SketchApplicationOptions, SketchPreferenceAccess},
    ui_scale::{UiScaleOptions, UiScalePreferenceAccess},
};
use oblikovati_persistence::application_options::ApplicationPreferenceFile;
use std::{cell::RefCell, io, path::Path, rc::Rc};
type OptionsDisk = Rc<RefCell<FakeDisk>>;
fn options_disk(encoded: &str) -> OptionsDisk {
    Rc::new(RefCell::new(FakeDisk {
        committed: encoded.as_bytes().to_vec(),
        ..Default::default()
    }))
}
fn load(disk: &OptionsDisk) -> io::Result<ApplicationPreferenceFile<FakeAtomicFiles>> {
    ApplicationPreferenceFile::load(Path::new("options.yaml"), FakeAtomicFiles(disk.clone()))
}
#[test]
fn legacy_defaults_and_independent_go_ui_values_load() {
    for encoded in ["", "ui: {}", "ui: {fontScale: 0, iconScale: -1}"] {
        assert_eq!(
            load(&options_disk(encoded)).unwrap().ui_scale(),
            UiScaleOptions::default()
        );
    }
    let options = load(&options_disk("ui: {fontScale: 1.5, iconScale: 2.5}"))
        .unwrap()
        .ui_scale();
    assert_eq!((options.font(), options.icon()), (1.5, 2.5));
}
#[test]
fn scale_save_reopens_and_retains_unknown_ui_and_other_groups() {
    let disk = options_disk("ui: {showMenuBar: false, future: [1, true]}\nfuture: {value: text}\n");
    let mut owner = load(&disk).unwrap();
    let options = UiScaleOptions::new(2., 1.5).unwrap();
    owner.set_ui_scale(options).unwrap();
    assert_eq!(owner.ui_scale(), options);
    assert_eq!(load(&disk).unwrap().ui_scale(), options);
    let saved: serde_yaml_ng::Value = serde_yaml_ng::from_slice(&disk.borrow().committed).unwrap();
    assert_eq!(saved["ui"]["showMenuBar"], false);
    assert_eq!(
        saved["ui"]["future"],
        serde_yaml_ng::from_str::<serde_yaml_ng::Value>("[1, true]").unwrap()
    );
    assert_eq!(saved["future"]["value"], "text");
}
#[test]
fn separate_group_owners_retain_each_others_latest_saved_values() {
    let disk = options_disk("ui: {showStatusBar: false}\nsketch: {}\n");
    let mut first = load(&disk).unwrap();
    let mut second = load(&disk).unwrap();
    let options = UiScaleOptions::new(1.5, 2.).unwrap();
    first.save_ui_scale(options).unwrap();
    second
        .save_options(SketchApplicationOptions {
            auto_project_origin: false,
            ..Default::default()
        })
        .unwrap();
    first
        .save_ui_scale(UiScaleOptions::new(2., 2.).unwrap())
        .unwrap();
    let reopened = load(&disk).unwrap();
    assert!(!reopened.sketch_options().auto_project_origin);
    assert_eq!(reopened.ui_scale().font(), 2.);
    assert_eq!(second.ui_scale(), options);
}
#[test]
fn every_atomic_failure_keeps_published_scale_and_bytes() {
    for phase in ["read", "stage", "write", "sync", "commit"] {
        let disk = options_disk("ui: {fontScale: 1.25, iconScale: 1.5}\n");
        let mut owner = load(&disk).unwrap();
        let before = disk.borrow().committed.clone();
        let options = owner.ui_scale();
        disk.borrow_mut().fail_at = Some(phase);
        let error = owner
            .save_ui_scale(UiScaleOptions::new(2., 2.).unwrap())
            .unwrap_err();
        assert!(error.to_string().contains(phase));
        assert_eq!(owner.ui_scale(), options);
        assert_eq!(disk.borrow().committed, before);
    }
}
#[test]
fn malformed_or_nonfinite_ui_values_decline_without_staging() {
    for encoded in [
        "ui: [1, 2]",
        "ui: {fontScale: .nan}",
        "ui: {iconScale: .inf}",
        "ui: {fontScale: -.inf}",
        "ui: {fontScale: large}",
    ] {
        let disk = options_disk(encoded);
        let error = load(&disk).err().unwrap();
        assert!(error.to_string().contains("options.yaml"));
        assert!(error.to_string().contains("expected"));
        assert_eq!(disk.borrow().calls, ["read"]);
    }
}
#[test]
fn corrupt_external_edit_is_refused_without_resetting_in_memory_scale() {
    let disk = options_disk("ui: {fontScale: 1.5}\n");
    let mut owner = load(&disk).unwrap();
    disk.borrow_mut().committed = b"ui: [".to_vec();
    assert!(owner.set_ui_scale(UiScaleOptions::default()).is_err());
    assert_eq!(owner.ui_scale().font(), 1.5);
    assert_eq!(disk.borrow().committed, b"ui: [");
    assert_eq!(disk.borrow().calls, ["read", "read"]);
}
