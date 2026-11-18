// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard access to full-width ribbon expansions (#3856).
use super::{ApplicationWorld, diagnostics};
use cucumber::{then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[when("the user presses Alt, then M, then Y")]
fn open_modify_by_keys(world: &mut ApplicationWorld) {
    for key in [
        ViewportPhysicalKey::LeftAlt,
        ViewportPhysicalKey::M,
        ViewportPhysicalKey::Y,
    ] {
        world.key(key, false);
    }
}
#[then("the Modify additional command area opens below the ribbon over the browser area")]
fn keyboard_expansion_placement(_: &mut ApplicationWorld) {
    let (panel, _) =
        diagnostics::window_geometry("##ribbon-additional-PartModify").expect("Modify expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(panel[1] >= ribbon[1] + size[1]);
    let (browser, browser_size) = diagnostics::window_geometry("Model").expect("model browser");
    assert!(
        panel[0] < browser[0] + browser_size[0],
        "expansion {panel:?}, browser {browser:?} {browser_size:?}: expected overlap"
    );
}
#[then("the solid body and model browser remain visible")]
fn body_and_browser(world: &mut ApplicationWorld) {
    super::expect_solid_visible(world);
    world.expect_text("Extrude1");
}
#[then("the additional command area closes and keyboard tips disappear")]
fn expansion_and_tips_close(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    assert!(!world.frame.runs.iter().any(|run| run.text == "HO"));
}

#[cfg(target_os = "linux")]
#[test]
fn x11_keyboard_opens_the_full_modify_expansion_over_the_browser() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    for key in [0xffe9, u32::from(b'm'), u32::from(b'y')] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    keyboard_expansion_placement(&mut world);
    let title = world
        .run_in_window("##ribbon-additional-PartModify", "Modify")
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(title));
    world.pump(Vec::new());
    expansion_and_tips_close(&mut world);
    world.close();
}
