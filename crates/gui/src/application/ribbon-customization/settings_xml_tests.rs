// SPDX-License-Identifier: GPL-2.0-only
//! Compatibility and round-trip checks for ribbon customization XML.
use super::*;
#[test]
fn round_trip_preserves_tabs_commands_and_options() {
    let commands = RibbonSettings::from([(
        RibbonTab::Tools,
        vec![
            Entry {
                command: Command::Sketches,
                large: true,
                text: false,
                expanded: true,
            },
            Entry {
                command: Command::Separator,
                large: false,
                text: false,
                expanded: false,
            },
            Entry {
                command: Command::Zoom(
                    crate::application::viewport_orientation::ZoomAction::Window,
                ),
                large: false,
                text: true,
                expanded: false,
            },
            Entry {
                command: Command::ZParallel,
                large: false,
                text: true,
                expanded: false,
            },
            Entry {
                command: Command::Pdf3D,
                large: false,
                text: true,
                expanded: false,
            },
            Entry {
                command: Command::ExportPdf3D,
                large: true,
                text: false,
                expanded: true,
            },
        ],
    )]);
    let settings = SettingsDocument {
        commands,
        expanded_builtins: BTreeSet::from(["part.tools.clipboard.copy".into()]),
        ungrouped_builtins: BTreeSet::from(["part.model.work-features.plane".into()]),
        large_builtins: BTreeSet::from([crate::shell_ribbon::ID.into()]),
        appearance: RibbonAppearance::Compact,
    };
    assert_eq!(decode(&encode(&settings).unwrap()).unwrap(), settings);
}
#[test]
fn every_command_has_a_distinct_stable_identifier() {
    let ids: std::collections::BTreeSet<_> = COMMANDS.iter().map(|item| item.id()).collect();
    assert_eq!(ids.len(), COMMANDS.len());
}
#[test]
fn version_one_commands_remain_in_the_main_panel() {
    let xml = b"<ribbon-customization version=\"1\"><tab id=\"tools\"><command id=\"sketch-visibility\" large=\"true\" text=\"false\"/></tab></ribbon-customization>";
    let settings = decode(xml).unwrap();
    assert!(!settings.commands[&RibbonTab::Tools][0].expanded);
    assert!(settings.commands[&RibbonTab::Tools][0].large);
    assert_eq!(decode(&encode(&settings).unwrap()).unwrap(), settings);
}
#[test]
fn version_two_and_invalid_builtin_locations_are_checked() {
    let old = b"<ribbon-customization version=\"2\"><tab id=\"tools\"><command id=\"measure\" large=\"false\" text=\"true\" expanded=\"true\"/></tab></ribbon-customization>";
    assert!(decode(old).unwrap().commands[&RibbonTab::Tools][0].expanded);
    for xml in [
        "<ribbon-customization version=\"3\"><expanded-builtin id=\"bad id\"/></ribbon-customization>",
        "<ribbon-customization version=\"3\"><expanded-builtin id=\"part.tools.clipboard.copy\"/><expanded-builtin id=\"part.tools.clipboard.copy\"/></ribbon-customization>",
        "<ribbon-customization version=\"2\"><expanded-builtin id=\"part.tools.clipboard.copy\"/></ribbon-customization>",
    ] {
        assert!(decode(xml.as_bytes()).is_err(), "{xml}");
    }
}
#[test]
fn old_settings_default_to_grouped_and_invalid_group_entries_are_rejected() {
    let old = b"<ribbon-customization version=\"4\" appearance=\"normal\"/>";
    assert!(decode(old).unwrap().ungrouped_builtins.is_empty());
    for xml in [
        "<ribbon-customization version=\"4\" appearance=\"normal\"><ungrouped-builtin id=\"part.model.work-features.plane\"/></ribbon-customization>",
        "<ribbon-customization version=\"5\" appearance=\"normal\"><ungrouped-builtin id=\"bad id\"/></ribbon-customization>",
        "<ribbon-customization version=\"5\" appearance=\"normal\"><ungrouped-builtin id=\"part.model.work-features.plane\"/><ungrouped-builtin id=\"part.model.work-features.plane\"/></ribbon-customization>",
    ] {
        assert!(decode(xml.as_bytes()).is_err(), "{xml}");
    }
}
#[test]
fn legacy_settings_default_to_normal_and_invalid_appearances_fail() {
    for version in ["1", "2", "3"] {
        let xml = format!("<ribbon-customization version=\"{version}\"/>");
        assert_eq!(
            decode(xml.as_bytes()).unwrap().appearance,
            RibbonAppearance::Normal
        );
    }
    for xml in [
        "<ribbon-customization version=\"4\"/>",
        "<ribbon-customization version=\"4\" appearance=\"unknown\"/>",
        "<ribbon-customization version=\"3\" appearance=\"normal\"/>",
    ] {
        assert!(decode(xml.as_bytes()).is_err(), "{xml}");
    }
}
#[test]
fn supported_versions_load_and_unknown_version_reports_the_expected_set() {
    for version in 1..=8 {
        let appearance = if version >= 4 {
            " appearance=\"normal\""
        } else {
            ""
        };
        let xml = format!("<ribbon-customization version=\"{version}\"{appearance}/>");
        assert!(
            decode(xml.as_bytes()).unwrap().large_builtins.is_empty(),
            "version {version} should default to standard button sizes"
        );
    }
    assert_eq!(
        decode(b"<ribbon-customization version=\"99\"/>").unwrap_err(),
        "ribbon version Some(\"99\"): expected 1 through 8"
    );
}

