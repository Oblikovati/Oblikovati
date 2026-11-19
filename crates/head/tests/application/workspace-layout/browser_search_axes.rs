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

#[given("a part with origin axes and planes is active in the MCAD Software")]
fn part_with_origin(world: &mut ApplicationWorld) {
    super::browser_feature::open_visible_extruded_part(world);
    origin_axes_present(world);
}

#[given("the browser search field contains \"axis\"")]
fn field_contains_axis(world: &mut ApplicationWorld) {
    search_for(world, "axis".into());
    world.expect_text("axis");
}

#[given("X Axis, Y Axis and Z Axis are displayed under Origin")]
fn filtered_axes_present(world: &mut ApplicationWorld) {
    matching_axes(world);
}

#[when(regex = r#"^the user selects all search text and replaces it with "([^"]*)"$"#)]
fn replace_search(world: &mut ApplicationWorld, query: String) {
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(&query);
}

#[then("the browser displays a no-results message instead of the axis rows")]
fn no_matching_axes(world: &mut ApplicationWorld) {
    world.expect_text("No matching entries");
    world.expect_no_text("X Axis");
    world.expect_no_text("Y Axis");
    world.expect_no_text("Z Axis");
}

#[then("the browser displays YZ Plane, XZ Plane and XY Plane under Origin")]
fn matching_planes(world: &mut ApplicationWorld) {
    for label in ["Origin", "YZ Plane", "XZ Plane", "XY Plane"] {
        world.expect_text(label);
    }
    world.expect_no_text("X Axis");
}

#[when("the user clears the query and closes the empty search field")]
fn clear_query_and_close(world: &mut ApplicationWorld) {
    clear_and_close(world);
}
