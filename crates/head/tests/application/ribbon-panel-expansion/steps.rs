// SPDX-License-Identifier: GPL-2.0-only
//! Native interaction with the Modify panel's additional commands.
use super::workspace_layout::browser_feature::{expect_solid_visible, open_visible_extruded_part};
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
const WINDOW: &str = "##ribbon-additional-PartModify";
const SIMPLIFICATION: &str = "##ribbon-additional-PartSimplification";

#[given("a part containing visible solid bodies is active in the MCAD Software")]
#[given("a part with a visible solid is active in the MCAD Software")]
fn solid_part(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    expect_solid_visible(world);
}
#[given("the 3D Model ribbon is expanded and the Modify panel is visible")]
fn full_modify(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    world.expect_text("Modify");
}
#[given("the Modify panel's additional command area is closed")]
fn additional_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[when("I select the expansion arrow on the Modify panel title")]
#[when("the user opens the Modify additional command area")]
fn open_modify(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Modify");
}
fn click_expansion_arrow(world: &mut ApplicationWorld, label: &str) {
    let title = world.run(label);
    if title.text.ends_with('v') {
        let at = [title.maximum[0] - 2., title.center()[1]];
        world.click(diagnostics::pointer_pixels(at));
        return;
    }
    let arrow = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == "v"
                && run.center()[0] > title.maximum[0]
                && (run.center()[1] - title.center()[1]).abs() < 6.
        })
        .min_by(|a, b| a.center()[0].total_cmp(&b.center()[0]))
        .unwrap_or_else(|| {
            panic!(
                "expected the {label} expansion arrow in {:?}",
                world.frame.runs
            )
        })
        .center();
    world.click(diagnostics::pointer_pixels(arrow));
}
#[then("an additional command area opens below the panel over the viewport")]
fn additional_below(_: &mut ApplicationWorld) {
    let (position, _) =
        diagnostics::window_geometry(WINDOW).expect("expected the Modify additional area");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(position[1] >= ribbon[1] + size[1]);
}
#[then("Move Bodies, Bend Part and Copy Object are visible")]
fn extra_commands(world: &mut ApplicationWorld) {
    for label in ["Move Bodies", "Bend Part", "Copy Object"] {
        world.expect_text(label);
    }
}
#[then("the solid bodies and model browser remain visible")]
fn solid_and_browser(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
    world.expect_text("Extrude1");
}
#[given("the 3D Model Modify panel is expanded with its pin control unselected")]
#[given("the unpinned Modify additional command area is open")]
#[given("the 3D Model Modify panel is expanded and unpinned")]
#[given("the 3D Model Modify additional command area is open")]
fn opened_unpinned(world: &mut ApplicationWorld) {
    open_modify(world);
    world.expect_text("Pin");
}
#[given("the Modify additional command area is open and pinned")]
#[given("the Modify panel is visible with its additional commands expanded and pinned")]
fn opened_pinned(world: &mut ApplicationWorld) {
    opened_unpinned(world);
    toggle_pin(world);
}
#[when("I select the pin control")]
#[when("I select the pin control again")]
#[when("the user unpins the Modify additional command area")]
#[when("the user unpins the additional area")]
#[when("the user clicks its pin control")]
fn toggle_pin(world: &mut ApplicationWorld) {
    let label = world.run("Pin");
    let position = [label.minimum[0] + 6., label.center()[1]];
    world.click(diagnostics::pointer_pixels(position));
}
#[then("the expanded Modify command area remains visible")]
#[then("the Modify additional command area remains visible")]
#[then("the additional command area remains open")]
fn pinned_visible(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    world.expect_text("Move Bodies");
}
#[then("the pin control is highlighted")]
fn pin_highlight(world: &mut ApplicationWorld) {
    let selected = world.pixel_left_of("Pin");
    toggle_pin(world);
    world.hover([1000, 700]);
    let unselected = world.pixel_left_of("Pin");
    assert_ne!(selected, unselected);
    toggle_pin(world);
}
#[then("the additional command area closes")]
#[then("the Modify additional command area is no longer visible")]
#[then("the Modify additional command area is not automatically displayed")]
fn area_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[when("the user clicks the Modify title at the bottom of the expanded area")]
#[when("the user clicks the Modify title at the bottom of the additional area")]
#[when("the user clicks its Modify title again")]
fn click_lower_title(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(WINDOW).unwrap();
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text.contains("Modif")
                && run.center()[0] >= position[0]
                && run.center()[0] <= position[0] + size[0]
                && run.center()[1] >= position[1]
                && run.center()[1] <= position[1] + size[1]
        })
        .expect("expected the additional area's Modify title");
    world.click(diagnostics::pointer_pixels([
        row.maximum[0] - 4.,
        row.center()[1],
    ]));
}
#[then("the 3D Model ribbon and part remain visible")]
fn ribbon_and_part(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    assert!(world.state().document.is_some());
}
#[when("the user selects the Tools ribbon tab")]
fn select_tools(world: &mut ApplicationWorld) {
    world.click_text("Tools");
}
#[when("the user selects the 3D Model ribbon tab")]
fn select_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}
#[given(
    "the Modify additional command area is pinned and remains open after a click in empty viewport space"
)]
fn pinned_before_home(world: &mut ApplicationWorld) {
    world.previous_document = world.state().document;
    opened_pinned(world);
    world.click([1000, 700]);
    pinned_visible(world);
}
#[when("the user selects the Home document tab")]
fn select_home_document(world: &mut ApplicationWorld) {
    world.click(diagnostics::pointer_pixels(
        world.expect_tab("Home").center(),
    ));
}
#[then("Home is displayed and the Modify additional command area is no longer visible")]
fn home_without_expansion(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##home-workspace").is_some());
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[when("the user returns to the part document tab")]
fn return_to_part(world: &mut ApplicationWorld) {
    world.click_tab(world.previous_document.unwrap());
}
#[then("the 3D Model ribbon is visible and the Modify additional command area is closed")]
fn model_without_expansion(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[then("the Modify additional command area closes")]
fn modify_closes(world: &mut ApplicationWorld) {
    area_closed(world);
}

#[given("the 3D Model Modify panel's unpinned additional command area is open")]
fn modify_unpinned(world: &mut ApplicationWorld) {
    opened_unpinned(world);
}
#[when("the user clicks the Simplification panel expansion arrow")]
#[when("the user opens the Simplification additional command area")]
fn open_simplification(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Simplification");
}
#[then("the Simplification additional command area opens")]
fn simplification_opens(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(SIMPLIFICATION).is_some());
}
#[then("it displays \"Remove Details\", \"Fill Voids\", and \"Define Envelopes\"")]
fn simplification_commands(world: &mut ApplicationWorld) {
    for label in ["Remove Details", "Fill Voids", "Define Envelopes"] {
        world.expect_text(label);
    }
}
#[then("the Simplification additional command area closes")]
fn simplification_closes(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(SIMPLIFICATION).is_none());
}
#[then("the part and 3D Model ribbon remain visible")]
fn part_and_model(world: &mut ApplicationWorld) {
    ribbon_and_part(world);
}
#[then("both the Modify and Simplification additional command areas are visible")]
#[then("both additional command areas remain visible")]
fn both_visible(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    assert!(diagnostics::window_geometry(SIMPLIFICATION).is_some());
}
#[then("the pinned Modify additional command area remains visible")]
fn modify_still_visible(world: &mut ApplicationWorld) {
    pinned_visible(world);
}
fn toggle_pin_in(world: &mut ApplicationWorld, window: &str) {
    let (position, size) = diagnostics::window_geometry(window).unwrap();
    let pin = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let center = run.center();
            run.text.contains("Pin")
                && (0..2).all(|i| center[i] >= position[i] && center[i] <= position[i] + size[i])
        })
        .expect("expected a pin control inside the expansion");
    world.click(diagnostics::pointer_pixels([
        pin.minimum[0] + 6.,
        pin.center()[1],
    ]));
}
#[when("the user opens and pins the Simplification additional command area")]
fn pin_simplification(world: &mut ApplicationWorld) {
    open_simplification(world);
    toggle_pin_in(world, SIMPLIFICATION);
}
#[when("the user unpins Simplification and clicks empty viewport space")]
fn unpin_simplification(world: &mut ApplicationWorld) {
    toggle_pin_in(world, SIMPLIFICATION);
    world.click([1000, 700]);
}
#[when("the user unpins Modify and clicks empty viewport space")]
fn unpin_modify(world: &mut ApplicationWorld) {
    toggle_pin_in(world, WINDOW);
    world.click([1000, 700]);
}
