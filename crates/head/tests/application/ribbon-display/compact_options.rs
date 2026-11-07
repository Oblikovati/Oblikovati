// SPDX-License-Identifier: GPL-2.0-only
//! Native interactions for the shared compact Options ribbon scenarios.
use super::*;

fn options_button(world: &ApplicationWorld) -> super::super::diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Options")
        .min_by(|a, b| a.minimum[1].total_cmp(&b.minimum[1]))
        .expect("expected the compact Options button")
        .clone()
}

fn options_button_color(world: &mut ApplicationWorld) -> [u8; 4] {
    let button = options_button(world);
    world.pixel_at([button.minimum[0] as usize - 4, button.center()[1] as usize])
}

#[given("the Tools ribbon is minimized to panel buttons")]
fn compact_tools(world: &mut ApplicationWorld) {
    full_tools(world);
    click_display_menu(world);
    world.click_text("Minimize to Panel Buttons");
    world.hover([1000, 600]);
    world.compact_options_idle_color = Some(options_button_color(world));
}

#[when("the user clicks the Options panel button")]
#[when("the user clicks the Options panel button again")]
fn click_options(world: &mut ApplicationWorld) {
    world.click(super::super::diagnostics::pointer_pixels(
        options_button(world).center(),
    ));
    world.hover([1000, 600]);
}

#[then("the temporary Options command panel appears below the button strip")]
#[then("the temporary Options command panel remains visible")]
fn options_panel_visible(world: &mut ApplicationWorld) {
    world.expect_text("Document Settings");
    let (position, _) = window_geometry("##ribbon-temporary-panel")
        .expect("Options commands: expected the temporary panel");
    let button = options_button(world);
    assert!(
        position[1] > button.maximum[1],
        "panel {position:?}, button {button:?}"
    );
}

#[then("the Options panel button is highlighted")]
#[then("the Options panel button remains highlighted")]
fn options_highlighted(world: &mut ApplicationWorld) {
    let idle = world
        .compact_options_idle_color
        .expect("expected idle button color");
    assert_ne!(
        options_button_color(world),
        idle,
        "expected a highlighted Options button"
    );
}

#[when("the user opens the ribbon display-mode menu")]
fn open_display_menu(world: &mut ApplicationWorld) {
    click_display_menu(world);
}

#[then(expr = "the menu shows {string} as selected")]
fn display_mode_checked(world: &mut ApplicationWorld, label: String) {
    assert!(world.checked(&label), "expected {label:?} selected");
}

#[when(expr = "the user selects {string}")]
fn select_display_mode(world: &mut ApplicationWorld, label: String) {
    world.click_text(&label);
}

#[then("the temporary Options command panel closes")]
fn options_panel_closed(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
    world.expect_no_text("Document Settings");
}

#[then("the ribbon displays tabs without a permanent panel-button strip")]
fn tabs_without_panels(world: &mut ApplicationWorld) {
    world.expect_text("Tools");
    world.expect_no_text("Options");
}
