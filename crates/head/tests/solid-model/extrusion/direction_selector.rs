// SPDX-License-Identifier: GPL-2.0-only
use super::direction::preview_depth;
use super::viewport_interaction::click_text_native;
use super::*;

const OPTIONS: [&str; 4] = ["Default", "Flipped", "Symmetric", "Asymmetric"];

#[given("the rectangle extrusion preview is open with icon direction controls")]
fn icon_preview(world: &mut ApplicationWorld) {
    selected_rectangle_preview(world);
    icons_visible(world);
}

#[given(
    "a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with an active rectangular Extrude preview and icon direction controls"
)]
fn published_icon_preview(world: &mut ApplicationWorld) {
    super::reference_extent::native_fixture_preview(world);
    icons_visible(world);
}

#[when("the user opens the Direction menu")]
fn open_style_menu(world: &mut ApplicationWorld) {
    click_text_native(world, "Direction");
}

#[then("Selector Style offers Icons and Dropdown")]
fn menu_offers_styles(world: &mut ApplicationWorld) {
    popup_option(world, "SelectorStyle");
    popup_option(world, "Icons");
    popup_option(world, "Dropdown");
}

#[then("Icons is selected")]
fn icons_is_selected(world: &mut ApplicationWorld) {
    assert!(world.checked_run(popup_option(world, "Icons")));
    assert!(!world.checked_run(popup_option(world, "Dropdown")));
}

#[when("the user chooses Dropdown")]
fn choose_dropdown(world: &mut ApplicationWorld) {
    click_popup_option(world, "Dropdown");
}

#[then("the Direction field displays Default")]
fn field_displays_default(world: &mut ApplicationWorld) {
    default_dropdown_visible(world);
    assert_eq!(preview_depth(world), [0., 1.]);
}

fn default_dropdown_visible(world: &ApplicationWorld) {
    world.run_in_window("Extrude", "Default");
    assert!(!world.frame.runs.iter().any(|run| run.text == "Asymmetric"));
}

#[when("the user opens the Direction list")]
fn open_direction_list(world: &mut ApplicationWorld) {
    click_text_native(world, "Default");
}

#[then("it offers Default, Flipped, Symmetric and Asymmetric")]
fn all_directions_in_list(world: &mut ApplicationWorld) {
    for label in OPTIONS {
        popup_option(world, label);
    }
    world.save_capture("direction-dropdown");
}

#[when("the user opens the Direction menu and chooses Icons")]
fn choose_icons(world: &mut ApplicationWorld) {
    if world
        .frame
        .runs
        .iter()
        .any(|run| run.text == "Asymmetric" && world.frame.is_popup_row(run))
    {
        world.escape();
    }
    open_style_menu(world);
    click_popup_option(world, "Icons");
}

#[then("the four direction icons are visible again")]
fn icons_restored(world: &mut ApplicationWorld) {
    icons_visible(world);
    world.save_capture("direction-icons");
}

fn icons_visible(world: &mut ApplicationWorld) {
    assert_eq!(
        super::direction::direction_icons(world).len(),
        OPTIONS.len()
    );
    let text = world.window_text("Extrude");
    let mut remaining = text.as_str();
    for label in OPTIONS {
        remaining = remaining
            .split_once(label)
            .unwrap_or_else(|| panic!("expected direction labels in order: {text:?}"))
            .1;
    }
}

fn popup_option<'w>(
    world: &'w ApplicationWorld,
    label: &str,
) -> &'w super::super::diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == label && world.frame.is_popup_row(run))
        .unwrap_or_else(|| panic!("expected {label} in the open popup"))
}

fn click_popup_option(world: &mut ApplicationWorld, label: &str) {
    let position = super::super::diagnostics::pointer_pixels(popup_option(world, label).center());
    super::viewport_interaction::click_at(world, position);
}

#[test]
fn native_extrude_selector_preserves_direction_and_remembers_style_on_cancel() {
    let mut world = ApplicationWorld::new();
    icon_preview(&mut world);
    open_style_menu(&mut world);
    choose_dropdown(&mut world);
    open_direction_list(&mut world);
    click_popup_option(&mut world, "Symmetric");
    world.run_in_window("Extrude", "Symmetric");
    assert_eq!(preview_depth(&mut world), [-0.5, 0.5]);
    choose_icons(&mut world);
    assert_eq!(preview_depth(&mut world), [-0.5, 0.5]);
    open_style_menu(&mut world);
    choose_dropdown(&mut world);
    click_text_native(&mut world, "Cancel");
    click_text_native(&mut world, "Extrude");
    world.pump(Vec::new());
    default_dropdown_visible(&world);
    world.close();
}
