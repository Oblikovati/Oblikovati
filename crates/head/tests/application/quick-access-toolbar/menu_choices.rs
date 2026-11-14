// SPDX-License-Identifier: GPL-2.0-only
//! Check the standard quick access customization choices.
use super::*;

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
fn menu_closed(_world: &mut ApplicationWorld) {
    assert!(window_geometry("##quick-access-customize").is_none());
}

#[then("the command check states remain unchanged")]
fn choices_unchanged(world: &mut ApplicationWorld) {
    open_customization(world);
    defaults_checked(world);
    common_choices_checked(world);
    optional_choices_unchecked(world);
    world.escape();
}
