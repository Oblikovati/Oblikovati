// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{diagnostics, ribbon_appearance::icon_quads::icon_quads_inside};
use super::{ApplicationWorld, click_in_extrude, numeric_preview};
use cucumber::{given, then, when};

const HIDE: &str = "HidePresets";
const EMPTY: &str = "NoPreset";

#[given("Extrusion with Hide Presets checked")]
fn preset_row_initially_hidden(world: &mut ApplicationWorld) {
    numeric_preview(world);
    assert!(preset_row_missing(world));
    open_settings(world);
    assert!(world.checked_run(menu_row(world, HIDE)));
}

#[when("Hide Presets is unchecked from the panel menu")]
fn reveal_presets(world: &mut ApplicationWorld) {
    click_menu_row(world, HIDE);
}

#[then("the No Preset row, plus and gear controls appear")]
fn assert_preset_controls(world: &mut ApplicationWorld) {
    let preset = world.run_in_window("Extrude", EMPTY);
    let geometry = world.run_in_window("Extrude", "InputGeometry");
    assert!(preset.center()[1] < geometry.center()[1]);
    preset_gear(world, EMPTY);
}

#[when("the preset gear menu is inspected and dismissed")]
fn inspect_preset_settings(world: &mut ApplicationWorld) {
    open_preset_menu(world, EMPTY);
    assert_empty_menu(world);
    dismiss_preset_settings(world);
    preset_panel_stays_open(world);
}

fn assert_empty_menu(world: &mut ApplicationWorld) {
    for label in [
        "SaveCurrent",
        "RenameCurrent",
        "DeleteCurrent",
        "UseCurrentPreset",
    ] {
        assert!(
            !world.run_is_enabled(label),
            "{label} should require a preset"
        );
    }
    preset_menu_choices(world);
    assert!(world.checked_run(menu_row(world, "RecentlyUsed")));
    assert!(world.checked_run(menu_row(world, EMPTY)));
}

#[then("the menu offers sort-order and new-command-default choices")]
fn preset_menu_choices(world: &mut ApplicationWorld) {
    menu_row(world, "SortOrder");
    menu_row(world, "NewCommandDefault");
    for label in [
        "RecentlyUsed",
        "DateCreated",
        "Alphanumeric",
        "LastModified",
        "LastUsed",
        "UseCurrentPreset",
        EMPTY,
    ] {
        menu_row(world, label);
    }
}

#[when("the user dismisses the menu without changing a choice")]
fn dismiss_preset_settings(world: &mut ApplicationWorld) {
    world.escape();
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|row| row.text == "SortOrder" && world.frame.is_popup_row(row))
    );
}

#[then("the feature Properties panel remains open")]
fn preset_panel_stays_open(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", EMPTY);
    world.run_in_window("Extrude", "DistanceA");
}

#[given(
    "a fresh copy of `fixtures/cad/revolve/revolve-profile.opd` is open in a disposable application profile"
)]
fn open_disposable_profile(world: &mut ApplicationWorld) {
    let relative = "fixtures/cad/revolve/revolve-profile.opd";
    let copy = world.scratch_folder().join("revolve-profile.opd");
    std::fs::copy(super::super::super::scenario_root().join(relative), &copy).unwrap();
    world.open_fixture(&copy);
}

#[given(
    "the user opens Extrude with the rectangular sketch profile selected and the preset row visible"
)]
fn open_extrude_with_presets(world: &mut ApplicationWorld) {
    world.click_text("Sketch1");
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.run_in_window("Extrude", "1Profile");
    open_settings(world);
    reveal_presets(world);
    assert_preset_controls(world);
}

#[when("the user opens Preset Settings")]
fn open_published_preset_menu(world: &mut ApplicationWorld) {
    open_preset_menu(world, EMPTY);
}

#[when("Hide Presets is restored")]
fn restore_hidden(world: &mut ApplicationWorld) {
    open_settings(world);
    assert!(!world.checked_run(menu_row(world, HIDE)));
    click_menu_row(world, HIDE);
}

#[then("the preset row disappears")]
fn preset_row_disappears(world: &mut ApplicationWorld) {
    assert!(preset_row_missing(world));
    world.run_in_window("Extrude", "DistanceA");
    assert!(world.run_is_enabled("OK"));
    world.save_capture("extrude-presets-hidden");
}

fn preset_row_missing(world: &ApplicationWorld) -> bool {
    !world.window_text("Extrude").contains(EMPTY)
}

fn open_settings(world: &mut ApplicationWorld) {
    click_in_extrude(world, "AdvancedSettings");
}

