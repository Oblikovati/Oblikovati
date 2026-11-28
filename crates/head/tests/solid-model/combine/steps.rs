// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Combine scenarios (combine-01..combine-08).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiCombinePreview;
use oblikovati_model::feature::combine::CombineOperation;
use setup::{
    COMBINE_WINDOW, EDIT_ERROR_WINDOW, accept_combine_and_wait, click_clear_on_row,
    click_combine_control, create_combine1_cut_two_tools, create_combine1_join_keep_tool,
    double_click_combine1, open_combine_fixture, open_combine_ribbon,
    select_block_and_both_cylinders, select_block_and_first_cylinder, set_keep_toolbodies,
};

#[given(expr = "the native fixture {string} is open")]
fn native_fixture_is_open(world: &mut ApplicationWorld, relative: String) {
    open_combine_fixture(world, &relative);
}

#[given("the block and cylindrical tool body are available for selection")]
fn block_and_cylinder_available(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        2,
        "expected 2 solid bodies in two-bodies.opd, saw {:?}",
        browser.bodies
    );
}

#[given("the user opens Combine")]
#[when("I open Combine")]
fn open_combine(world: &mut ApplicationWorld) {
    open_combine_ribbon(world);
}

#[then("Basebody and Toolbodies require selections and OK is unavailable")]
fn base_and_tool_require_selections(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert!(
        text.contains("Basebody")
            && text.contains("SelectSolid")
            && text.contains("Toolbodies")
            && text.contains("SelectSolids"),
        "expected Basebody/Toolbodies selection prompts in Combine panel, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("I select the block as base and the cylindrical body as tool")]
fn select_block_and_cylinder(world: &mut ApplicationWorld) {
    select_block_and_first_cylinder(world);
}

#[then("each selector shows 1 Solid and OK becomes available")]
fn each_selector_shows_one_solid_and_ok_available(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert_eq!(
        text.matches("1Solid").count(),
        2,
        "expected both selectors showing 1 Solid, saw {text}"
    );
    let preview = world.state().combine_preview.expect("expected preview");
    assert_eq!(preview.result, GuiCombinePreview::Ready);
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[given("Combine is open with the block as base and the cylindrical body as tool")]
fn combine_open_with_block_and_cylinder(world: &mut ApplicationWorld) {
    open_combine_ribbon(world);
    select_block_and_first_cylinder(world);
}

#[when("I select the first Boolean option and enable Keep Toolbodies")]
fn select_join_and_keep_toolbodies(world: &mut ApplicationWorld) {
    click_combine_control(world, "Join");
    set_keep_toolbodies(world, true);
}

#[when("I accept Combine")]
fn accept_combine(world: &mut ApplicationWorld) {
    accept_combine_and_wait(world);
}

#[then("Combine1 appears in the browser")]
fn combine1_appears_in_browser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Combine1" && feature.kind == "Combine"),
        "expected Combine1 in browser features: {:?}",
        browser.features
    );
}

#[then("the browser still shows two solid bodies")]
fn browser_still_shows_two_solid_bodies(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        2,
        "expected 2 solid bodies with Keep Toolbodies enabled, saw {:?}",
        browser.bodies
    );
}

#[given(
    "the user creates Combine1 using the first Boolean option, the block as base, the cylindrical body as tool and Keep Toolbodies enabled"
)]
fn create_combine1_first_option_keep_tools(world: &mut ApplicationWorld) {
    create_combine1_join_keep_tool(world);
}

#[when("I double-click Combine1 in the browser")]
fn double_click_combine1_step(world: &mut ApplicationWorld) {
    double_click_combine1(world);
}

#[then(
    "the Combine1 properties open with one base body, one tool body, and Keep Toolbodies checked"
)]
fn verify_combine1_reopened_properties(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview");
    assert!(preview.request.base_body.is_some());
    assert_eq!(preview.request.tool_bodies.len(), 1);
    assert!(preview.request.keep_toolbodies);
    assert_eq!(preview.request.operation, CombineOperation::Join);
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert_eq!(text.matches("1Solid").count(), 2, "{text}");
}

#[given("Combine1 is open for editing")]
fn combine1_open_for_editing(world: &mut ApplicationWorld) {
    double_click_combine1(world);
}

#[when("I select the second Boolean option and accept")]
fn select_cut_and_accept(world: &mut ApplicationWorld) {
    click_combine_control(world, "Cut");
    click_combine_control(world, "OK");
    world.pump(Vec::new());
}

#[then("the MCAD Software reports problems executing the Edit Combine Feature command")]
#[then("the MCAD Software reports a problem executing Edit Combine Feature")]
fn reports_edit_combine_feature_error(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(EDIT_ERROR_WINDOW).is_some(),
        "expected Edit Combine Feature modal, saw {}",
        world.text()
    );
    let modal = super::super::world::squeezed(&world.window_text(EDIT_ERROR_WINDOW));
    assert!(
        modal.contains("EditCombineFeature"),
        "expected Edit Combine Feature error text, saw {modal}"
    );
}

