// SPDX-License-Identifier: GPL-2.0-only
//! Native interaction with the Modify panel's additional commands.
use super::workspace_layout::browser_feature::{
    expect_outline, expect_solid_visible, open_visible_extruded_part,
};
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
#[path = "bend_notice.rs"]
mod bend_notice;
#[path = "builtin_commands.rs"]
mod builtin_commands;
#[path = "command_context.rs"]
mod command_context;
#[path = "custom_commands.rs"]
mod custom_commands;
#[path = "face_selection.rs"]
mod face_selection;
#[path = "footer_comparison.rs"]
mod footer_comparison;
#[path = "grouped_plane.rs"]
mod grouped_plane;
#[cfg(target_os = "linux")]
#[path = "keyboard_face_retention.rs"]
mod keyboard_face_retention;
#[path = "keyboard_full.rs"]
mod keyboard_full;
#[cfg(target_os = "linux")]
#[path = "shell_native.rs"]
mod shell_native;
#[path = "shell_placement.rs"]
mod shell_placement;
#[path = "sketch_block_measure.rs"]
mod sketch_block_measure;
#[path = "sketch_block_placement.rs"]
mod sketch_block_placement;
#[path = "sketch_constrain.rs"]
mod sketch_constrain;
#[path = "sketch_constrain_menu.rs"]
mod sketch_constrain_menu;
#[path = "sketch_create_live.rs"]
mod sketch_create_live;
#[path = "sketch_layout.rs"]
mod sketch_layout;
#[path = "temporary_restore.rs"]
mod temporary_restore;
#[cfg(target_os = "linux")]
#[path = "temporary_tab_native.rs"]
mod temporary_tab_native;
#[path = "title_appearance.rs"]
mod title_appearance;
#[path = "title_customization.rs"]
mod title_customization;
const WINDOW: &str = "##ribbon-additional-PartModify";
const SIMPLIFICATION: &str = "##ribbon-additional-PartSimplification";

