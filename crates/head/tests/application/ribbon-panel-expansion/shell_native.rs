// SPDX-License-Identifier: GPL-2.0-only
//! Native Shell ribbon and preview refusal interaction (#4106, Martensite #554).
use super::{ApplicationWorld, diagnostics};

#[test]
fn x11_shell_properties_fit_and_cancel_at_minimum_size() {
    use super::super::x11_input;
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    let features = world.state().browser.unwrap().features.len();
    click_label(&mut world, "Shell");
    x11_input::resize_application_window([640, 480]);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent(), [640, 480]);
    world.save_capture("shell-at-minimum-size");
    assert_shell_properties_fit(&mut world);
    let cancel = world.run_in_window("Shell", "Cancel").center();
    x11_input::click(diagnostics::pointer_pixels(cancel));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Shell").is_none());
    assert_eq!(world.state().browser.unwrap().features.len(), features);
    world.close();
}

fn assert_shell_properties_fit(world: &mut ApplicationWorld) {
    let panel = diagnostics::window_pixel_bounds("Shell").expect("visible Shell properties");
    let viewport = world.viewport_bounds().map(|edge| edge as f32);
    assert!(panel[0] >= viewport[0] && panel[1] >= viewport[1]);
    assert!(panel[2] <= viewport[2] && panel[3] <= viewport[3]);
    let thickness = world.run_in_window("Shell", "Thickness");
    let value = world.run_in_window("Shell", "1mm");
    assert!(
        thickness.maximum[0] < value.minimum[0],
        "Thickness label must lead the field"
    );
    let cancel = world.run_in_window("Shell", "Cancel");
    assert!(cancel.maximum[1] < panel[3] && panel[3] - cancel.maximum[1] < 40.);
    assert!(!world.run_is_enabled("OK"));
}

#[test]
fn x11_shell_opens_on_a_selected_face_reports_refusal_and_cancels() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    let face = super::super::ribbon_keyboard_navigation::visible_face_point(&mut world);
    super::super::x11_input::click(face);
    world.pump(Vec::new());
    assert!(world.state().model_face_chosen);
    let shell = diagnostics::pointer_pixels(world.run("Shell").center());
    super::super::x11_input::click(shell);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Shell").is_some());
    world.expect_text("Thickness");
    assert!(
        world.text().to_lowercase().contains("tolerance"),
        "expected the kernel refusal, got {}",
        world.text()
    );
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Shell").is_none());
    assert!(world.state().model_face_chosen);
    world.close();
}

#[test]
fn x11_shell_moves_to_expanded_and_back_across_tab_switch() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let shell = diagnostics::pointer_pixels(world.run("Shell").center());
    super::super::x11_input::right_click(shell);
    world.pump(Vec::new());
    let row = diagnostics::pointer_pixels(world.run("Move to Expanded Panel").center());
    super::super::x11_input::click(row);
    world.pump(Vec::new());
    world.expect_no_text("Shell");
    click_label(&mut world, "Tools");
    click_label(&mut world, "3D Model");
    world.expect_no_text("Shell");
    super::click_expansion_arrow(&mut world, "Modify");
    let expanded = "##ribbon-additional-PartModify";
    let shell = diagnostics::pointer_pixels(world.run_in_window(expanded, "Shell").center());
    super::super::x11_input::right_click(shell);
    world.pump(Vec::new());
    let row = diagnostics::pointer_pixels(world.run("Move to Main Panel").center());
    super::super::x11_input::click(row);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(expanded).is_none());
    world.expect_text("Shell");
    world.close();
}

fn click_label(world: &mut ApplicationWorld, label: &str) {
    let at = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(at);
    world.pump(Vec::new());
}

#[test]
fn x11_shell_from_expanded_panel_reports_kernel_refusal_and_cancels() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    let face = super::super::ribbon_keyboard_navigation::visible_face_point(&mut world);
    super::super::x11_input::click(face);
    world.pump(Vec::new());
    assert!(world.state().model_face_chosen);
    let shell = diagnostics::pointer_pixels(world.run("Shell").center());
    super::super::x11_input::right_click(shell);
    world.pump(Vec::new());
    click_label(&mut world, "Move to Expanded Panel");
    super::click_expansion_arrow(&mut world, "Modify");
    let expanded = "##ribbon-additional-PartModify";
    let shell = diagnostics::pointer_pixels(world.run_in_window(expanded, "Shell").center());
    super::super::x11_input::click(shell);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Shell").is_some());
    assert!(
        world.text().to_lowercase().contains("tolerance"),
        "expected the kernel refusal, got {}",
        world.text()
    );
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Shell").is_none());
    assert!(world.state().model_face_chosen);
    world.close();
}
