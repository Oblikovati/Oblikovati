// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/fake_atomic_files.rs"]
mod fake_files;
use fake_files::{FakeAtomicFiles, FakeDisk};
use oblikovati_model::{
    application_options::{SketchApplicationOptions, SketchPreferenceAccess},
    marking_menu::{MarkingMenuPreferenceAccess, MarkingMenus},
    orientation_preferences::OrientationPreferenceAccess,
    theme::ThemeDefinition,
};
use oblikovati_persistence::{
    application_options::ApplicationPreferenceFile,
    marking_menu_preferences::MarkingMenuPreferenceFile,
    orientation_preferences::OrientationPreferenceFile, theme_preferences::ThemePreferenceFile,
};
use std::{cell::RefCell, path::Path, rc::Rc};

type Disk = Rc<RefCell<FakeDisk>>;
fn disk(source: &[u8]) -> Disk {
    Rc::new(RefCell::new(FakeDisk {
        committed: source.to_vec(),
        ..Default::default()
    }))
}
fn unchanged(disk: &Disk, source: &[u8]) {
    assert_eq!(disk.borrow().committed, source);
    assert!(!disk.borrow().calls.contains(&"stage"));
}

#[test]
fn application_options_refuse_ambiguous_retained_keys() {
    let source = b"sketch: {}\nfuture: {'<<': {hidden: true}}\n";
    let disk = disk(source);
    let mut file =
        ApplicationPreferenceFile::load(Path::new("options.yaml"), FakeAtomicFiles(disk.clone()))
            .unwrap();
    let before = file.sketch_options().clone();
    let mut changed = before.clone();
    changed.auto_project_origin = !before.auto_project_origin;
    assert!(file.save_options(changed).is_err());
    assert_eq!(file.sketch_options(), &before);
    unchanged(&disk, source);
}
#[test]
fn marking_menu_refuses_ambiguous_retained_keys() {
    let source = b"menus: []\nfuture: {'<<': {hidden: true}}\n";
    let disk = disk(source);
    let mut file =
        MarkingMenuPreferenceFile::load(Path::new("menu.yaml"), FakeAtomicFiles(disk.clone()))
            .unwrap();
    let before = file.menus();
    assert!(file.save(&MarkingMenus::default()).is_err());
    assert_eq!(file.menus(), before);
    unchanged(&disk, source);
}
#[test]
fn theme_refuses_ambiguous_retained_keys() {
    let source = b"activeTheme: Light\nfuture: {'<<': {hidden: true}}\n";
    let disk = disk(source);
    let mut file =
        ThemePreferenceFile::load(Path::new("theme.yaml"), FakeAtomicFiles(disk.clone())).unwrap();
    assert!(file.save_active(&ThemeDefinition::dark().unwrap()).is_err());
    assert_eq!(file.active_name(), "Light");
    unchanged(&disk, source);
}
#[test]
fn orientation_refuses_ambiguous_retained_keys() {
    let source = b"lockToSelection: false\nfuture: {'<<': {hidden: true}}\n";
    let disk = disk(source);
    let mut file = OrientationPreferenceFile::load(
        Path::new("orientation.yaml"),
        FakeAtomicFiles(disk.clone()),
    )
    .unwrap();
    assert!(file.set_orientation_lock(true).is_err());
    assert!(!file.orientation_lock());
    unchanged(&disk, source);
}

#[test]
fn application_ui_extension_is_checked_before_staging() {
    let source = b"sketch: {}\nui: {fontScale: 1, future: {'<<': {hidden: true}}}\n";
    let disk = disk(source);
    let mut file =
        ApplicationPreferenceFile::load(Path::new("options.yaml"), FakeAtomicFiles(disk.clone()))
            .unwrap();
    assert!(
        file.save_options(SketchApplicationOptions::default())
            .is_err()
    );
    unchanged(&disk, source);
}

#[test]
fn literal_merge_text_as_a_value_is_safe() {
    let source = b"sketch: {}\nfuture: {description: '<<'}\n";
    let disk = disk(source);
    let mut file =
        ApplicationPreferenceFile::load(Path::new("options.yaml"), FakeAtomicFiles(disk.clone()))
            .unwrap();
    file.save_options(SketchApplicationOptions::default())
        .unwrap();
    let value: serde_yaml_ng::Value = serde_yaml_ng::from_slice(&disk.borrow().committed).unwrap();
    assert_eq!(value["future"]["description"].as_str(), Some("<<"));
}

#[test]
fn keymap_dynamic_merge_key_is_refused_before_staging() {
    use oblikovati_api::wire::KeymapExport;
    use oblikovati_model::keymap::KeymapPreferenceAccess;
    use oblikovati_persistence::keymap_preferences::KeymapPreferenceFile;
    use std::collections::BTreeMap;
    let source = b"chords: {}\n";
    let disk = disk(source);
    let mut file =
        KeymapPreferenceFile::load(Path::new("keys.yaml"), FakeAtomicFiles(disk.clone())).unwrap();
    let delta = KeymapExport {
        chords: Some(BTreeMap::from([("<<".into(), "Ctrl+A".into())])),
        ..Default::default()
    };
    assert!(file.set_keymap(&delta).is_err());
    unchanged(&disk, source);
}

#[test]
fn application_refuses_all_ambiguous_key_spellings() {
    for field in [
        "{'<<': {x: 1}}",
        "{!!str <<: {x: 1}}",
        "{<<: {x: 1}}",
        "{!!merge '<<': {x: 1}}",
        "{!custom '<<': {x: 1}}",
        "[{nested: {'<<': {x: 1}}}]",
    ] {
        let source = format!("sketch: {{}}\nfuture: {field}\n");
        let disk = disk(source.as_bytes());
        let loaded = ApplicationPreferenceFile::load(
            Path::new("options.yaml"),
            FakeAtomicFiles(disk.clone()),
        );
        match loaded {
            Ok(mut file) => {
                let error = file
                    .save_options(SketchApplicationOptions::default())
                    .unwrap_err();
                assert!(error.to_string().contains("<<"), "{field}: {error}");
            }
            Err(error) => assert!(error.to_string().contains("expected"), "{field}: {error}"),
        }
        unchanged(&disk, source.as_bytes());
    }
}

#[test]
fn project_configuration_refuses_unknown_merge_keys() {
    use oblikovati_persistence::project_file::ProjectConfiguration;
    for source in [
        "schemaVersion: 1\n'<<': {name: hidden}\n",
        "schemaVersion: 1\n<<: {name: hidden}\n",
    ] {
        assert!(
            ProjectConfiguration::decode(source.as_bytes()).is_err(),
            "{source}"
        );
    }
    let valid = ProjectConfiguration::decode(b"schemaVersion: 1\nname: '<<'\n").unwrap();
    assert_eq!(
        ProjectConfiguration::decode(&valid.encode().unwrap()).unwrap(),
        valid
    );
}
