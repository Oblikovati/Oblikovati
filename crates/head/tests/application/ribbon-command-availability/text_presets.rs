// SPDX-License-Identifier: GPL-2.0-only
//! Drive the published preset checklist through the native Format Text window.
use super::*;

#[given("Format Text shows No Preset")]
fn start_text_without_a_preset(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, [900, 580]);
    type_content(world, b'a');
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    assert_eq!(world.state().sketch_text.unwrap().content, "ab");
    world.run_in_window("Format Text", "NoPreset");
}

#[when("the user opens the preset list")]
fn open_preset_list(world: &mut ApplicationWorld) {
    click_in_text_editor(world, "No Preset");
}

#[then("No Preset and Last Used are offered")]
fn default_choices_are_offered(world: &mut ApplicationWorld) {
    let last = world.run("Last Used");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "NoPreset" && run.list == last.list)
    );
}

#[when("the user dismisses it, click plus and press Escape")]
fn discard_inline_preset_name(world: &mut ApplicationWorld) {
    click_in_text_editor(world, "Content");
    open_preset_name(world);
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
}

fn open_preset_name(world: &mut ApplicationWorld) {
    click_in_text_editor(world, "+");
    let label = world.run_in_window("Format Text", "Presetname");
    assert!(
        world.frame.runs.iter().any(|run| {
            run.text == "ab"
                && run.list == label.list
                && (run.center()[1] - label.center()[1]).abs() < 3.
        }),
        "expected proposed name ab in the inline field"
    );
}

fn click_in_text_editor(world: &mut ApplicationWorld, label: &str) {
    let name = label.split_whitespace().collect::<String>();
    let center = world.run_in_window("Format Text", &name).center();
    click_preset_ui(world, center);
}

fn click_preset_ui(world: &mut ApplicationWorld, position: [f32; 2]) {
    x11_input::click(diagnostics::pointer_pixels(position));
    world.pump(Vec::new());
}

#[then("proposed inline name ab is canceled and No Preset returns")]
fn name_was_discarded(world: &mut ApplicationWorld) {
    world.expect_no_text("Preset name");
    world.run_in_window("Format Text", "NoPreset");
    assert_eq!(world.state().sketch_text.unwrap().content, "ab");
    assert!(world.edited_geometry().texts().is_empty());
}

#[when("the user opens preset settings")]
fn open_preset_settings(world: &mut ApplicationWorld) {
    click_in_text_editor(world, "Preset settings");
}

#[then("current-preset save, rename and delete are disabled")]
fn current_actions_are_disabled(world: &mut ApplicationWorld) {
    for label in ["Save Current", "Rename Current", "Delete Current"] {
        world.expect_text(label);
        assert!(!world.run_is_enabled(label), "expected {label} disabled");
    }
}

#[then("Recently Used sort order and No Preset new-text default are checked")]
fn defaults_are_checked(world: &mut ApplicationWorld) {
    world.expect_text("Sort Order");
    world.expect_text("New Text Default");
    assert!(world.checked("Recently Used"));
    let choice = world.run("No Preset");
    assert!(world.checked_run(choice));
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("text-preset-defaults");
    }
}

#[test]
fn sketch_text_can_save_and_apply_a_session_preset_without_committing_geometry() {
    let mut world = ApplicationWorld::new();
    start_text_without_a_preset(&mut world);
    open_preset_name(&mut world);
    x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    world.expect_no_text("Preset name");
    click_editor_button(&mut world, "Cancel");
    open_editor(&mut world, [950, 630]);
    open_preset_list(&mut world);
    let last = world.run("Last Used").clone();
    let preset = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "ab" && run.list == last.list)
        .unwrap()
        .center();
    click_preset_ui(&mut world, preset);
    assert_eq!(world.state().sketch_text.unwrap().content, "ab");
    assert!(world.edited_geometry().texts().is_empty());
    click_editor_button(&mut world, "OK");
    open_editor(&mut world, [990, 635]);
    open_preset_list(&mut world);
    let last = world.run("Last Used").center();
    click_preset_ui(&mut world, last);
    assert_eq!(world.state().sketch_text.unwrap().content, "ab");
    assert_eq!(world.edited_geometry().texts().len(), 1);
    world.close();
}
