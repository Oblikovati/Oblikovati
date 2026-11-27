// SPDX-License-Identifier: GPL-2.0-only
//! Scenario steps for the ribbon appearance selector.
use super::*;
use cucumber::{then, when};

#[when("the user right-clicks a ribbon panel title and opens \"Ribbon Appearance\"")]
fn open_appearance_from_model_panel(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Modify").center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
}

#[then("the submenu offers \"Normal\", \"Text Off\", \"Small\", \"Compact\", and \"Large\"")]
fn appearance_choices_are_available(world: &mut ApplicationWorld) {
    for label in ["Normal", "Text Off", "Small", "Compact", "Large"] {
        assert!(
            world.run_is_enabled(label),
            "appearance {label:?}: expected an enabled choice"
        );
    }
}

#[then("\"Normal\" is marked as selected")]
#[then("Normal is checked")]
fn normal_appearance_is_selected(world: &mut ApplicationWorld) {
    assert!(world.checked("Normal"));
}

#[then("the submenu offers \"Reset Ribbon\"")]
fn reset_ribbon_is_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Reset Ribbon"));
}

#[then("Text Off, Small, Compact, Large and Reset Ribbon are listed")]
fn display_mode_choices(world: &mut ApplicationWorld) {
    appearance_choices_are_available(world);
    reset_ribbon_is_available(world);
}

#[cucumber::given("the full 3D Model ribbon uses the Text Off appearance")]
fn model_text_off(world: &mut ApplicationWorld) {
    super::modes_native::choose_appearance(world, "Modify", "Text Off");
}
#[cucumber::given("the full 3D Model ribbon uses the Small appearance")]
fn model_small(world: &mut ApplicationWorld) {
    super::modes_native::choose_appearance(world, "Modify", "Small");
}
#[cucumber::given("the full 3D Model ribbon uses the Compact appearance")]
fn model_compact(world: &mut ApplicationWorld) {
    super::modes_native::choose_appearance(world, "Modify", "Compact");
}
#[when("the user selects \"Text Off\" from the Ribbon Appearance submenu")]
fn select_text_off(world: &mut ApplicationWorld) {
    choose_and_record(world, "Text Off");
}
#[when("the user selects \"Small\" from the Ribbon Appearance submenu")]
fn select_small(world: &mut ApplicationWorld) {
    choose_and_record(world, "Small");
}
#[when("the user selects \"Compact\" from the Ribbon Appearance submenu")]
fn select_compact(world: &mut ApplicationWorld) {
    choose_and_record(world, "Compact");
}
#[when("the user selects \"Large\" from the Ribbon Appearance submenu")]
fn select_large(world: &mut ApplicationWorld) {
    choose_and_record(world, "Large");
}
fn choose_and_record(world: &mut ApplicationWorld, appearance: &str) {
    let icons = super::icon_quads::ribbon_icon_quads(super::modes_native::ribbon_bottom());
    let edge = icons
        .iter()
        .map(|(min, max)| max[0] - min[0])
        .fold(0., f32::max);
    world.observed_view = Some([super::modes_native::ribbon_bottom() as i32, edge as i32]);
    world.observed_row_gap = Some(world.run("Work Features").minimum[0]);
    super::modes_native::choose_appearance(world, "Modify", appearance);
}
#[then("command icons remain visible without command labels")]
pub(super) fn icon_only_commands(world: &mut ApplicationWorld) {
    let icons = super::icon_quads::ribbon_icon_quads(super::modes_native::ribbon_bottom());
    assert!(icons.len() >= 4, "expected visible ribbon icons");
    super::modes_native::assert_no_ribbon_labels(
        world,
        &["Extrude", "Revolve", "Hole", "Fillet", "Chamfer", "UCS"],
    );
}
#[then("panel titles remain visible")]
#[then("the panel titles remain visible")]
fn panel_titles_remain(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    if world.state().authoring == oblikovati_gui::GuiAuthoringContext::Sketch {
        world.expect_text("Create");
        world.expect_text("Constrain");
    } else {
        world.expect_text("Work Features");
    }
}
#[then("the ribbon panels reflow")]
fn panel_positions_change(world: &mut ApplicationWorld) {
    let before = world
        .observed_row_gap
        .expect("expected the prior panel position");
    let after = world.run("Work Features").minimum[0];
    assert!(
        (after - before).abs() > 1.,
        "panel position {before} -> {after}: expected reflow"
    );
}
#[when("the user reopens the Ribbon Appearance submenu")]
fn reopen_appearance(world: &mut ApplicationWorld) {
    super::modes_native::open_appearance_from(world, "Modify");
}
#[then("\"Text Off\" is marked as selected")]
fn text_off_selected(world: &mut ApplicationWorld) {
    assert!(world.checked("Text Off"));
}
#[then("\"Small\" is marked as selected")]
fn small_selected(world: &mut ApplicationWorld) {
    assert!(world.checked("Small"));
}
#[then("\"Compact\" is marked as selected")]
fn compact_selected(world: &mut ApplicationWorld) {
    assert!(world.checked("Compact"));
}
#[then("\"Large\" is marked as selected")]
fn large_selected(world: &mut ApplicationWorld) {
    assert!(world.checked("Large"));
}
#[then("the command icons are smaller and the command area is shorter")]
pub(super) fn small_icons_and_band(world: &mut ApplicationWorld) {
    let [prior_height, prior_edge] = world
        .observed_view
        .expect("expected the previous ribbon metrics");
    let icons = super::icon_quads::ribbon_icon_quads(super::modes_native::ribbon_bottom());
    assert!(
        icons.len() >= 4
            && icons
                .iter()
                .all(|(min, max)| max[0] - min[0] < prior_edge as f32)
    );
    assert!(super::modes_native::ribbon_bottom() < prior_height as f32);
}
#[then("multiple command icons are arranged in a single horizontal row")]
pub(super) fn compact_single_row(_world: &mut ApplicationWorld) {
    let icons = super::icon_quads::ribbon_icon_quads(super::modes_native::ribbon_bottom());
    assert!(icons.len() >= 4, "expected multiple ribbon icons");
    assert!(
        icons
            .iter()
            .all(|(min, _)| (min[1] - icons[0].0[1]).abs() < 1.)
    );
}
#[then("at least one group is displayed as a compact panel button")]
fn compact_group_button(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Simplification").center());
    assert!(!diagnostics::command_popup_open());
    world.click(title);
    world.pump(Vec::new());
    assert!(diagnostics::command_popup_open());
    world.escape();
    world.pump(Vec::new());
    assert!(!diagnostics::command_popup_open());
}
#[then("larger command icons appear in two rows without visible command labels")]
pub(super) fn large_two_rows(world: &mut ApplicationWorld) {
    let icons = super::icon_quads::ribbon_icon_quads(super::modes_native::ribbon_bottom());
    assert!(icons.len() >= 4 && icons.iter().all(|(min, max)| max[0] - min[0] >= 39.));
    assert!(
        icons
            .iter()
            .any(|(min, _)| (min[1] - icons[0].0[1]).abs() > 20.)
    );
    super::modes_native::assert_no_ribbon_labels(world, &["Extrude", "Revolve", "Hole", "UCS"]);
}
#[then("the ribbon shows its mixed-size icons and command labels again")]
fn normal_mixed_icons(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
    let icons = super::icon_quads::ribbon_icon_quads(super::modes_native::ribbon_bottom());
    assert!(icons.iter().any(|(min, max)| max[0] - min[0] >= 39.));
    assert!(icons.iter().any(|(min, max)| max[0] - min[0] < 30.));
}
