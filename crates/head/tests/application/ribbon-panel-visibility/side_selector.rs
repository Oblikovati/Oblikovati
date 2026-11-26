// SPDX-License-Identifier: GPL-2.0-only
use super::super::diagnostics::{pointer_pixels, window_geometry};
use super::*;
#[path = "selector_help.rs"]
mod selector_help;
#[path = "shell_selector.rs"]
mod shell_selector;

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
#[when("the user clicks the direct panel selector")]
#[when("the user opens the direct panel selector")]
#[when("the user opens the direct panel visibility list")]
#[when("the user opens the direct ribbon panel selector")]
#[given("the direct panel selector is open")]
#[given("the direct panel visibility list is open")]
#[given("the ribbon panel selector is open")]
pub(super) fn open_side_selector(world: &mut ApplicationWorld) {
    click_selector(world);
    assert!(selector_popup_open(), "expected panel selector popup");
}

pub(super) fn click_selector(world: &mut ApplicationWorld) {
    if window_geometry("##ribbon-column").is_some() {
        click_side_selector(world);
    } else {
        let (position, size) = window_geometry("##application-ribbon").unwrap();
        world.click(pointer_pixels([
            position[0] + size[0] - 22.,
            position[1] + 45.,
        ]));
    }
}

pub(super) fn selector_popup_open() -> bool {
    ["##application-ribbon", "##ribbon-column"]
        .into_iter()
        .any(|window| {
            super::super::diagnostics::window_popup_open(window, "##ribbon-panel-selector-list")
        })
}

#[when("I select Modify in the panel selector")]
#[when("the user selects Modify in the panel selector")]
#[when("the user clicks Modify in the checklist")]
#[when("the user clicks Modify in the checklist again")]
fn toggle_selector_modify(world: &mut ApplicationWorld) {
    if window_geometry("##ribbon-column").is_some() {
        world.ribbon_height = side_group_y(world, "Work Features");
    }
    world.click_text("Modify");
}

#[when("I hide Modify through the direct panel selector")]
#[when("the user hides Modify through the direct panel selector")]
fn hide_modify_with_selector(world: &mut ApplicationWorld) {
    open_side_selector(world);
    toggle_selector_modify(world);
}

#[when("I switch to the View ribbon")]
#[when("the user switches to the View ribbon")]
fn switch_to_view(world: &mut ApplicationWorld) {
    world.click_text("View");
}

#[when("I return to the 3D Model ribbon")]
#[when("the user returns to the 3D Model ribbon")]
fn return_to_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}

#[then("Navigate remains visible")]
fn navigate_still_visible(world: &mut ApplicationWorld) {
    side_group_y(world, "Navigate");
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
#[then("the Modify group is visible in the side-docked ribbon")]
#[then("the Modify panel returns to the 3D Model ribbon")]
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
#[then("the Modify panel disappears from the 3D Model ribbon")]
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

#[test]
fn side_selector_home_and_end_keep_keyboard_focus() {
    use oblikovati_viewport::ViewportPhysicalKey as Key;
    for side in ["Left", "Right"] {
        let mut world = ApplicationWorld::new();
        world.add_part();
        dock_side(&mut world, side);
        click_side_selector(&mut world);
        for _ in 0..3 {
            world.key(Key::DownArrow, false);
        }
        for key in [Key::Home, Key::End] {
            world.key(key, false);
            world.key(Key::Space, false);
            assert!(
                !world.checked("Create"),
                "{side} {key:?}: expected Create toggle"
            );
            world.key(Key::Space, false);
            assert!(world.checked("Create"));
        }
        world.close();
    }
}

#[test]
fn side_additional_commands_remain_open_with_selector_and_close_on_escape() {
    for side in ["Left", "Right"] {
        side_additional_escape(side);
    }
}
fn side_additional_escape(side: &str) {
    let mut world = ApplicationWorld::new();
    world.add_part();
    dock_side(&mut world, side);
    world.click_text("Modify");
    open_side_additional(&mut world);
    assert!(window_geometry("##ribbon-additional-PartModify").is_some());
    click_side_selector(&mut world);
    assert!(window_geometry("##ribbon-additional-PartModify").is_some());
    world.save_capture(&format!("side-additional-selector-{side}"));
    world.escape();
    assert!(window_geometry("##ribbon-additional-PartModify").is_none());
    assert_eq!(popup_depth(), 0);
    world.close();
}

#[test]
fn pinned_side_additional_commands_survive_outside_click_and_close_when_hidden() {
    for side in ["Left", "Right"] {
        pinned_side_additional(side);
    }
}

#[test]
fn side_context_menu_restores_group_and_checkmark() {
    for side in ["Left", "Right"] {
        context_menu_round_trip(side);
    }
}
fn context_menu_round_trip(side: &str) {
    let mut world = ApplicationWorld::new();
    world.add_part();
    dock_side(&mut world, side);
    let before = side_group_positions(&world);
    expand_show_panels(&mut world);
    assert!(world.checked("Modify"));
    toggle_modify(&mut world);
    assert_eq!(popup_depth(), 0, "expected context menus closed");
    let hidden = side_group_positions(&world);
    assert_eq!(hidden[0], before[0], "Create stays in place");
    assert!(
        hidden[1..]
            .iter()
            .zip(&before[1..])
            .all(|(now, old)| now < old)
    );
    expand_show_panels(&mut world);
    assert!(!world.checked("Modify"));
    toggle_modify(&mut world);
    assert_eq!(popup_depth(), 0, "expected context menus closed");
    assert_eq!(side_group_positions(&world), before);
    expand_show_panels(&mut world);
    assert!(world.checked("Modify"));
    world.escape();
    world.close();
}

#[test]
fn side_tab_click_dismisses_selector_and_selects_view() {
    for side in ["Left", "Right"] {
        let mut world = ApplicationWorld::new();
        world.add_part();
        dock_side(&mut world, side);
        click_side_selector(&mut world);
        assert!(popup_depth() > 0);
        world.click_text("View");
        assert_eq!(popup_depth(), 0, "{side}: expected selector closed");
        side_group_y(&world, "Navigate");
        world.close();
    }
}

fn side_group_positions(world: &ApplicationWorld) -> [f32; 4] {
    ["Create", "Work Features", "Pattern", "Simplification"].map(|name| side_group_y(world, name))
}
fn pinned_side_additional(side: &str) {
    let mut world = ApplicationWorld::new();
    world.add_part();
    dock_side(&mut world, side);
    world.click_text("Modify");
    open_side_additional(&mut world);
    let pin = world.run("Pin");
    world.click(pointer_pixels([pin.minimum[0] + 6., pin.center()[1]]));
    world.click([1000, 700]);
    assert!(window_geometry("##ribbon-additional-PartModify").is_some());
    click_side_selector(&mut world);
    world.click_text("Modify");
    assert!(window_geometry("##ribbon-additional-PartModify").is_none());
    world.click_text("Modify");
    assert!(window_geometry("##ribbon-additional-PartModify").is_none());
    world.close();
}

pub(in super::super) fn open_side_additional(world: &mut ApplicationWorld) {
    let title = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| {
            run.text.ends_with(['v', '▼'])
                && super::super::diagnostics::panel_caption_matches(&run.text, "Modify")
        })
        .expect("expected the Modify expansion arrow");
    let arrow = [title.maximum[0] - 2., title.center()[1]];
    world.click(pointer_pixels(arrow));
}
