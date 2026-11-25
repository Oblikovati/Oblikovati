// SPDX-License-Identifier: GPL-2.0-only
//! Verify staged display settings and their effect on the actual canvas menu.
use super::super::*;
use super::open_marking_menu_tab;
use oblikovati_model::marking_menu::{MarkingMenuOverflow, MarkingMenuStyle};

#[given("Marking Menu Style is \"Show Icon and Text\"")]
#[then("the Marking Menu Style is \"Show Icon and Text\"")]
#[then("Marking Menu Style is \"Show Icon and Text\"")]
fn default_style(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "ShowIconandText");
    assert_eq!(
        world.state().marking_menu_display.style,
        MarkingMenuStyle::IconAndText
    );
}

#[given("Overflow Menu is \"Full Menu\"")]
#[then("the Overflow Menu is \"Full Menu\"")]
#[then("Overflow Menu is \"Full Menu\"")]
fn default_overflow(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "FullMenu");
    assert_eq!(
        world.state().marking_menu_display.overflow,
        MarkingMenuOverflow::Full
    );
}

#[when("the user opens the Marking Menu Style selector")]
fn show_styles(world: &mut ApplicationWorld) {
    click_in_customize(world, "ShowIconandText");
}

#[when("the user opens the Overflow Menu selector")]
fn show_overflow(world: &mut ApplicationWorld) {
    click_in_customize(world, "FullMenu");
}

#[then(
    "the available styles include \"Show Icon and Text\", \"Show Icon Only\" and \"Show Text Only\""
)]
fn styles_visible(world: &mut ApplicationWorld) {
    choices_visible(world, ["ShowIconandText", "ShowIconOnly", "ShowTextOnly"]);
}

#[then(
    "the available overflow choices include \"Full Menu\", \"Radial Menu Only\" and \"Short Menu\""
)]
fn overflow_visible(world: &mut ApplicationWorld) {
    choices_visible(world, ["FullMenu", "RadialMenuOnly", "ShortMenu"]);
}

fn choices_visible(world: &ApplicationWorld, labels: [&str; 3]) {
    let list = world.run(labels[1]).list;
    for name in labels {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.list == list && run.text == name),
            "expected {name} in the open selector"
        );
    }
}

#[then("\"Show Icon and Text\" remains selected")]
fn default_style_selected(world: &mut ApplicationWorld) {
    world.hover([1250, 780]);
    assert!(world.row_accented_beside("Show Icon and Text", "Show Icon Only"));
}

#[then("\"Full Menu\" remains selected")]
fn full_overflow_selected(world: &mut ApplicationWorld) {
    world.hover([1250, 780]);
    assert!(world.row_accented_beside("Full Menu", "Radial Menu Only"));
}

#[when("the user dismisses the selector without choosing an option")]
fn dismiss_selector(world: &mut ApplicationWorld) {
    world.escape();
    assert!(diagnostics::window_geometry("Customize").is_some());
    for choice in [
        "Show Icon Only",
        "Show Text Only",
        "Radial Menu Only",
        "Short Menu",
        "Sheet Metal",
        "Constraint Glyphs",
        "Projected Curves",
    ] {
        world.expect_no_text(choice);
    }
}

#[test]
fn marking_presentation_changes_the_canvas_menu_and_reloads() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_marking_menu_tab(&mut world);
    show_styles(&mut world);
    world.click_text("Show Icon Only");
    show_overflow(&mut world);
    world.click_text("Radial Menu Only");
    default_live_presentation(&mut world);
    click_in_customize(&mut world, "Apply");
    chosen_live_presentation(&mut world);
    close_customization_button(&mut world);
    world.right_click([800, 430]);
    assert!(diagnostics::command_popup_open());
    world.expect_no_text("Look At");
    assert!(!world.frame.runs.iter().any(|run| {
        run.center()[1] > 350. && ["Extrude", "Fillet", "OffsetPlane"].contains(&run.text.as_str())
    }));
    let icons = super::super::super::ribbon_appearance::icon_quads::icon_quads_inside((
        [700., 340.],
        [1200., 750.],
    ));
    assert!(
        icons.len() >= 3,
        "expected the radial command icons: {icons:?}"
    );
    world.reopen_application();
    chosen_live_presentation(&mut world);
    world.close();
}

fn default_live_presentation(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().marking_menu_display.style,
        MarkingMenuStyle::IconAndText
    );
    assert_eq!(
        world.state().marking_menu_display.overflow,
        MarkingMenuOverflow::Full
    );
}

fn chosen_live_presentation(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().marking_menu_display.style,
        MarkingMenuStyle::IconOnly
    );
    assert_eq!(
        world.state().marking_menu_display.overflow,
        MarkingMenuOverflow::RadialOnly
    );
}
