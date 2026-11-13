// SPDX-License-Identifier: GPL-2.0-only
use super::super::diagnostics::{pointer_pixels, window_geometry};
use super::*;

fn click_side_selector(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry("##ribbon-column").expect("expected side ribbon");
    world.click(pointer_pixels([
        position[0] + size[0] - 19.,
        position[1] + size[1] - 20.,
    ]));
    world.pump(Vec::new());
}

#[when("I open the direct panel selector")]
#[when("the user clicks the direct panel selector at the bottom of the ribbon")]
#[when("the user clicks the direct panel selector arrow")]
fn open_side_selector(world: &mut ApplicationWorld) {
    click_side_selector(world);
    assert!(popup_depth() > 0, "expected side selector popup");
}

#[when("I select Modify in the panel selector")]
#[when("the user clicks Modify in the checklist")]
#[when("the user clicks Modify in the checklist again")]
fn toggle_side_modify(world: &mut ApplicationWorld) {
    world.ribbon_height = side_group_y(world, "Work Features");
    world.click_text("Modify");
}

pub(super) fn side_group_y(world: &ApplicationWorld, label: &str) -> f32 {
    let (position, size) = window_geometry("##ribbon-column").expect("expected side ribbon");
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            let center = run.center();
            run.text.replace(' ', "") == label.replace(' ', "")
                && (0..2).all(|axis| {
                    center[axis] >= position[axis] && center[axis] < position[axis] + size[axis]
                })
        })
        .unwrap_or_else(|| {
            panic!(
                "expected {label:?} inside side ribbon {position:?} {size:?}; runs {:?}",
                world
                    .frame
                    .runs
                    .iter()
                    .filter(|run| run.text.contains("Work"))
                    .map(|run| (&run.text, run.center()))
                    .collect::<Vec<_>>()
            )
        })
        .center()[1]
}

#[then("the Modify group returns to the side-docked ribbon")]
fn side_modify_returns(world: &mut ApplicationWorld) {
    side_group_y(world, "Modify");
}

#[then("Modify is checked and its panel is visible on the left-docked ribbon")]
#[then("Modify is checked and its panel is visible on the right-docked ribbon")]
fn checked_modify_visible(world: &mut ApplicationWorld) {
    side_modify_checked(world);
    side_modify_returns(world);
}

#[then("the Modify group disappears from the side-docked ribbon")]
fn side_modify_hidden(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry("##ribbon-column").expect("expected side ribbon");
    let visible = world.frame.runs.iter().any(|run| {
        let center = run.center();
        run.text == "Modify"
            && (0..2).all(|axis| {
                center[axis] >= position[axis] && center[axis] < position[axis] + size[axis]
            })
    });
    assert!(!visible, "expected no Modify group inside side ribbon");
}

#[then("Modify is unchecked and its panel is absent from the ribbon")]
fn unchecked_modify_hidden(world: &mut ApplicationWorld) {
    side_modify_unchecked(world);
    side_modify_hidden(world);
}

#[then("the panel selector remains open with Modify unchecked")]
fn side_modify_unchecked(world: &mut ApplicationWorld) {
    assert!(popup_depth() > 0, "expected side selector popup");
    assert!(!world.checked("Modify"));
}

#[then("the panel selector remains open with Modify checked")]
fn side_modify_checked(world: &mut ApplicationWorld) {
    assert!(popup_depth() > 0, "expected side selector popup");
    assert!(world.checked("Modify"));
}

pub(super) fn popup_depth() -> i32 {
    let context = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.expect("expected GUI");
    context.OpenPopupStack.Size
}

#[given("the 3D Model ribbon is docked on the left with Modify visible")]
#[given("the 3D Model ribbon is docked on the left with the Modify panel visible")]
fn dock_left_with_modify(world: &mut ApplicationWorld) {
    dock_side(world, "Left");
    side_group_y(world, "Modify");
}

#[given("the 3D Model ribbon is docked on the right with Modify visible")]
fn dock_right_with_modify(world: &mut ApplicationWorld) {
    dock_side(world, "Right");
    side_group_y(world, "Modify");
}

#[then("the panel checklist opens with Modify checked")]
fn checklist_checked(world: &mut ApplicationWorld) {
    side_modify_checked(world);
}

#[then("the checklist remains open")]
fn checklist_open(_: &mut ApplicationWorld) {
    assert!(popup_depth() > 0, "expected side selector popup");
}

#[when("the user clicks the direct panel selector again")]
#[when("the user clicks the direct panel selector arrow again")]
fn close_side_selector(world: &mut ApplicationWorld) {
    click_side_selector(world);
}

#[then("the checklist closes and the ribbon remains docked on the right")]
fn checklist_closed_right(_: &mut ApplicationWorld) {
    assert_eq!(popup_depth(), 0, "expected selector popup closed");
    let (position, _) = window_geometry("##ribbon-column").expect("expected side ribbon");
    let (_, band) = window_geometry("##application-ribbon").expect("expected ribbon band");
    assert!(position[0] > band[0] / 2., "expected ribbon on right");
}

fn dock_side(world: &mut ApplicationWorld, side: &str) {
    open_context_menu(world);
    world.click_text("Docking Positions");
    world.click_text(side);
    world.pump(Vec::new());
}

#[test]
fn side_selector_toggles_panel_on_both_sides() {
    for side in ["Left", "Right"] {
        let mut world = ApplicationWorld::new();
        world.add_part();
        dock_side(&mut world, side);
        click_side_selector(&mut world);
        assert!(popup_depth() > 0, "{side}: expected selector popup");
        assert!(world.checked("Modify"));
        world.click_text("Modify");
        assert!(!world.checked("Modify"));
        assert!(popup_depth() > 0, "{side}: expected popup to remain open");
        world.click_text("Modify");
        assert!(world.checked("Modify"));
        click_side_selector(&mut world);
        assert_eq!(
            popup_depth(),
            0,
            "{side}: expected second click to close popup"
        );
        click_side_selector(&mut world);
        assert!(popup_depth() > 0, "{side}: expected popup to reopen");
        world.escape();
        assert_eq!(popup_depth(), 0, "{side}: expected popup closed");
        world.close();
    }
}

#[test]
fn side_selector_preserves_open_group_until_escape() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    dock_side(&mut world, "Left");
    world.click_text("Modify");
    world.expect_text("Fillet");
    click_side_selector(&mut world);
    assert!(popup_depth() > 0, "expected selector popup");
    world.expect_text("Fillet");
    world.escape();
    assert_eq!(popup_depth(), 0, "expected selector popup closed");
    world.expect_no_text("Fillet");
    world.close();
}

#[test]
fn hiding_an_open_side_group_closes_its_commands() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    dock_side(&mut world, "Left");
    world.click_text("Modify");
    world.expect_text("Fillet");
    click_side_selector(&mut world);
    world.click_text("Modify");
    assert!(!world.checked("Modify"));
    world.expect_no_text("Fillet");
    assert!(popup_depth() > 0, "expected selector popup to remain open");
    world.close();
}
