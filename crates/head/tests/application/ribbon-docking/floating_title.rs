// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{given, then, when};

#[given("the ribbon is floating")]
fn floating(world: &mut ApplicationWorld) {
    undock_ribbon(world);
    expect_floating(world);
}
#[when("the user right-clicks the floating ribbon title bar")]
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
fn dock_choices(world: &mut ApplicationWorld) {
    for label in ["Dock to Top", "Dock to Left", "Dock to Right"] {
        world.expect_text(label);
    }
}
#[when("the user selects Dock to Top")]
fn select_top(world: &mut ApplicationWorld) {
    world.click_text("Dock to Top");
    world.pump(Vec::new());
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