#[given("a part with a solid body is active in the MCAD Software")]
#[given("a part containing a solid is active in the MCAD Software")]
#[given("a part containing visible solid bodies is active in the MCAD Software")]
#[given("a part with a visible solid is active in the MCAD Software")]
fn solid_part(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    expect_solid_visible(world);
}
#[given("the full 3D Model ribbon is visible with the Modify panel")]
#[given("the full 3D Model ribbon is displayed")]
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
#[when("the user opens the Modify panel's additional commands")]
#[when("the user opens the Modify additional command area")]
#[when("the user opens the Modify additional command area again")]
#[when("the user opens the Modify additional-command area")]
fn open_modify(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Modify");
}
pub(in super::super) fn click_expansion_arrow(world: &mut ApplicationWorld, label: &str) {
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
#[then("Move Bodies, Bend Part and Copy Object are visible in the additional area")]
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
#[given("the Modify additional command area is open and unpinned")]
#[given("the 3D Model Modify panel is expanded and unpinned")]
#[given("the 3D Model Modify additional command area is open")]
#[given("the Modify additional command area is open")]
pub(super) fn opened_unpinned(world: &mut ApplicationWorld) {
    if diagnostics::window_geometry("##ribbon-column").is_some() {
        world.click_text("Modify");
        super::ribbon_panel_visibility::side_selector::open_side_additional(world);
        world.expect_text("Pin");
        return;
    }
    open_modify(world);
    world.expect_text("Pin");
}
#[given("the Modify additional command area is open and pinned")]
#[given("the Modify panel is visible with its additional commands expanded and pinned")]
fn opened_pinned(world: &mut ApplicationWorld) {
    opened_unpinned(world);
    toggle_pin(world);
}
#[when("the user selects the pin control")]
#[when("the user selects the pin control again")]
#[when("I select the pin control")]
#[when("I select the pin control again")]
#[when("the user unpins the Modify additional command area")]
#[when("the user unpins the additional area")]
#[when("the user unpins the additional command area")]
#[when("the user clicks its pin control")]
pub(super) fn toggle_pin(world: &mut ApplicationWorld) {
    let label = world.run("Pin");
    let position = [label.minimum[0] + 6., label.center()[1]];
    world.click(diagnostics::pointer_pixels(position));
}
#[then("the Modify additional commands remain visible")]
#[then("the expanded Modify command area remains visible")]
#[then("the Modify additional command area remains visible")]
#[then("the Modify additional-command area opens")]
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
#[then("the Modify additional command area is closed")]
#[then("the additional commands are closed")]
#[then("the additional command area closes")]
#[then("the additional area closes")]
#[then("the additional command area is no longer open")]
#[then("the Modify additional command area remains closed")]
#[then("the Modify additional command area is no longer visible")]
#[then("the Modify additional command area is not automatically displayed")]
fn area_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[given("the Modify additional command area is unpinned")]
fn modify_starts_unpinned(world: &mut ApplicationWorld) {
    open_modify(world);
    click_lower_title(world);
    area_closed(world);
}
#[when("the user clicks the Modify panel title")]
fn click_modify_title(world: &mut ApplicationWorld) {
    world.click_text("Modify");
}
#[then("the additional area remains visible with those commands")]
fn commands_remain_visible(world: &mut ApplicationWorld) {
    pinned_visible(world);
    extra_commands(world);
}
#[cfg(target_os = "linux")]
#[test]
fn x11_pointer_expansion_stays_open_after_escape() {
    let mut world = ApplicationWorld::new();
    solid_part(&mut world);
    let title = diagnostics::pointer_pixels(world.run("Modify").center());
    super::x11_input::click(title);
    world.pump(Vec::new());
    commands_remain_visible(&mut world);
    super::x11_input::press_escape();
    world.pump(Vec::new());
    commands_remain_visible(&mut world);
    let lower = world.run_in_window(WINDOW, "Modify");
    super::x11_input::click(diagnostics::pointer_pixels(lower.center()));
    world.pump(Vec::new());
    area_closed(&mut world);
    world.close();
}
#[when("the user clicks the Modify title at the bottom of the expanded area")]
#[when("the user clicks the Modify title at the bottom of the additional area")]
#[when("the user clicks its Modify title again")]
#[when("the user clicks the lower Modify title")]
fn click_lower_title(world: &mut ApplicationWorld) {
    let row = world.run_in_window(WINDOW, "Modify");
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
#[when("the user selects the Tools ribbon tab again")]
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
#[given("a part containing an extrusion is active in the MCAD Software")]
fn extrusion_part(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    expect_solid_visible(world);
}
#[given("the full 3D Model ribbon has the Modify additional-command area open and unpinned")]
fn browser_unpinned(world: &mut ApplicationWorld) {
    full_modify(world);
    opened_unpinned(world);
}
#[given("the full 3D Model ribbon has the Modify additional-command area open and pinned")]
fn browser_pinned(world: &mut ApplicationWorld) {
    full_modify(world);
    opened_pinned(world);
}
#[when("the user selects the extrusion in the model browser")]
fn select_browser_extrusion(world: &mut ApplicationWorld) {
    world.click_text("Extrude1");
}
#[then("the additional-command area closes")]
fn browser_expansion_closed(world: &mut ApplicationWorld) {
    area_closed(world);
}
#[then("the additional-command area remains open")]
fn browser_expansion_open(world: &mut ApplicationWorld) {
    pinned_visible(world);
}
#[then("the extrusion is selected in the browser")]
fn browser_extrusion_selected(world: &mut ApplicationWorld) {
    assert!(world.row_highlighted("Extrude1"));
}
#[then("its geometry is highlighted in the viewport")]
fn browser_geometry_highlighted(world: &mut ApplicationWorld) {
    expect_outline(world, true);
}
#[when("the user unpins the additional-command area and clicks empty viewport space")]
fn unpin_browser_expansion(world: &mut ApplicationWorld) {
    unpin_modify(world);
}
#[when("the user clicks the 3D Model tab")]
#[when("the user clicks the 3D Model tab again")]
#[when("the user clicks the selected 3D Model tab")]
fn click_model_tab(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}
#[when("the user clicks the selected 3D Model tab again")]
fn click_model_tab_again(world: &mut ApplicationWorld) {
    std::thread::sleep(std::time::Duration::from_millis(500));
    click_model_tab(world);
}
#[then("the additional area shows Move Bodies, Bend Part and Copy Object")]
#[then("the area shows Move Bodies, Bend Part, and Copy Object")]
fn temporary_commands(world: &mut ApplicationWorld) {
    extra_commands(world);
}
#[then("the additional area and the temporary command area close")]
#[then("both command areas close")]
fn temporary_and_additional_close(world: &mut ApplicationWorld) {
    area_closed(world);
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("the Modify additional-command area remains closed")]
fn temporary_expansion_closed(world: &mut ApplicationWorld) {
    area_closed(world);
}
#[then("the temporary command area remains visible")]
fn temporary_remains(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_some());
}
#[then("the Modify additional-command area remains visible")]
fn temporary_expansion_visible(world: &mut ApplicationWorld) {
    pinned_visible(world);
}
#[then("the temporary command area closes")]
fn temporary_area_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("the Modify additional-command area closes")]
fn temporary_modify_closes(world: &mut ApplicationWorld) {
    area_closed(world);
}
#[then("the temporary command area displays the Tools panels")]
fn temporary_tools(world: &mut ApplicationWorld) {
    temporary_remains(world);
    world.expect_text("Document Settings");
}
#[then("the temporary command area displays the 3D Model panels")]
fn temporary_model(world: &mut ApplicationWorld) {
    temporary_remains(world);
    world.expect_text("Start 2D Sketch");
}

#[when("the user opens the Modify additional command area and pins it")]
#[when("the user opens and pins the Modify additional command area")]
fn open_and_pin_modify(world: &mut ApplicationWorld) {
    open_modify(world);
    toggle_pin(world);
}

#[then("the ribbon context menu appears and the additional commands remain visible")]
fn title_menu_with_expansion(world: &mut ApplicationWorld) {
    world.expect_text("Show Panels");
    pinned_visible(world);
}

#[then("the context menu offers \"Customize User Commands\"")]
fn context_offers_customization(world: &mut ApplicationWorld) {
    world.expect_text("Customize User Commands");
}
#[then("the dialog closes and the full 3D Model ribbon remains visible")]
fn customization_closed_with_model_ribbon(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
    world.expect_text("Start 2D Sketch");
}

#[cfg(target_os = "linux")]
#[path = "sketch_inference_scope_native.rs"]
mod sketch_inference_scope_native;