#[test]
fn exported_interface_keeps_ribbon_data_importable() {
    let mut settings = SettingsDocument::default();
    settings.commands.insert(
        RibbonTab::Tools,
        vec![Entry {
            command: Command::Sketches,
            large: true,
            text: false,
            expanded: false,
        }],
    );
    let interface = std::array::from_fn(|_| serde_json::json!({"example": "A & B"}));
    let xml = String::from_utf8(encode_with_interface(&settings, &interface).unwrap()).unwrap();
    assert!(xml.contains("version=\"8\"") && xml.contains("<custom-panels>"));
    assert_eq!(decode(xml.as_bytes()).unwrap(), settings);
    let unrecognized = xml.replacen("<mini-toolbar", "<unknown-tool", 1);
    assert!(decode(unrecognized.as_bytes()).is_err());
    let duplicate = xml.replacen("<mini-toolbar", "<marking-menu", 1);
    assert!(decode(duplicate.as_bytes()).is_err());
}
#[test]
fn assembly_design_has_a_separate_versioned_target() {
    let entry = Entry {
        command: Command::Sketches,
        large: false,
        text: true,
        expanded: false,
    };
    let settings = SettingsDocument {
        commands: BTreeMap::from([
            (RibbonTab::Tools, vec![entry.clone()]),
            (RibbonTab::Assemble, vec![entry]),
        ]),
        ..Default::default()
    };
    let xml = String::from_utf8(encode(&settings).unwrap()).unwrap();
    assert!(xml.contains("version=\"7\""));
    assert!(xml.contains("id=\"assembly-design\""));
    assert_eq!(decode(xml.as_bytes()).unwrap(), settings);
    let mislabeled = xml.replacen("version=\"7\"", "version=\"6\"", 1);
    assert!(decode(mislabeled.as_bytes()).is_err());
}
#[test]
fn rejects_invalid_versions_duplicate_commands_and_dtd() {
    for text in [
        "<ribbon-customization version=\"99\"/>",
        "<!DOCTYPE x><ribbon-customization version=\"1\"/>",
        "<ribbon-customization version=\"1\"><tab id=\"tools\"><command id=\"sketch-visibility\" large=\"false\" text=\"false\"/><command id=\"sketch-visibility\" large=\"false\" text=\"false\"/></tab></ribbon-customization>",
    ] {
        assert!(decode(text.as_bytes()).is_err());
    }
}
