// SPDX-License-Identifier: GPL-2.0-only
//! Check the standard quick access customization choices.
use super::*;

const CHOICES: [&str; 22] = [
    "New",
    "Open",
    "Save",
    "Undo",
    "Redo",
    "Home",
    "Projects",
    "Update",
    "Select",
    "Refine Appearance",
    "Material Browser",
    "Materials",
    "Appearance Browser",
    "Appearance Override",
    "Adjust",
    "Clear",
    "Parameters",
    "Design Doctor",
    "Print",
    "iProperties",
    "Return",
    "Measure",
];

fn read_choices(world: &ApplicationWorld, labels: &[&str]) -> Vec<(String, bool)> {
    labels
        .iter()
        .map(|label| {
            (
                (*label).to_owned(),
                world.checked_run(menu_choice(world, label)),
            )
        })
        .collect()
}

fn menu_choice<'a>(
    world: &'a ApplicationWorld,
    label: &str,
) -> &'a super::super::diagnostics::RenderedRun {
    let expected = super::super::world::squeezed(label);
    let menu = world.run("Show Below the Ribbon").list;
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.list == menu
                && (run.text == expected
                // The text decoder can omit the narrow leading glyph in iProperties.
                || (label == "iProperties" && run.text == "Properties"))
        })
        .unwrap_or_else(|| {
            panic!(
                "expected quick-access choice {label:?} in {:?}",
                world.text()
            )
        })
}

#[then(
    "the menu lists New, Open, Save, Undo, Redo, Home, Projects, Update, Select, Refine Appearance, Material Browser, Materials, Appearance Browser, Appearance Override, Adjust, Clear, Parameters, Design Doctor, Print, iProperties, Return, and Measure"
)]
fn lists_current_choices(world: &mut ApplicationWorld) {
    for label in CHOICES {
        menu_choice(world, label);
    }
}

#[then("each listed command displays its current checked or unchecked state")]
fn capture_current_choices(world: &mut ApplicationWorld) {
    world.observed_quick_access = Some(read_choices(world, &CHOICES));
}

#[then(
    "Projects, Update, Select, Refine Appearance, Material Browser, Materials, Appearance Browser, Appearance Override, Adjust, Clear, Parameters, and Design Doctor are checked"
)]
fn common_choices_checked(world: &mut ApplicationWorld) {
    for label in [
        "Projects",
        "Update",
        "Select",
        "Refine Appearance",
        "Material Browser",
        "Materials",
        "Appearance Browser",
        "Appearance Override",
        "Adjust",
        "Clear",
        "Parameters",
        "Design Doctor",
    ] {
        world.expect_text(label);
        assert!(world.checked(label), "expected {label} checked");
    }
}

#[then("Print, iProperties, Return, and Measure are unchecked")]
fn optional_choices_unchecked(world: &mut ApplicationWorld) {
    for label in ["Print", "iProperties", "Return", "Measure"] {
        world.expect_text(label);
        assert!(!world.checked(label), "expected {label} unchecked");
    }
}

#[then("the menu offers Show Below the Ribbon")]
fn below_offered(world: &mut ApplicationWorld) {
    world.expect_text("Show Below the Ribbon");
}

#[then("the customization menu closes")]
fn menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Below the Ribbon");
    world.expect_no_text("Show Above the Ribbon");
}

#[then("the command check states remain unchanged")]
fn choices_unchanged(world: &mut ApplicationWorld) {
    open_customization(world);
    let before = world
        .observed_quick_access
        .as_ref()
        .expect("expected captured quick-access choices");
    assert_eq!(*before, read_choices(world, &CHOICES));
    world.escape();
}

#[test]
fn menu_state_comparison_accepts_a_nondefault_profile() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_customization(&mut world);
    world.click_text("Print");
    open_customization(&mut world);
    let labels = ["New", "Print"];
    let before = read_choices(&world, &labels);
    assert_eq!(before, [("New".into(), true), ("Print".into(), true)]);
    world.escape();
    menu_closed(&mut world);
    open_customization(&mut world);
    assert_eq!(before, read_choices(&world, &labels));
    world.escape();
    world.close();
}

#[test]
fn unimplemented_choices_are_visible_but_cannot_be_added() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_customization(&mut world);
    for label in [
        "Refine Appearance",
        "Appearance Override",
        "Adjust",
        "Design Doctor",
    ] {
        world.expect_text(label);
        assert!(!world.run_is_enabled(label), "{label} must be unavailable");
        assert!(!world.checked(label), "{label} must not be selected");
    }
    world.click_text("Refine Appearance");
    assert!(!world.checked("Refine Appearance"));
    world.escape();
    assert!(!on_row(&world, "Refine Appearance"));
    world.close();
}
