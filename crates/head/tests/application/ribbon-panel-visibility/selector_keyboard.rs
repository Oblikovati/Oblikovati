// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard focus remains on the chosen panel when Home or End is pressed.
use super::super::diagnostics::window_geometry;
use super::*;
use oblikovati_viewport::ViewportPhysicalKey as Key;

fn focus_id() -> u32 {
    let context = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.expect("expected GUI");
    context.NavId
}

#[given("the full top-docked 3D Model ribbon is visible")]
fn top_model_ribbon(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-column").is_none());
    world.expect_text("Start 2D Sketch");
}

#[given("the direct panel selector is open with Create focused and checked")]
fn create_focused(world: &mut ApplicationWorld) {
    let (_, size) = window_geometry("##application-ribbon").unwrap();
    world.click([size[0] as i32 - 22, 45]);
    for _ in 0..3 {
        world.key(Key::DownArrow, false);
    }
    assert!(world.checked("Create"));
    world.key(Key::Space, false);
    assert!(
        !world.checked("Create"),
        "expected focused Create to toggle"
    );
    world.key(Key::Space, false);
    assert!(world.checked("Create"));
    let focused = focus_id();
    assert_ne!(focused, 0, "expected a focused panel row");
    world.panel_selector_focus = Some(focused);
}

#[when("the user presses Home")]
fn press_home(world: &mut ApplicationWorld) {
    world.key(Key::Home, false);
}

#[when("the user presses End")]
fn press_end(world: &mut ApplicationWorld) {
    world.key(Key::End, false);
}

#[then("Create remains focused and checked")]
fn create_still_focused(world: &mut ApplicationWorld) {
    assert_eq!(
        Some(focus_id()),
        world.panel_selector_focus,
        "expected Create focus"
    );
    assert!(world.checked("Create"));
}

#[then("the panel selector remains open")]
fn selector_open(_: &mut ApplicationWorld) {
    assert!(super::side_selector::selector_popup_open());
}

#[then("the panel selector closes")]
fn selector_closed(_: &mut ApplicationWorld) {
    assert!(!super::side_selector::selector_popup_open());
}

#[given("the Primitives panel is hidden")]
fn primitives_hidden(world: &mut ApplicationWorld) {
    assert!(model_panel_run(world, "Primitives").is_none());
}

pub(super) fn model_panel_run<'a>(
    world: &'a ApplicationWorld,
    label: &str,
) -> Option<&'a super::super::diagnostics::RenderedRun> {
    let inside = world.run("Start 2D Sketch").center();
    let list = super::super::diagnostics::child_draw_list_at("##application-ribbon", inside)
        .expect("expected the 3D Model panel draw list");
    ribbon_runs(world, label).find(|run| run.list == list)
}

#[when("the user presses Down twice")]
fn press_down_twice(world: &mut ApplicationWorld) {
    for _ in 0..2 {
        world.key(Key::DownArrow, false);
    }
}

#[then("the Primitives row has keyboard focus and is unchecked")]
fn primitives_focused(world: &mut ApplicationWorld) {
    selector_open(world);
    assert!(!world.checked("Primitives"));
    let focused = focus_id();
    assert_ne!(focused, 0, "expected a focused panel row");
    world.panel_selector_focus = Some(focused);
}

#[then("Primitives is checked and its panel appears in the ribbon")]
fn primitives_shown(world: &mut ApplicationWorld) {
    assert_eq!(Some(focus_id()), world.panel_selector_focus);
    assert!(world.checked("Primitives"));
    assert!(model_panel_run(world, "Primitives").is_some());
    for command in ["Box", "Cylinder", "Sphere", "Torus"] {
        let run = model_panel_run(world, command).expect("expected primitive command");
        assert_ne!(run.alpha, u8::MAX, "expected {command} disabled");
    }
    world.save_capture("primitives-visible");
}

#[then("Primitives is unchecked and its panel disappears")]
fn primitives_hidden_after_toggle(world: &mut ApplicationWorld) {
    if let Some(focused) = world.panel_selector_focus {
        assert_eq!(focus_id(), focused);
    }
    assert!(!world.checked("Primitives"));
    primitives_hidden(world);
}

fn model_selector_rows(world: &ApplicationWorld) -> Vec<&super::super::diagnostics::RenderedRun> {
    assert!(super::side_selector::selector_popup_open());
    let list = world.run("Primitives").list;
    let mut rows: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == list)
        .collect();
    rows.sort_by(|left, right| left.center()[1].total_cmp(&right.center()[1]));
    rows
}

#[given("the Sketch row is the first row and 3D Print is the last row")]
fn capture_model_selector_rows(world: &mut ApplicationWorld) {
    let rows = model_selector_rows(world);
    assert_eq!(rows.first().map(|run| run.text.as_str()), Some("Sketch"));
    assert_eq!(rows.last().map(|run| run.text.as_str()), Some("3DPrint"));
    world.observed_counts = rows
        .into_iter()
        .map(|run| usize::from(world.checked_run(run)))
        .collect();
}

#[when("the user presses Down to focus the Sketch row")]
fn begin_selector_keyboard_navigation(world: &mut ApplicationWorld) {
    world.key(Key::DownArrow, false);
    sketch_focused(world);
}

fn expect_selector_focus(world: &ApplicationWorld, label: &str) {
    let context = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.expect("expected GUI");
    let window = unsafe { context.NavWindow.as_ref() }.expect("expected a focused selector");
    let text = std::ffi::CString::new(label).unwrap();
    let expected = unsafe { imgui::sys::igImHashStr(text.as_ptr(), 0, window.ID) };
    assert!(
        model_selector_rows(world)
            .iter()
            .any(|row| row.text == label.replace(' ', ""))
    );
    assert_eq!(
        context.NavId, expected,
        "expected focus on the {label} checkbox"
    );
}

#[then("keyboard focus moves to the 3D Print row")]
fn print_focused(world: &mut ApplicationWorld) {
    expect_selector_focus(world, "3D Print");
}

#[then("keyboard focus moves to the Sketch row")]
fn sketch_focused(world: &mut ApplicationWorld) {
    expect_selector_focus(world, "Sketch");
}

#[then("no panel checkbox has changed")]
fn selector_checks_unchanged(world: &mut ApplicationWorld) {
    let current: Vec<_> = model_selector_rows(world)
        .into_iter()
        .map(|run| usize::from(world.checked_run(run)))
        .collect();
    assert_eq!(current, world.observed_counts);
}

#[test]
fn panel_selector_wraps_arrow_focus_without_changing_visibility() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::side_selector::open_side_selector(&mut world);
    capture_model_selector_rows(&mut world);
    begin_selector_keyboard_navigation(&mut world);
    world.key(Key::UpArrow, false);
    print_focused(&mut world);
    world.key(Key::DownArrow, false);
    sketch_focused(&mut world);
    selector_checks_unchanged(&mut world);
    world.close();
}
