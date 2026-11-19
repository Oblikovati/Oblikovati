// SPDX-License-Identifier: GPL-2.0-only
//! Native Shell ribbon and preview refusal interaction (#4106, Martensite #554).
use super::{ApplicationWorld, diagnostics};

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