fn menu_row<'a>(world: &'a ApplicationWorld, name: &str) -> &'a diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|row| row.text == name && world.frame.is_popup_row(row))
        .unwrap_or_else(|| panic!("expected popup row {name:?}"))
}

fn click_menu_row(world: &mut ApplicationWorld, name: &str) {
    let point = diagnostics::pointer_pixels(menu_row(world, name).center());
    world.click(point);
    world.pump(Vec::new());
}

fn click_preset_control(
    world: &mut ApplicationWorld,
    location: impl Fn(&ApplicationWorld) -> [i32; 2],
) {
    // The row gets its final width after the popup closes and the panel refits.
    world.hover(location(world));
    super::super::viewport_interaction::click_at(world, location(world));
    world.pump(Vec::new());
}

fn preset_add(world: &ApplicationWorld, caption: &str) -> [f32; 2] {
    let row = world.run_in_window("Extrude", caption);
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "+" && (run.center()[1] - row.center()[1]).abs() < 3.)
        .expect("expected the add button on the preset row")
        .center()
}

fn preset_gear(world: &ApplicationWorld, caption: &str) -> [i32; 2] {
    let [x, y] = preset_add(world, caption);
    let icons = icon_quads_inside(([x + 4., y - 12.], [x + 64., y + 12.]));
    let [(lo, hi)] = icons.as_slice() else {
        panic!("expected one gear to the right of the preset add button: {icons:?}");
    };
    diagnostics::pointer_pixels([(lo[0] + hi[0]) / 2., (lo[1] + hi[1]) / 2.])
}

fn open_preset_menu(world: &mut ApplicationWorld, selector: &str) {
    click_preset_control(world, |world| preset_gear(world, selector));
}

fn add_named_preset(world: &mut ApplicationWorld, name: &str) {
    click_preset_control(world, |world| {
        diagnostics::pointer_pixels(preset_add(world, EMPTY))
    });
    world.run_in_window("Extrude", "Presetname");
    world.pump(Vec::new());
    world.type_text(name);
    click_in_extrude(world, "Savepreset");
}

fn reopen_extrude(world: &mut ApplicationWorld) {
    click_preset_control(world, |world| {
        diagnostics::pointer_pixels(world.run_in_window("Extrude", "Cancel").center())
    });
    assert!(diagnostics::window_geometry("Extrude").is_none());
    world.click_text("Extrude");
    world.pump(Vec::new());
}

fn assert_distance(world: &mut ApplicationWorld, value: &str) {
    let draft = world.state().extrude_draft_preview.unwrap();
    assert_eq!(draft.distance, value);
}

#[test]
fn named_extrude_preset_restores_distance_into_a_new_draft() {
    let mut world = ApplicationWorld::new();
    preset_row_initially_hidden(&mut world);
    reveal_presets(&mut world);
    add_named_preset(&mut world, "Saved depth");
    world.run_in_window("Extrude", "Saveddepth");

    reopen_extrude(&mut world);
    super::super::replace_extrusion_distance(&mut world, "8");
    click_in_extrude(&mut world, EMPTY);
    click_menu_row(&mut world, "Saveddepth");
    assert_distance(&mut world, "10.00 mm");

    update_and_remove_preset(&mut world);
    world.close();
}

fn update_and_remove_preset(world: &mut ApplicationWorld) {
    super::super::replace_extrusion_distance(world, "7");
    open_preset_menu(world, "Saveddepth");
    click_menu_row(world, "SaveCurrent");
    open_preset_menu(world, "Saveddepth");
    click_menu_row(world, "RenameCurrent");
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text("Short depth");
    click_in_extrude(world, "Savepreset");
    world.run_in_window("Extrude", "Shortdepth");

    open_preset_menu(world, "Shortdepth");
    click_menu_row(world, "UseCurrentPreset");
    reopen_extrude(world);
    assert_distance(world, "7");
    open_preset_menu(world, "Shortdepth");
    click_menu_row(world, "DeleteCurrent");
    world.run_in_window("Extrude", EMPTY);
}

#[test]
fn last_used_extrude_settings_come_from_the_accepted_draft() {
    let mut world = ApplicationWorld::new();
    preset_row_initially_hidden(&mut world);
    reveal_presets(&mut world);
    open_preset_menu(&mut world, EMPTY);
    click_menu_row(&mut world, "LastUsed");
    super::super::replace_extrusion_distance(&mut world, "8");
    click_in_extrude(&mut world, "OK");
    assert_eq!(super::feature_count(&mut world), 1);
    world.click_text("Extrude");
    world.pump(Vec::new());
    assert_distance(&mut world, "8");
    world.close();
}
