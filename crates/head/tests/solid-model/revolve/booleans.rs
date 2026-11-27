// SPDX-License-Identifier: GPL-2.0-only
//! Revolve Boolean BDD steps on `revolve-boolean-base.opd` (revolve-21..24).
use super::{edit_and_surface::reopen_feature, *};

fn pick_boolean_base_axis(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    world.pump(Vec::new());
    // Long edge of Sketch1 in XZ plane along Z=1.0 cm, x in [1.0, 2.0].
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, [1.5, 0., 1.0]).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
}

#[given(
    "a fresh copy of `fixtures/cad/revolve/revolve-boolean-base.opd` is open with its block and XZ sketch visible"
)]
#[given(
    "a writable copy of `fixtures/cad/revolve/revolve-boolean-base.opd` is open with the block and XZ sketch visible"
)]
fn open_revolve_boolean_base(world: &mut ApplicationWorld) {
    open_fixture_copy(world, "fixtures/cad/revolve/revolve-boolean-base.opd");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
}

#[given(
    "the rectangle in the visible XZ sketch is selected with its long edge at Z=10 mm as the axis for a full turn"
)]
fn select_boolean_base_profile_and_axis(world: &mut ApplicationWorld) {
    invoke_revolve(world);
    pick_boolean_base_axis(world);
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    assert_eq!(preview.request.extent, GuiRevolveExtent::Full);
    assert_eq!(preview.request.output, FeatureOutputMode::Join);
}

#[when("OK is clicked with the first Boolean icon selected")]
fn accept_default_join_boolean(world: &mut ApplicationWorld) {
    click_revolve_button(world, "OK");
}

#[then("Revolution1 appears with a protrusion on the existing solid")]
#[then("Revolution1 and its protrusion return")]
fn revolution1_protrusion_appears(world: &mut ApplicationWorld) {
    world.expect_text("Revolution1");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert!(browser.bodies[0].faces > 6, "expected protrusion faces > 6");
    let (y_min, _) = preview_y_bounds(world);
    assert!(
        y_min < -0.35,
        "expected protrusion into -Y < -0.35 cm, got {y_min}"
    );
}

#[then("the browser shows Solid Bodies(1)")]
#[then("Solid Bodies(1) remains visible")]
fn browser_shows_one_solid(world: &mut ApplicationWorld) {
    world.expect_text("Solid Bodies");
    assert_eq!(world.state().browser.unwrap().bodies.len(), 1);
}

#[then("the original block and the XZ sketch return")]
fn original_block_and_xz_sketch_return(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1" && sketch.visible)
    );
}

#[given(
    "I create and accept a full-turn Revolution1 using the XZ rectangle, its long edge at Z=10 mm as axis, and Join"
)]
fn create_full_turn_join_revolve(world: &mut ApplicationWorld) {
    select_boolean_base_profile_and_axis(world);
    accept_default_join_boolean(world);
    revolution1_protrusion_appears(world);
}

#[when("Boolean selector style is changed to Dropdown")]
fn switch_boolean_style_to_dropdown(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    click_revolve_button(world, "Boolean");
    world.click_text("Dropdown");
    world.pump(Vec::new());
}

#[then("the current selection is named Join")]
fn boolean_selection_is_join(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("BooleanJoin"), "{panel}");
}

#[then("Join Cut Intersect and New Solid are listed")]
fn all_four_booleans_listed(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Join");
    for label in ["Join", "Cut", "Intersect", "New Solid"] {
        world.expect_text(label);
    }
}

#[when("Cut is selected and OK is clicked")]
fn select_cut_and_accept(world: &mut ApplicationWorld) {
    world.click_text("Cut");
    world.pump(Vec::new());
    assert_eq!(
        world.state().revolve_preview.unwrap().request.output,
        FeatureOutputMode::Cut
    );
    click_revolve_button(world, "OK");
}

#[then("the protrusion is replaced by a recess")]
fn protrusion_replaced_by_recess(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert!(browser.bodies[0].faces > 6);
    let (y_min, _) = preview_y_bounds(world);
    assert!(
        y_min > -0.05,
        "expected cut recess to remove -Y protrusion (>= 0 cm), got {y_min}"
    );
}

#[given("I edit Revolution1, select Cut, accept it and reopen the feature for editing")]
fn edit_revolve_to_cut_and_reopen(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    click_revolve_button(world, "Cut");
    click_revolve_button(world, "OK");
    protrusion_replaced_by_recess(world);
    reopen_feature(world, "Revolution1");
}

#[when("Intersect is selected and OK is clicked")]
fn select_intersect_and_accept(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Intersect");
    assert_eq!(
        world.state().revolve_preview.unwrap().request.output,
        FeatureOutputMode::Intersect
    );
    click_revolve_button(world, "OK");
}

#[then("the large block is replaced by a small curved solid")]
fn large_block_replaced_by_small_solid(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    let (y_min, y_max) = preview_y_bounds(world);
    let (z_min, z_max) = preview_z_bounds(world);
    assert!(
        y_min > -0.05 && y_max < 0.45 && z_min > 0.55 && z_max < 1.45,
        "expected small half-cylinder Y in [0, 0.4], Z in [0.6, 1.4], got Y=[{y_min}, {y_max}] Z=[{z_min}, {z_max}]"
    );
}

#[then("Revolution1 and Solid Bodies(1) remain in the browser")]
fn revolution1_and_one_solid_remain(world: &mut ApplicationWorld) {
    world.expect_text("Revolution1");
    browser_shows_one_solid(world);
}

#[given("I edit Revolution1 to Cut and accept it")]
fn edit_revolve_to_cut_and_accept(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    click_revolve_button(world, "Cut");
    click_revolve_button(world, "OK");
}

#[given("I reopen Revolution1, change Cut to Intersect, accept it and reopen it for editing")]
fn edit_cut_to_intersect_and_reopen(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    select_intersect_and_accept(world);
    large_block_replaced_by_small_solid(world);
    reopen_feature(world, "Revolution1");
}

#[when("New Solid is selected")]
fn select_new_solid(world: &mut ApplicationWorld) {
    click_revolve_button(world, "NewSolid");
}

#[then("Body Name displays Solid2 and a preview appears")]
fn body_name_displays_solid2(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("BodyNameSolid2"), "{panel}");
    assert_eq!(
        world.state().revolve_preview.unwrap().result,
        GuiRevolvePreview::Ready
    );
}

#[then("the block and revolved shape are visible")]
fn block_and_revolved_shape_visible(world: &mut ApplicationWorld) {
    let (y_min, y_max) = preview_y_bounds(world);
    assert!(
        y_min < -0.35 && y_max > 2.5,
        "expected block and new revolved solid, got Y=[{y_min}, {y_max}]"
    );
}
