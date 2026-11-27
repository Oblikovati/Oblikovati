// SPDX-License-Identifier: GPL-2.0-only
//! Selected-tab behavior through native X11 pointer events (#3856).
use super::{ApplicationWorld, click_expansion_arrow, diagnostics};

#[test]
fn selected_tab_keeps_then_closes_temporary_expansion() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::super::ribbon_display::click_display_menu(&mut world);
    world.click_text("Minimize to Tabs");
    world.click_text("3D Model");
    click_expansion_arrow(&mut world, "Modify");
    let tab = diagnostics::pointer_pixels(world.run("3D Model").center());
    super::super::x11_input::click(tab);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    super::super::x11_input::click(tab);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.close();
}

#[test]
fn temporary_modify_footer_hides_pin_and_dismisses_on_outside_click() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    native_click_text(&mut world, "Modify");
    let area = "##ribbon-additional-PartModify";
    assert!(world.window_text(area).contains("Pin"));
    super::super::ribbon_display::click_display_menu(&mut world);
    native_click_text(&mut world, "Minimize to Tabs");
    native_click_text(&mut world, "3D Model");
    native_click_text(&mut world, "Modify");
    let footer = world.window_text(area);
    assert!(footer.contains("Modify"));
    assert!(
        !footer.contains("Pin"),
        "temporary footer showed Pin: {footer}"
    );
    for command in ["Move Bodies", "Bend Part", "Copy Object"] {
        world.expect_text(command);
    }
    super::super::x11_input::click([1000, 700]);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(area).is_none());
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.close();
}

fn native_click_text(world: &mut ApplicationWorld, label: &str) {
    let position = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(position);
    world.pump(Vec::new());
}
