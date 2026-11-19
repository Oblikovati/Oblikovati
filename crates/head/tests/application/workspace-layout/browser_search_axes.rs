// SPDX-License-Identifier: GPL-2.0-only
//! Verify browser search across capitalization and clearing (#3897).
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[given("its model browser contains X Axis, Y Axis and Z Axis under Origin")]
fn origin_axes_present(world: &mut ApplicationWorld) {
    for label in ["Origin", "X Axis", "Y Axis", "Z Axis"] {
        world.expect_text(label);
    }
}

#[when(regex = r#"^the user opens browser search and enters "([^"]*)"$"#)]
fn search_for(world: &mut ApplicationWorld, query: String) {
    world.click_text("Search");
    world.click_text("Type Keyword");
    world.type_text(&query);
}

#[then("the browser displays X Axis, Y Axis and Z Axis under Origin")]
fn matching_axes(world: &mut ApplicationWorld) {
    origin_axes_present(world);
    world.expect_no_text("XY Plane");
    world.expect_no_text("Center Point");
}

#[when("the user clears the search and closes the empty search field")]
fn clear_and_close(world: &mut ApplicationWorld) {
    world.click_text("x");
    world.click_text("x");
}

#[then("the unfiltered browser tree is displayed")]
fn unfiltered_tree(world: &mut ApplicationWorld) {
    for label in ["Origin", "XY Plane", "X Axis", "Center Point"] {
        world.expect_text(label);
    }
}
