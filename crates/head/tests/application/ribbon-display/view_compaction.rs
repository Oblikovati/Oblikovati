// SPDX-License-Identifier: GPL-2.0-only
//! Switch width-compacted View panels with native pointer input.
use super::*;

#[given("a part is active in the MCAD Software with the full View ribbon displayed")]
fn part_on_view(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
    world.click_text("View");
    world.expect_text("User Interface");
}

#[given("the narrow window shows Appearance and Windows as compact panel buttons")]
fn narrow_view_panels(world: &mut ApplicationWorld) {
    super::window_compaction::narrow_window(world);
    world.expect_text("Appearance");
    world.expect_text("Windows");
    world.expect_no_text("User Interface");
}

fn compact_panel_point(world: &ApplicationWorld, name: &str) -> [i32; 2] {
    let label = world.run(name).center();
    super::super::diagnostics::pointer_pixels([label[0], label[1] - 35.])
}
fn click_compact_panel(world: &mut ApplicationWorld, name: &str) {
    super::super::x11_input::click(compact_panel_point(world, name));
    world.pump(Vec::new());
}

#[when("the user clicks the Appearance panel button")]
fn click_appearance(world: &mut ApplicationWorld) {
    click_compact_panel(world, "Appearance");
}
#[then("the temporary Appearance panel opens below the ribbon")]
fn appearance_popup(world: &mut ApplicationWorld) {
    world.expect_text("Theme");
    assert!(world.run("Theme").minimum[1] > band_height());
}
#[when("the user clicks the Windows panel button")]
#[when("the user clicks the Windows panel button again")]
fn click_windows(world: &mut ApplicationWorld) {
    click_compact_panel(world, "Windows");
}
#[then("the Appearance panel closes and the temporary Windows panel opens")]
#[then("the Windows panel remains open")]
fn windows_popup(world: &mut ApplicationWorld) {
    world.expect_no_text("Theme");
    world.expect_text("User Interface");
    assert!(world.run("User Interface").minimum[1] > band_height());
}
#[then("the Windows panel closes")]
fn windows_popup_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("User Interface");
}

#[test]
#[ignore = "requires an X11 window manager"]
fn dragging_off_another_compact_panel_does_not_open_it() {
    let mut world = ApplicationWorld::new();
    part_on_view(&mut world);
    narrow_view_panels(&mut world);
    click_appearance(&mut world);
    appearance_popup(&mut world);
    let at = compact_panel_point(&world, "Windows");
    super::super::x11_input::left_button(at, true);
    world.pump(Vec::new());
    super::super::x11_input::move_pointer([600, 600]);
    world.pump(Vec::new());
    super::super::x11_input::left_button([600, 600], false);
    world.pump(Vec::new());
    windows_popup_closed(&mut world);
    world.close();
}
