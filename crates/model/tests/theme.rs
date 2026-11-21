// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{self, Rgba, ThemeKind, ThemeToken};
use oblikovati_model::theme::{ThemeDefinition, ThemeLibrary, theme_tokens};
use serde::Deserialize;
use std::collections::BTreeMap;
#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct GoThemeOracle {
    application_revision: String,
    tokens: Vec<ThemeToken>,
    builtins: Vec<GoThemeRecord>,
    library: Vec<GoLibraryRecord>,
}
#[derive(Deserialize)]
struct GoThemeRecord {
    name: String,
    kind: ThemeKind,
    colors: BTreeMap<String, String>,
    channels: BTreeMap<ThemeToken, Rgba>,
}
#[derive(Deserialize)]
struct GoLibraryRecord {
    stage: String,
    active: String,
    revision: u64,
    names: Vec<String>,
    colors: BTreeMap<String, String>,
}
fn oracle() -> GoThemeOracle {
    serde_json::from_str(include_str!("theme/go-baseline.json")).unwrap()
}
#[test]
fn builtins_match_every_go_token_channel_hex_value_and_public_contract() {
    let expected = oracle();
    assert_eq!(
        expected.application_revision,
        "aa8297d1b0ac33c024f19e54821536d76e8a447a"
    );
    assert_eq!(theme_tokens().cloned().collect::<Vec<_>>(), expected.tokens);
    let actual = [
        ThemeDefinition::dark().unwrap(),
        ThemeDefinition::light().unwrap(),
    ];
    for (theme, expected) in actual.iter().zip(expected.builtins) {
        assert_eq!(theme.name(), expected.name);
        assert_eq!(theme.kind(), &expected.kind);
        assert_eq!(theme.palette(), &expected.channels);
        assert_eq!(theme.hex_colors(), expected.colors);
        let contract: &dyn oblikovati_api::contract::Theme = theme;
        assert_eq!(contract.name(), theme.name());
        assert_eq!(contract.kind(), theme.kind().clone());
        assert_eq!(
            contract.color(types::TOKEN_CHROME_TEXT),
            theme.color(&types::TOKEN_CHROME_TEXT)
        );
    }
}
fn assert_library_stage(library: &ThemeLibrary, expected: &GoLibraryRecord) {
    assert_eq!(
        library.active().name(),
        expected.active,
        "{}",
        expected.stage
    );
    assert_eq!(library.revision(), expected.revision, "{}", expected.stage);
    assert_eq!(
        library
            .themes()
            .iter()
            .map(|theme| theme.name().to_string())
            .collect::<Vec<_>>(),
        expected.names
    );
    assert_eq!(
        library.active().hex_colors(),
        expected.colors,
        "{}",
        expected.stage
    );
}
fn direct_tint() -> Rgba {
    Rgba {
        r: 0.1,
        g: 0.2,
        b: 0.3,
        a: 0.4,
    }
}
fn derived_tint() -> Rgba {
    Rgba {
        r: 0.9,
        g: 0.8,
        b: 0.7,
        a: 0.6,
    }
}
#[test]
fn library_selection_custom_edits_and_removal_follow_the_go_trace() {
    let expected = oracle();
    let mut library = ThemeLibrary::new(Vec::new(), "Missing").unwrap();
    assert_library_stage(&library, &expected.library[0]);
    assert!(library.set_active("Light"));
    assert_library_stage(&library, &expected.library[1]);
    assert!(!library.set_active("Light"));
    assert_library_stage(&library, &expected.library[2]);
    assert!(!library.set_active("Missing"));
    assert_library_stage(&library, &expected.library[3]);
    assert!(
        !library
            .edit_active_color(&types::TOKEN_CHROME_ACCENT, direct_tint())
            .unwrap()
    );
    assert_library_stage(&library, &expected.library[4]);
    assert_custom_stages(&mut library, &expected.library);
    assert_removal_stages(&mut library, &expected.library);
}
fn assert_custom_stages(library: &mut ThemeLibrary, stages: &[GoLibraryRecord]) {
    library.duplicate("Light", "Workshop").unwrap();
    assert_library_stage(library, &stages[5]);
    assert!(
        library
            .edit_active_color(&types::TOKEN_CHROME_ACCENT, direct_tint())
            .unwrap()
    );
    assert_library_stage(library, &stages[6]);
    for index in [7, 8] {
        assert!(
            library
                .edit_active_color(&types::TOKEN_CHROME_BUTTON_HOVER, derived_tint())
                .unwrap()
        );
        assert_library_stage(library, &stages[index]);
    }
    library.duplicate("Dark", "Second").unwrap();
    assert_library_stage(library, &stages[9]);
}
fn assert_removal_stages(library: &mut ThemeLibrary, stages: &[GoLibraryRecord]) {
    assert_eq!(library.customs().count(), 2);
    library.remove("Workshop").unwrap();
    assert_library_stage(library, &stages[10]);
    library.remove("Second").unwrap();
    assert_library_stage(library, &stages[11]);
    assert_eq!(library.customs().count(), 0);
}
#[test]
fn cloned_customs_keep_direct_and_derived_xml_edits_independent() {
    let original = ThemeDefinition::dark().unwrap();
    let mut custom = original.duplicate("Workshop").unwrap();
    custom
        .set_color(&types::TOKEN_CHROME_ACCENT, direct_tint())
        .unwrap();
    custom
        .set_color(&types::TOKEN_CHROME_BUTTON_HOVER, derived_tint())
        .unwrap();
    assert_eq!(original.hex_colors(), oracle().builtins[0].colors);
    let encoded = custom.to_xml().unwrap();
    let restored = ThemeDefinition::from_xml(&encoded, "Fallback", types::THEME_CUSTOM).unwrap();
    assert_eq!(restored.name(), "Workshop");
    assert_eq!(restored.hex_colors(), custom.hex_colors());
    assert_eq!(restored.to_xml().unwrap(), encoded);
}
#[test]
fn unknown_xml_attributes_namespaces_comments_and_text_survive_an_edit() {
    let input = include_str!("../assets/themes/dark.xml")
        .replace("<bpy>", "<bpy xmlns:v=\"urn:vendor\" vendor_root=\"kept\">")
        .replace(
            "</bpy>",
            "<v:extra v:setting=\"future\">text<!-- preserved --><![CDATA[<raw>]]></v:extra></bpy>",
        );
    let mut custom =
        ThemeDefinition::from_xml(input.as_bytes(), "Vendor", types::THEME_CUSTOM).unwrap();
    custom
        .set_color(&types::TOKEN_CHROME_TEXT, direct_tint())
        .unwrap();
    let encoded = String::from_utf8(custom.to_xml().unwrap()).unwrap();
    for retained in [
        "vendor_root=\"kept\"",
        "v:setting=\"future\"",
        "xmlns:v=\"urn:vendor\"",
        "<!-- preserved -->",
        "<![CDATA[<raw>]]>",
        ">text",
    ] {
        assert!(
            encoded.contains(retained),
            "missing retained XML {retained:?}"
        );
    }
    let restored =
        ThemeDefinition::from_xml(encoded.as_bytes(), "Fallback", types::THEME_CUSTOM).unwrap();
    assert_eq!(restored.hex_colors(), custom.hex_colors());
}
#[test]
fn malformed_snapshots_keep_mapped_values_and_first_child_mapping_is_preserved() {
    let dark = include_str!("../assets/themes/dark.xml");
    let input = dark.replace(
        "</bpy>",
        "<oblikovati_tokens name=\"\" chrome.text=\"wrong\" ignored=\"#ffffff\"/></bpy>",
    );
    let parsed =
        ThemeDefinition::from_xml(input.as_bytes(), "Fallback", types::THEME_CUSTOM).unwrap();
    assert_eq!(parsed.name(), "Fallback");
    assert_eq!(parsed.hex_colors(), oracle().builtins[0].colors);
    let duplicate = dark.replacen("<Theme>", "<Theme/><Theme>", 1);
    assert!(
        ThemeDefinition::from_xml(duplicate.as_bytes(), "Invalid", types::THEME_CUSTOM)
            .unwrap_err()
            .contains("complete Blender attributes")
    );
}
#[test]
fn invalid_inputs_and_failed_library_edits_leave_the_original_state() {
    let mut library = ThemeLibrary::new(Vec::new(), "Dark").unwrap();
    let before = library.active().to_xml().unwrap();
    for (base, name) in [("Missing", "New"), ("Dark", ""), ("Light", "Dark")] {
        assert!(library.duplicate(base, name).is_err());
    }
    for name in ["Dark", "Missing"] {
        assert!(library.remove(name).is_err());
    }
    assert_eq!(library.revision(), 1);
    assert_eq!(library.active().to_xml().unwrap(), before);
    library.duplicate("Dark", "Custom").unwrap();
    let before = library.active().to_xml().unwrap();
    let invalid = Rgba {
        r: f32::NAN,
        ..direct_tint()
    };
    assert!(
        library
            .edit_active_color(&types::TOKEN_CHROME_TEXT, invalid)
            .unwrap_err()
            .contains("finite")
    );
    assert!(
        library
            .edit_active_color(&ThemeToken("unknown".into()), direct_tint())
            .is_err()
    );
    assert_eq!(library.active().to_xml().unwrap(), before);
    assert_eq!(library.revision(), 2);
}
#[test]
fn loaded_customs_are_unique_owned_snapshots_and_unknown_tokens_are_visible() {
    let custom = ThemeDefinition::light()
        .unwrap()
        .duplicate("Custom")
        .unwrap();
    let library = ThemeLibrary::new(vec![custom.clone()], "Custom").unwrap();
    assert_eq!(library.active().name(), "Custom");
    assert_eq!(library.customs().count(), 1);
    assert!(ThemeLibrary::new(vec![custom.clone(), custom], "Dark").is_err());
    assert!(ThemeLibrary::new(vec![ThemeDefinition::light().unwrap()], "Dark").is_err());
    let fallback = library.active().color(&ThemeToken("unmapped".into()));
    assert_eq!(
        fallback,
        Rgba {
            r: 1.,
            g: 0.,
            b: 1.,
            a: 1.
        }
    );
    let dark = include_bytes!("../assets/themes/dark.xml");
    assert!(ThemeDefinition::from_xml(dark, "", types::THEME_CUSTOM).is_err());
    assert!(ThemeDefinition::from_xml(dark, "Name", ThemeKind("unknown".into())).is_err());
    assert!(ThemeDefinition::from_xml(b"<bpy>", "Name", types::THEME_CUSTOM).is_err());
}
