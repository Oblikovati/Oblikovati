// SPDX-License-Identifier: GPL-2.0-only
//! Exercise named and last-accepted defaults through the native Format Text editor.
use super::*;
use text_fraction::{
    assert_stacked, existing_blue_text, replace_with_fraction, set_fraction_width,
    stack_selected_fraction,
};
use text_presets::{choose_preset_action, open_preset_list, open_preset_settings};

#[test]
fn sketch_text_new_placements_distinguish_named_and_last_used_defaults() {
    let mut world = ApplicationWorld::new();
    save_g_with_a_different_last_accepted_width(&mut world);
    let saved_geometry = world.edited_geometry();
    text_edit::open_existing(&mut world);
    world.run_in_window("Format Text", "*g");
    set_new_text_default(&mut world, "Use Current Preset");
    activate_text(&mut world);
    place_with_default(&mut world, [970, 650]);
    expect_fraction_default(&mut world, "g", "100");
    check_last_used_default(&mut world);
    check_clear_default_keeps_g(&mut world);
    assert_eq!(world.edited_geometry(), saved_geometry);
    world.close();
}

fn save_g_with_a_different_last_accepted_width(world: &mut ApplicationWorld) {
    existing_blue_text(world);
    replace_with_fraction(world);
    stack_selected_fraction(world);
    text_fraction_preset::name_fraction_preset(world, b'g');
    set_fraction_width(world, b"50");
    expect_fraction_default(world, "*g", "50");
    click_editor_button(world, "OK");
    world.expect_ready();
}

fn set_new_text_default(world: &mut ApplicationWorld, label: &str) {
    open_preset_settings(world);
    choose_preset_action(world, label);
    world.expect_no_text("New Text Default");
    click_editor_button(world, "Cancel");
    assert!(
        world.state().sketch_text.is_none(),
        "closed editor after setting {label}"
    );
}

fn place_with_default(world: &mut ApplicationWorld, anchor: [i32; 2]) {
    x11_input::click(anchor);
    world.pump(Vec::new());
    world.expect_text("Format Text");
}

fn expect_fraction_default(world: &mut ApplicationWorld, caption: &str, width: &str) {
    world.run_in_window("Format Text", caption);
    world.run_in_window("Format Text", width);
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected a stacked fraction preview from the new Text default");
    };
    assert_stacked(&style);
    assert_eq!(
        oblikovati_model::sketch::uniform_sketch_text_width(
            &style.content,
            style.formatting_runs.as_deref().unwrap_or_default(),
        ),
        width.parse().ok()
    );
}

fn check_last_used_default(world: &mut ApplicationWorld) {
    set_new_text_default(world, "Last Used");
    place_with_default(world, [980, 660]);
    expect_fraction_default(world, "*g", "50");
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("last-used-asterisk-and-clock");
    }
    open_preset_settings(world);
    assert!(world.checked("Last Used"));
    choose_preset_action(world, "No Preset");
    world.expect_no_text("New Text Default");
    click_editor_button(world, "Cancel");
}

fn check_clear_default_keeps_g(world: &mut ApplicationWorld) {
    place_with_default(world, [990, 670]);
    let draft = world.state().sketch_text.unwrap();
    assert_eq!(draft, oblikovati_gui::GuiSketchTextDraft::default());
    world.run_in_window("Format Text", "3.05");
    world.run_in_window("Format Text", "100");
    open_preset_list(world);
    let list = world.run("Last Used").list;
    assert_eq!(world.run("g").list, list);
}
