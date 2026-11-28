// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Hole preset settings (hole-10a, hole-10, new-hole default) and sketch picking (hole-11).
use super::super::super::*;
use super::setup::{
    HOLE_WINDOW, click_hole_control, click_hole_ribbon, click_popup_item, open_hole_fixture,
    open_hole_preset_settings, popup_item, red_preview_z_bounds, reopen_hole1, scene_bounds,
};
use cucumber::{given, then, when};
use oblikovati_gui::GuiHolePreview;
use std::cell::RefCell;

thread_local! {
    static HOLE11_Z_BEFORE_FLIP: RefCell<Option<(f32, f32)>> = const { RefCell::new(None) };
    static HOLE11_Z_EXTENDED: RefCell<Option<(f32, f32)>> = const { RefCell::new(None) };
    static HOLE11_Z_SHRUNK: RefCell<Option<(f32, f32)>> = const { RefCell::new(None) };
}

#[when("the user reopens Hole1 and expands Input Geometry")]
#[when("the user reopens it and expands Input Geometry")]
fn reopen_and_expand_input_geometry(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    let text = super::super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    if !text.contains("1Position") {
        click_hole_control(world, ">InputGeometry");
    }
}

#[then("the Positions selector shows one position")]
fn positions_shows_one_position(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("1Position"),
        "expected 1 Position in Hole window, saw {text}"
    );
}

#[then("the menu shows sort-order and new-hole-default choices")]
fn menu_shows_sort_and_default_choices(world: &mut ApplicationWorld) {
    let _ = popup_item(world, "Recently Used");
    let _ = popup_item(world, "Last Used");
    let _ = popup_item(world, "No Preset");
}

#[when("the user dismisses the menu and cancels the edit")]
#[when("the user dismisses the menu and cancels Hole creation")]
fn dismiss_menu_and_cancel_hole(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    click_hole_control(world, "Cancel");
}

#[then("the existing solid remains visible")]
fn existing_solid_remains_visible(world: &mut ApplicationWorld) {
    assert!(!world.app().window.scene_triangles.is_empty());
    assert!(super::super::super::diagnostics::window_geometry(HOLE_WINDOW).is_none());
}

#[given(
    "a disposable copy of `fixtures/cad/body-properties-start.opd` is open in an isolated application profile"
)]
fn open_body_properties_start_isolated(world: &mut ApplicationWorld) {
    open_hole_fixture(world, "fixtures/cad/body-properties-start.opd");
}

#[given("the user opens Hole creation and Preset Settings")]
#[when("the user opens Hole creation and Preset Settings again")]
fn open_hole_creation_and_presets(world: &mut ApplicationWorld) {
    click_hole_ribbon(world);
    open_hole_preset_settings(world);
}

#[when("the user selects No Preset under New Hole Default and reopens Preset Settings")]
fn select_no_preset_and_reopen(world: &mut ApplicationWorld) {
    click_popup_item(world, "No Preset");
    open_hole_preset_settings(world);
}

#[then("No Preset is checked")]
fn no_preset_is_checked(world: &mut ApplicationWorld) {
    assert!(world.checked_run(popup_item(world, "No Preset")));
}

#[when("the user selects Last Used under New Hole Default and reopens Preset Settings")]
fn select_last_used_and_reopen(world: &mut ApplicationWorld) {
    click_popup_item(world, "Last Used");
    open_hole_preset_settings(world);
}

#[then("Last Used is checked")]
#[then("Last Used remains checked")]
#[then("Last Used is checked as the new-hole default")]
fn last_used_is_checked(world: &mut ApplicationWorld) {
    assert!(world.checked_run(popup_item(world, "Last Used")));
}

#[given(
    "a writable copy of `fixtures/cad/hole/hole1-through-unique-candidate.opd` is open in a disposable application profile"
)]
fn open_hole1_disposable_profile(world: &mut ApplicationWorld) {
    open_hole_fixture(
        world,
        "fixtures/cad/hole/hole1-through-unique-candidate.opd",
    );
}

#[given("Hole1 has one position")]
fn hole1_has_one_position(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Hole1"));
}

#[given("the user reopens Hole1, chooses Last Used in Preset Settings, and cancels the edit")]
fn reopen_hole1_choose_last_used_cancel(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    open_hole_preset_settings(world);
    click_popup_item(world, "Last Used");
    click_hole_control(world, "Cancel");
}

#[then("no named preset is selected")]
fn no_named_preset_selected(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("NoPreset"),
        "expected No Preset selected, saw {text}"
    );
}

#[then("Sort Order offers Recently Used, Date Created, Alphanumeric and Last Modified")]
fn sort_order_offers_four_choices(world: &mut ApplicationWorld) {
    for label in [
        "Recently Used",
        "Date Created",
        "Alphanumeric",
        "Last Modified",
    ] {
        let _ = popup_item(world, label);
    }
}