#[when("I cancel the error")]
fn cancel_combine_error_modal(world: &mut ApplicationWorld) {
    let center = world.run_in_window(EDIT_ERROR_WINDOW, "Cancel").center();
    world.click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the model and Combine1 return to view")]
#[then("the previous Combine1 result returns to view")]
fn previous_combine1_returns_to_view(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(EDIT_ERROR_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(COMBINE_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Combine1"),
        "expected Combine1 preserved in browser: {:?}",
        browser.features
    );
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[given(
    "the two cylindrical tool profiles have 5 mm radii and XY centers at (10, 15) mm and (30, 15) mm"
)]
fn verify_two_cylindrical_tool_profiles(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        3,
        "expected 3 solid bodies in three-bodies.opd, saw {:?}",
        browser.bodies
    );
}

#[when("I choose the block as the base and both cylinders as tools")]
fn choose_block_and_both_cylinders(world: &mut ApplicationWorld) {
    select_block_and_both_cylinders(world);
}

#[then("Combine shows one base solid and two tool solids")]
fn combine_shows_one_base_and_two_tools(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview");
    assert!(preview.request.base_body.is_some());
    assert_eq!(preview.request.tool_bodies.len(), 2);
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert!(
        text.contains("1Solid") && text.contains("2Solids"),
        "expected 1 Solid and 2 Solids in Combine panel, saw {text}"
    );
}

#[then("OK is available")]
fn ok_is_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("OK"), "expected OK available");
}

#[given("Combine is open with the block as base and both cylinders as tools")]
fn combine_open_with_block_and_both_cylinders(world: &mut ApplicationWorld) {
    open_combine_ribbon(world);
    select_block_and_both_cylinders(world);
}

#[given("Keep Toolbodies is unchecked")]
fn keep_toolbodies_unchecked(world: &mut ApplicationWorld) {
    set_keep_toolbodies(world, false);
}

#[when("I select the second Boolean option and click OK")]
fn select_cut_and_click_ok(world: &mut ApplicationWorld) {
    click_combine_control(world, "Cut");
    accept_combine_and_wait(world);
}

#[then("Combine1 appears and the Solid Bodies folder shows one solid")]
fn combine1_appears_and_one_solid_remains(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Combine1"),
        "expected Combine1 in browser: {:?}",
        browser.features
    );
    assert_eq!(
        browser.bodies.len(),
        1,
        "expected 1 solid body after Cut without Keep Toolbodies, saw {:?}",
        browser.bodies
    );
}

#[when("I reopen Combine1")]
fn reopen_combine1_step(world: &mut ApplicationWorld) {
    double_click_combine1(world);
}

#[then("the dialog shows one base, two tools, the second option, and Keep Toolbodies unchecked")]
fn verify_two_tool_cut_reopened(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview");
    assert!(preview.request.base_body.is_some());
    assert_eq!(preview.request.tool_bodies.len(), 2);
    assert_eq!(preview.request.operation, CombineOperation::Cut);
    assert!(!preview.request.keep_toolbodies);
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert!(
        text.contains("1Solid") && text.contains("2Solids"),
        "{text}"
    );
}

#[given(
    "the user creates Combine1 using the second Boolean option with the block as base, both cylinders as tools, and Keep Toolbodies unchecked"
)]
fn create_combine1_cut_two_tools_step(world: &mut ApplicationWorld) {
    create_combine1_cut_two_tools(world);
}

#[when("I select the third Boolean option and click OK")]
fn select_intersect_and_click_ok(world: &mut ApplicationWorld) {
    click_combine_control(world, "Intersect");
    click_combine_control(world, "OK");
    world.pump(Vec::new());
}

#[when("I clear the base selection")]
fn clear_base_selection(world: &mut ApplicationWorld) {
    click_clear_on_row(world, "Basebody");
}

#[then("Basebody requests a solid and OK is disabled")]
fn basebody_requests_solid_ok_disabled(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview");
    assert!(preview.request.base_body.is_none());
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert!(
        text.contains("SelectSolid"),
        "expected Basebody to request a solid, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("I clear the tool selection")]
fn clear_tool_selection(world: &mut ApplicationWorld) {
    click_clear_on_row(world, "Toolbodies");
}

#[then("Toolbodies requests solids")]
fn toolbodies_requests_solids(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview");
    assert!(preview.request.tool_bodies.is_empty());
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert!(
        text.contains("SelectSolids"),
        "expected Toolbodies to request solids, saw {text}"
    );
}

#[when("I cancel and reopen Combine1")]
fn cancel_and_reopen_combine1(world: &mut ApplicationWorld) {
    click_combine_control(world, "Cancel");
    assert!(super::super::diagnostics::window_geometry(COMBINE_WINDOW).is_none());
    double_click_combine1(world);
}

#[then("the base and tool each show one solid")]
fn base_and_tool_each_show_one_solid(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview");
    assert!(preview.request.base_body.is_some());
    assert_eq!(preview.request.tool_bodies.len(), 1);
    let text = super::super::world::squeezed(&world.window_text(COMBINE_WINDOW));
    assert_eq!(text.matches("1Solid").count(), 2, "{text}");
}

#[then("the Combine panel closes")]
fn combine_panel_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(COMBINE_WINDOW).is_none(),
        "expected Combine panel to close"
    );
}
