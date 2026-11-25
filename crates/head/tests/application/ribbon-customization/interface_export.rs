// SPDX-License-Identifier: GPL-2.0-only
//! Validate the applied settings exported from the live native interface.
use super::{ApplicationWorld, export_new_file};
use cucumber::{given, then, when};
use serde_json::Value;
use xml::reader::XmlEvent;

#[given("the current customization has no staged changes")]
fn clean_customization(world: &mut ApplicationWorld) {
    for label in ["Apply", "Cancel"] {
        assert!(world.run_in_window("Customize", label).alpha < u8::MAX);
    }
}

#[when("the user exports the current settings to a new XML file")]
fn export_current_settings(world: &mut ApplicationWorld) {
    export_new_file(world);
}

#[then("the XML file parses as a customization settings document")]
fn xml_is_valid(world: &mut ApplicationWorld) {
    super::exported_xml_valid(world);
}

#[then("it contains ribbon state, custom panel, and quick access toolbar settings")]
fn ribbon_settings_are_present(world: &mut ApplicationWorld) {
    let ribbon = section(world, "ribbon-state");
    assert_eq!(ribbon["mode"], "full");
    assert_eq!(ribbon["placement"], "top");
    assert_eq!(ribbon["show-panel-titles"], true);
    assert!(section_attributes(world, "custom-panels").is_empty());
    let toolbar = section(world, "quick-access-toolbar");
    assert_eq!(toolbar["placement"], "title-bar");
    let context = toolbar["active-context"].as_str().unwrap();
    let shown = toolbar["contexts"][context]["standard"].as_array().unwrap();
    assert!(shown.iter().any(|button| button == "Open"));
    assert!(toolbar["contexts"][context]["added"].is_array());
}

#[then("it contains keyboard mapping settings")]
fn keyboard_settings_are_present(world: &mut ApplicationWorld) {
    let mappings = section(world, "keyboard-mappings");
    assert_eq!(mappings["default-multi-character-aliases"], false);
    assert!(mappings["bindings"].is_object());
}

#[then("it contains marking menu and mini-toolbar settings")]
fn canvas_menu_settings_are_present(world: &mut ApplicationWorld) {
    let marking = section(world, "marking-menu");
    assert_eq!(marking["enabled"], true);
    assert_eq!(marking["classic"], false);
    assert_eq!(marking["style"], "icon-and-text");
    assert_eq!(marking["overflow"], "full");
    let menus = marking["menus"].as_array().unwrap();
    assert_eq!(menus.len(), 3);
    assert!(menus.iter().all(|menu| menu["quadrants"].is_array()));
    assert_eq!(section(world, "mini-toolbar")["visible"], true);
    clean_customization(world);
}

fn section(world: &ApplicationWorld, name: &str) -> Value {
    let attrs = section_attributes(world, name);
    let [state] = attrs.as_slice() else {
        panic!("expected one state attribute in {name}");
    };
    assert_eq!(state.name.local_name, "state");
    serde_json::from_str(&state.value).expect("exported interface settings object")
}

fn section_attributes(
    world: &ApplicationWorld,
    target: &str,
) -> Vec<xml::attribute::OwnedAttribute> {
    let file = std::fs::File::open(world.saved_path.as_ref().unwrap()).unwrap();
    xml::EventReader::new(file)
        .into_iter()
        .find_map(|event| match event.expect("valid XML") {
            XmlEvent::StartElement {
                name, attributes, ..
            } if name.local_name == target => Some(attributes),
            _ => None,
        })
        .unwrap_or_else(|| panic!("expected {target} in the exported file"))
}