#[then("New Hole Default offers Last Used, Use Current Preset and No Preset")]
fn new_hole_default_offers_three_choices(world: &mut ApplicationWorld) {
    for label in ["Last Used", "Use Current Preset", "No Preset"] {
        let _ = popup_item(world, label);
    }
}

#[then("Use Current Preset is disabled")]
fn use_current_preset_is_disabled(world: &mut ApplicationWorld) {
    assert!(
        popup_item(world, "Use Current Preset").alpha < 200,
        "expected Use Current Preset disabled when no named preset is selected"
    );
}

#[given("a writable copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open")]
fn open_extrusion_base_circle(world: &mut ApplicationWorld) {
    open_hole_fixture(world, "fixtures/cad/extrusion/extrusion-base-circle.opd");
}

#[given("the top-face circular sketch is visible")]
fn top_face_circular_sketch_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.sketches.iter().any(|s| s.visible),
        "expected visible top-face circular sketch"
    );
}

#[given("the user opens Hole with center-point creation enabled")]
fn open_hole_with_center_point(world: &mut ApplicationWorld) {
    click_hole_ribbon(world);
}

#[when("the user clicks near the upper circular sketch")]
fn click_near_upper_circular_sketch(world: &mut ApplicationWorld) {
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let top_z = f64::from(max[2]);
    for pt in [
        [cx, cy, top_z],
        [0.0, 0.0, top_z],
        [cx + 0.1, cy + 0.1, top_z],
    ] {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .hole_preview
            .as_ref()
            .is_some_and(|p| p.result == GuiHolePreview::Ready)
        {
            let bounds = red_preview_z_bounds(world);
            HOLE11_Z_BEFORE_FLIP.with(|cell| *cell.borrow_mut() = bounds);
            HOLE11_Z_EXTENDED.with(|cell| *cell.borrow_mut() = bounds);
            return;
        }
    }
    panic!(
        "expected click near upper circular sketch to place Hole position, saw {:?}",
        world.state().hole_preview
    );
}

#[then("Positions shows one position and OK is enabled")]
fn positions_one_and_ok_enabled(world: &mut ApplicationWorld) {
    positions_shows_one_position(world);
    assert!(world.run_is_enabled("OK"));
}

#[when("the user chooses the other Direction icon")]
fn choose_other_direction(world: &mut ApplicationWorld) {
    click_hole_control(world, "Flipped");
}

#[then("the red preview changes sides")]
fn red_preview_changes_sides(world: &mut ApplicationWorld) {
    let before = HOLE11_Z_BEFORE_FLIP
        .with(|cell| *cell.borrow())
        .expect("expected red preview bounds before flip");
    let after = red_preview_z_bounds(world).expect("expected red preview bounds after flip");
    assert!(
        after.1 > before.1 + 0.05 || after.0 > before.0 + 0.05,
        "expected flipped red preview to shift sides: before={before:?}, after={after:?}"
    );
}

#[when("the user restores the first direction and clear Extend Start")]
fn restore_direction_and_clear_extend_start(world: &mut ApplicationWorld) {
    click_hole_control(world, "Default");
    click_hole_control(world, "ExtendStart");
}

#[then("the red preview is reduced on one side of the sketch plane")]
fn red_preview_reduced_on_start_side(world: &mut ApplicationWorld) {
    let extended = HOLE11_Z_EXTENDED
        .with(|cell| *cell.borrow())
        .expect("expected extended red preview bounds");
    let shrunk = red_preview_z_bounds(world).expect("expected shrunk red preview bounds");
    HOLE11_Z_SHRUNK.with(|cell| *cell.borrow_mut() = Some(shrunk));
    assert!(
        (shrunk.1 - shrunk.0) < (extended.1 - extended.0) - 0.1,
        "expected Extend Start off to shrink red preview: extended={extended:?}, shrunk={shrunk:?}"
    );
}

#[when("the user checks Extend Start again")]
fn check_extend_start_again(world: &mut ApplicationWorld) {
    click_hole_control(world, "ExtendStart");
}

#[then("the larger preview returns")]
fn larger_preview_returns(world: &mut ApplicationWorld) {
    let shrunk = HOLE11_Z_SHRUNK
        .with(|cell| *cell.borrow())
        .expect("expected shrunk bounds");
    let restored = red_preview_z_bounds(world).expect("expected restored bounds");
    assert!(
        (restored.1 - restored.0) > (shrunk.1 - shrunk.0) + 0.1,
        "expected larger preview to return: shrunk={shrunk:?}, restored={restored:?}"
    );
}

#[when("the user cancels the command")]
fn cancel_hole_command(world: &mut ApplicationWorld) {
    click_hole_control(world, "Cancel");
}

#[then("the Hole property window closes")]
fn hole_window_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry(HOLE_WINDOW).is_none(),
        "expected Hole window to close"
    );
}
