// SPDX-License-Identifier: GPL-2.0-only
use super::super::{direction::preview_depth, replace_extrusion_distance};
use super::preset_visibility::{menu_row, open_settings};
use super::{ApplicationWorld, feature_count, native_fixture_preview};
use cucumber::{given, then};

const ENTER: &str = "SingleENTERtofinishcommand";

#[given(
    regex = r"^a feature Properties panel has valid geometry and Single ENTER to finish command is (checked|unchecked)$"
)]
fn set_enter_preference(world: &mut ApplicationWorld, setting: String) {
    native_fixture_preview(world);
    open_settings(world);
    let checked = setting == "checked";
    toggle_enter_if_needed(world, checked);
    open_settings(world);
    assert_eq!(world.checked_run(menu_row(world, ENTER)), checked);
    world.escape();
}

fn toggle_enter_if_needed(world: &mut ApplicationWorld, checked: bool) {
    let row = menu_row(world, ENTER);
    if world.checked_run(row) == checked {
        world.escape();
        return;
    }
    let point = super::super::super::diagnostics::pointer_pixels(row.center());
    world.click(point);
    world.pump(Vec::new());
}

#[given("the user has entered a valid value in the active Properties field")]
fn type_valid_distance(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "8");
    assert_eq!(feature_count(world), 0);
    assert_eq!(preview_depth(world), [0., 0.8]);
    assert!(world.run_is_enabled("OK"));
}

#[then("the feature is committed and Properties closes")]
fn confirm_extrusion_on_enter(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    assert!(super::super::super::diagnostics::window_geometry("Extrude").is_none());
    assert_eq!(preview_depth(world), [0., 0.8]);
    world.expect_text("Extrusion1");
}

#[then("the value is applied to the preview and Properties remains open")]
fn retain_extrude_preview_on_enter(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 0);
    assert_eq!(preview_depth(world), [0., 0.8]);
    let draft = world.state().extrude_draft_preview.expect("valid preview");
    assert_eq!(draft.distance, "8");
    world.run_in_window("Extrude", "DistanceA");
    assert!(world.run_is_enabled("OK"));
}
