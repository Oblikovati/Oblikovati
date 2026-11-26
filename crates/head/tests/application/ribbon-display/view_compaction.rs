// SPDX-License-Identifier: GPL-2.0-only
//! Switch width-compacted View panels with native pointer input.
use super::*;

#[given("a part is active in the MCAD Software with the full View ribbon displayed")]
fn part_on_view(world: &mut ApplicationWorld) {
    super::super::workspace_layout::part_open(world);
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

#[given("the narrow window shows Windows as a compact panel button")]
fn narrow_windows_panel(world: &mut ApplicationWorld) {
    super::overflow::view_in_overflow(world);
    world.expect_text("Windows");
    world.expect_no_text("User Interface");
}

pub(super) fn click_compact_panel(world: &mut ApplicationWorld, name: &str) {
    super::super::desktop_input::click(world, compact_panel_point(world, name));
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
#[given("the temporary Windows panel is open")]
fn open_windows_panel(world: &mut ApplicationWorld) {
    click_windows(world);
    windows_popup(world);
}
#[then("the Appearance panel closes and the temporary Windows panel opens")]
#[then("the Windows panel remains open")]
fn windows_popup(world: &mut ApplicationWorld) {
    world.expect_no_text("Theme");
    world.expect_text("User Interface");
    assert!(world.run("User Interface").minimum[1] > band_height());
}
#[then("the Windows panel closes")]
#[then("the temporary Windows panel remains closed")]
fn windows_popup_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("User Interface");
}

#[then("the Tools ribbon groups are displayed")]
fn tools_groups_visible(world: &mut ApplicationWorld) {
    world.save_capture("tools-after-closing-view");
    let row = super::super::diagnostics::child_scroll("##application-ribbon", "##ribbon-panels")
        .expect("expected the ribbon command row");
    for name in ["Measure", "Material and Appearance"] {
        confirm_tools_group(world, row.list, name);
    }
    assert_eq!(
        ribbon_tab_underlines(world, ["Sketch", "Tools"]),
        [false, true]
    );
}

fn confirm_tools_group(world: &mut ApplicationWorld, list: usize, name: &str) {
    use super::super::{desktop_input, diagnostics, world::squeezed};
    let expected = squeezed(name);
    let group = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.list == list
                && (diagnostics::panel_caption_matches(&run.text, name)
                    || run
                        .text
                        .strip_suffix("...")
                        .is_some_and(|prefix| !prefix.is_empty() && expected.starts_with(prefix)))
        })
        .unwrap_or_else(|| {
            panic!(
                "expected {name} in the visible Tools command row: {:?}",
                world.text()
            )
        });
    if group.text.ends_with("...") {
        desktop_input::hover(world, diagnostics::pointer_pixels(group.center()));
        world.expect_text(&format!("Show {name} commands"));
    }
}

#[then("the View ribbon groups are displayed")]
fn view_groups_visible(world: &mut ApplicationWorld) {
    super::overflow::view_groups_shown(world);
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
