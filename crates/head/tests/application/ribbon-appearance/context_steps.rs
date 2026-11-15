// SPDX-License-Identifier: GPL-2.0-only
//! Appearance scenarios across tabs, expanded panels, and sketch editing.
use super::*;
use cucumber::{given, then, when};

const MODIFY_AREA: &str = "##ribbon-additional-PartModify";
fn area_geometry() -> ([f32; 2], [f32; 2]) {
    diagnostics::window_geometry(MODIFY_AREA).expect("expected the open Modify area")
}
#[then("its ribbon commands use the compact layout")]
#[then("the Tools ribbon commands still use the compact layout")]
fn tools_compact(world: &mut ApplicationWorld) {
    world.expect_text("Options");
    modes_native::assert_no_ribbon_labels(world, &["Measure", "Application Options", "Find"]);
    let icons = icon_quads::ribbon_icon_quads(modes_native::ribbon_bottom());
    assert!(
        icons.len() >= 4
            && icons
                .iter()
                .all(|(min, _)| (min[1] - icons[0].0[1]).abs() < 1.)
    );
}
#[when(
    "the user cycles the ribbon through panel buttons, panel titles, and tabs only back to the full ribbon"
)]
fn cycle_to_full(world: &mut ApplicationWorld) {
    world.click_text("Ribbon Display");
    world.expect_text("Options");
    world.click_text("Ribbon Display");
    world.expect_text("Options");
    world.click_text("Ribbon Display");
    world.expect_no_text("Options");
    world.click_text("Ribbon Display");
    world.expect_text("Options");
}
#[then("the Ribbon Appearance submenu marks \"Compact\" as selected")]
fn compact_selected_from_tools(world: &mut ApplicationWorld) {
    modes_native::open_appearance_from(world, "Options");
    assert!(world.checked("Compact"));
}
#[when("the user selects \"Text Off\" from Ribbon Appearance")]
fn text_off_with_expansion(world: &mut ApplicationWorld) {
    world.observed_row_gap = Some(area_geometry().1[0]);
    modes_native::choose_appearance(world, "Modify", "Text Off");
}
#[then("the Modify additional command area remains open")]
fn modify_area_stays_open(_: &mut ApplicationWorld) {
    area_geometry();
}
#[then("its command icons remain visible without their text labels")]
fn expanded_icons_without_labels(world: &mut ApplicationWorld) {
    let (origin, size) = area_geometry();
    let end = [origin[0] + size[0], origin[1] + size[1]];
    assert!(icon_quads::icon_quads_inside((origin, end)).len() >= 3);
    for label in ["Move Bodies", "Bend Part", "Copy Object"] {
        world.expect_no_text(label);
    }
}
#[then("the expanded area becomes narrower")]
fn expanded_area_narrows(world: &mut ApplicationWorld) {
    let before = world
        .observed_row_gap
        .expect("expected the original Modify width");
    assert!(area_geometry().1[0] < before);
}
#[then("the Modify panel title remains visible")]
fn modify_title_visible(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
}
#[when("the user selects \"Normal\" from Ribbon Appearance")]
fn normal_with_expansion(world: &mut ApplicationWorld) {
    world.observed_view = Some([area_geometry().1[0] as i32, 0]);
    modes_native::choose_appearance(world, "Modify", "Normal");
}
#[then("its command labels are visible again")]
fn expanded_labels_restored(world: &mut ApplicationWorld) {
    for label in ["Move Bodies", "Bend Part", "Copy Object"] {
        world.expect_text(label);
    }
}
#[then("the expanded area returns to its wider layout")]
fn expanded_area_restored(world: &mut ApplicationWorld) {
    let narrow = world
        .observed_view
        .expect("expected the narrow Modify width")[0];
    let original = world
        .observed_row_gap
        .expect("expected the original Modify width");
    let width = area_geometry().1[0];
    assert!(width > narrow as f32 && (width - original).abs() < 1.);
}
#[given("the ribbon appearance is Normal")]
fn normal_tools_appearance(world: &mut ApplicationWorld) {
    modes_native::open_appearance_from(world, "Options");
    assert!(world.checked("Normal"));
    world.escape();
    world.escape();
}
#[when("the user right-clicks the Tools Options panel title")]
fn tools_options_context(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Options").center());
    world.right_click(title);
}
#[when("the user selects Text Off")]
fn select_text_off_open_menu(world: &mut ApplicationWorld) {
    world.click_text("Text Off");
}
#[then("the Tools command icons are visible without their command labels")]
fn tools_icon_only(world: &mut ApplicationWorld) {
    modes_native::assert_no_ribbon_labels(
        world,
        &[
            "Measure",
            "Application Options",
            "Document Settings",
            "Find",
        ],
    );
    assert!(icon_quads::ribbon_icon_quads(modes_native::ribbon_bottom()).len() >= 4);
}
#[then("the Tools panel titles remain visible")]
fn tools_titles_visible(world: &mut ApplicationWorld) {
    world.expect_text("Options");
    world.expect_text("Measure");
}
#[then("the 3D Model command icons are visible without their command labels")]
fn model_icon_only(world: &mut ApplicationWorld) {
    mode_steps::icon_only_commands(world);
}
#[then("the 3D Model panel titles remain visible")]
fn model_titles_visible(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    world.expect_text("Work Features");
}
#[then("Text Off is marked as selected")]
fn text_off_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Text Off"));
}
#[when("the user selects Normal")]
fn normal_open_menu(world: &mut ApplicationWorld) {
    world.click_text("Normal");
}
#[then("command labels are visible on the 3D Model ribbon")]
fn model_labels_visible(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
}
#[then("the full Tools ribbon and its command labels are visible")]
fn tools_labels_visible(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_text("Find");
}
#[given("Ribbon Appearance is set to Normal")]
fn sketch_normal(world: &mut ApplicationWorld) {
    modes_native::open_appearance_from(world, "Create");
    assert!(world.checked("Normal"));
    world.escape();
    world.escape();
}
fn choose_sketch(world: &mut ApplicationWorld, mode: &str) {
    let icons = icon_quads::ribbon_icon_quads(modes_native::ribbon_bottom());
    let edge = icons
        .iter()
        .map(|(min, max)| max[0] - min[0])
        .fold(0., f32::max);
    world.observed_view = Some([modes_native::ribbon_bottom() as i32, edge as i32]);
    modes_native::choose_appearance(world, "Create", mode);
}
#[when("the user selects Small from the Ribbon Appearance submenu")]
fn sketch_small(world: &mut ApplicationWorld) {
    choose_sketch(world, "Small");
}
#[when("the user selects Compact from the Ribbon Appearance submenu")]
fn sketch_compact(world: &mut ApplicationWorld) {
    choose_sketch(world, "Compact");
}
#[when("the user selects Large from the Ribbon Appearance submenu")]
fn sketch_large(world: &mut ApplicationWorld) {
    choose_sketch(world, "Large");
}
#[when("the user selects Normal from the Ribbon Appearance submenu")]
fn sketch_restore(world: &mut ApplicationWorld) {
    choose_sketch(world, "Normal");
}
#[then("command icons become smaller and the command area becomes shorter")]
fn sketch_small_metrics(world: &mut ApplicationWorld) {
    mode_steps::small_icons_and_band(world);
}
#[then("command icons form a dense horizontal layout within the panel groups")]
fn sketch_compact_row(world: &mut ApplicationWorld) {
    mode_steps::compact_single_row(world);
}
#[then("command icons become larger and occupy multiple rows")]
fn sketch_large_rows(world: &mut ApplicationWorld) {
    mode_steps::large_two_rows(world);
}
#[then("command labels are hidden")]
fn sketch_labels_hidden(world: &mut ApplicationWorld) {
    modes_native::assert_no_ribbon_labels(
        world,
        &["Rectangle", "Horizontal", "Construction", "Move", "Delete"],
    );
}
#[then("the normal mixed-size command layout is restored")]
fn sketch_normal_restored(world: &mut ApplicationWorld) {
    world.expect_text("Rectangle");
    let icons = icon_quads::ribbon_icon_quads(modes_native::ribbon_bottom());
    assert!(icons.iter().any(|(min, max)| max[0] - min[0] >= 39.));
    assert!(icons.iter().any(|(min, max)| max[0] - min[0] < 30.));
}
