// SPDX-License-Identifier: GPL-2.0-only
//! Common acceptance and help steps for Part Create properties.
use super::*;

#[then("OK and Help are available")]
fn help_and_ok(world: &mut ApplicationWorld) {
    let tool = active_tool();
    assert!(world.run_is_enabled("OK"));
    assert!(world.run_is_enabled("Help"));
    click_button(world, tool, "Help");
    let expected = if tool == "Loft" {
        "Selecteachprofileorpoint"
    } else {
        "Startthepath"
    };
    assert!(super::super::world::squeezed(&world.window_text(tool)).contains(expected));
}

#[when("OK is clicked")]
pub(super) fn accept(world: &mut ApplicationWorld) {
    click_button(world, active_tool(), "OK");
}

fn click_button(world: &mut ApplicationWorld, tool: &str, label: &str) {
    let pixel =
        super::super::diagnostics::pointer_pixels(world.run_in_window(tool, label).center());
    world.click(pixel);
    world.pump(Vec::new());
}

fn active_tool() -> &'static str {
    ["Extrude", "Loft", "Sweep"]
        .into_iter()
        .find(|tool| super::super::diagnostics::window_geometry(tool).is_some())
        .expect("expected a Part Create properties window")
}
