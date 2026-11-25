// SPDX-License-Identifier: GPL-2.0-only
//! Verify browser search across capitalization and clearing (#3897).
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[given("its model browser contains X Axis, Y Axis and Z Axis under Origin")]
#[given("the model browser contains X Axis, Y Axis and Z Axis under Origin")]
fn origin_axes_present(world: &mut ApplicationWorld) {
    for label in ["Origin", "X Axis", "Y Axis", "Z Axis"] {
        world.expect_text(label);
    }
}

#[when(regex = r#"^the user opens browser search and enters "(axis|AXIS|origin)"$"#)]
fn search_for(world: &mut ApplicationWorld, query: String) {
    super::dock_search::click_model_search(world);
    world.click_text("Type Keyword");
    world.type_text(&query);
}

#[then("the browser displays X Axis, Y Axis and Z Axis under Origin")]
#[then("the browser shows the three matching axes under Origin")]
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
#[then("the unfiltered browser tree returns")]
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

#[then("the browser shows Origin as a matching collapsed row")]
fn matching_origin_collapsed(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_no_text("XY Plane");
    world.expect_no_text("Center Point");
}

#[then("the Origin text is highlighted")]
fn origin_match_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.text_is_selected("Origin"),
        "expected the Origin match highlight"
    );
}

#[when("the user expands the matching Origin row")]
fn expand_origin_match(world: &mut ApplicationWorld) {
    world.press_disclosure("Origin");
}

#[then("its plane, axis and center-point children are displayed")]
#[then("the Origin children are visible")]
fn origin_children(world: &mut ApplicationWorld) {
    for label in [
        "YZ Plane",
        "XZ Plane",
        "XY Plane",
        "X Axis",
        "Y Axis",
        "Z Axis",
        "Center Point",
    ] {
        world.expect_text(label);
    }
}

#[then("the child labels appear dimmed while the search remains active")]
fn origin_children_dimmed(world: &mut ApplicationWorld) {
    let parent = world.run("Origin");
    let child = world.run("XY Plane");
    let brightness = |run: &super::super::diagnostics::RenderedRun| {
        run.color.iter().map(|value| u32::from(*value)).sum::<u32>() * u32::from(run.alpha)
    };
    assert!(
        brightness(child) < brightness(parent),
        "expected dimmed child {child:?} below {parent:?}"
    );
}

#[when("the user selects X Axis in the filtered results")]
fn select_filtered_axis(world: &mut ApplicationWorld) {
    world.click_text("X Axis");
}

#[then("the X Axis row is highlighted")]
fn axis_row_selected(world: &mut ApplicationWorld) {
    assert!(world.row_highlighted("X Axis"), "expected X Axis selection");
}

#[then("its corresponding axis is highlighted in the viewport")]
fn axis_visible(world: &mut ApplicationWorld) {
    super::browser_feature::expect_outline(world, true);
}

#[when("the user clears the search using the search close control")]
#[when("the user activates the search close control again")]
fn use_search_close(world: &mut ApplicationWorld) {
    world.click_text("x");
}

#[then("X Axis remains selected and highlighted in the viewport")]
fn axis_selection_retained(world: &mut ApplicationWorld) {
    axis_row_selected(world);
    axis_visible(world);
}

#[then("the search field closes")]
fn search_field_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Type Keyword");
}

#[then("the axis highlight disappears from the viewport")]
fn axis_no_longer_visible(world: &mut ApplicationWorld) {
    super::browser_feature::expect_outline(world, false);
}
