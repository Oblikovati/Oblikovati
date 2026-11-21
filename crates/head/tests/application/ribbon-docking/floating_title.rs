// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{given, then, when};

#[given("the ribbon is floating")]
fn floating(world: &mut ApplicationWorld) {
    undock_ribbon(world);
    expect_floating(world);
}
#[given("a part is active and the ribbon is floating")]
fn floating_part(world: &mut ApplicationWorld) {
    part_active(world);
    floating(world);
}
#[when("the user right-clicks the active 3D Model tab and chooses Undock Ribbon")]
#[when("the user undocks the ribbon")]
fn undock_from_model_tab(world: &mut ApplicationWorld) {
    super::active_tab::right_click_model(world);
    choose_undock(world);
}
#[when("the user right-clicks the floating ribbon title bar")]
#[when("the user right-clicks the floating ribbon title")]
#[when("the user opens the floating title menu again")]
fn open_title_menu(world: &mut ApplicationWorld) {
    let (position, _) = window_geometry(FLOATING).expect("expected a floating ribbon");
    world.right_click(super::super::diagnostics::pointer_pixels([
        position[0] + 60.,
        position[1] + 8.,
    ]));
    world.pump(Vec::new());
}
#[then("a menu offers \"Dock to Top\", \"Dock to Left\", and \"Dock to Right\"")]
#[then("the title menu offers Dock to Top, Dock to Left, and Dock to Right")]
#[then("the context menu offers Dock to Top, Dock to Left and docks to Right")]
#[then("the menu offers Dock to Top, Dock to Left and docks to Right")]
fn dock_choices(world: &mut ApplicationWorld) {
    for label in ["Dock to Top", "Dock to Left", "Dock to Right"] {
        world.expect_text(label);
    }
}

#[then("it does not offer Undock Ribbon")]
fn floating_title_omits_undock(world: &mut ApplicationWorld) {
    world.expect_no_text("Undock Ribbon");
}
#[when("the user selects Dock to Top")]
#[when("the user chooses Dock to Top")]
#[when("chooses Dock to Top")]
fn select_top(world: &mut ApplicationWorld) {
    world.click_text("Dock to Top");
    world.pump(Vec::new());
}

#[when(regex = "^chooses Dock to (Left|Right) from the floating title menu$")]
fn choose_title_dock(world: &mut ApplicationWorld, side: String) {
    open_title_menu(world);
    world.click_text(&format!("Dock to {side}"));
    world.pump(Vec::new());
}

#[when("the user docks the floating ribbon at the top")]
fn restore_from_floating_title(world: &mut ApplicationWorld) {
    open_title_menu(world);
    select_top(world);
}
#[then("the ribbon is docked horizontally above the viewport")]
#[then("the full ribbon is docked horizontally above the workspace")]
fn ribbon_top(world: &mut ApplicationWorld) {
    expect_top_dock(world);
    same_part(world);
}

#[test]
fn title_menu_docks_to_each_edge() {
    let mut world = ApplicationWorld::new();
    world.previous_document = Some(world.add_part());
    for (label, side) in [
        ("Dock to Left", Some(false)),
        ("Dock to Right", Some(true)),
        ("Dock to Top", None),
    ] {
        undock_ribbon(&mut world);
        open_title_menu(&mut world);
        world.click_text(label);
        world.pump(Vec::new());
        match side {
            Some(right) => {
                let (position, _) =
                    window_geometry(COLUMN).expect("expected a docked ribbon column");
                let width = world.extent()[0] as f32;
                assert_eq!(
                    position[0] > width / 2.,
                    right,
                    "unexpected column at {position:?}"
                );
            }
            None => expect_top_dock(&mut world),
        }
        same_part(&mut world);
    }
    world.close();
}

#[then("the title menu closes")]
fn title_menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Dock to Top");
    world.expect_no_text("Dock to Left");
    world.expect_no_text("Dock to Right");
}

#[cfg(target_os = "linux")]
#[test]
fn x11_floating_title_docks_right_and_the_active_tab_restores_top() {
    use super::super::{diagnostics, x11_input};
    let mut world = ApplicationWorld::new();
    super::part_active(&mut world);
    native_ribbon_right_click_tab(&mut world);
    native_ribbon_click(&mut world, "Undock Ribbon");
    expect_floating(&mut world);
    let (origin, _) = window_geometry(FLOATING).unwrap();
    x11_input::right_click(diagnostics::pointer_pixels([
        origin[0] + 60.,
        origin[1] + 8.,
    ]));
    world.pump(Vec::new());
    native_ribbon_click(&mut world, "Dock to Right");
    ribbon_on_right(&mut world);
    native_ribbon_right_click_tab(&mut world);
    native_ribbon_click(&mut world, "Docking Positions");
    assert!(world.checked("Right"));
    native_ribbon_click(&mut world, "Top");
    expect_top_dock(&mut world);
    same_part(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
fn native_ribbon_right_click_tab(world: &mut ApplicationWorld) {
    let point = super::super::diagnostics::pointer_pixels(world.run("3D Model").center());
    super::super::x11_input::right_click(point);
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
fn native_ribbon_click(world: &mut ApplicationWorld, caption: &str) {
    let point = super::super::diagnostics::pointer_pixels(world.run(caption).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}
