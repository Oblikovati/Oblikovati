// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[given(
    "a fresh copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open in Extrude with the top circular profile selected"
)]
#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open with its circular profile on the existing solid"
)]
fn open_top_circle(world: &mut ApplicationWorld) {
    published_circle_part(world);
    world.click_text("Sketch1");
    world.click_text("Extrude");
}

#[when("the user selects Through All and Cut in the direction into the base")]
fn select_through_all(world: &mut ApplicationWorld) {
    // A short numeric distance cannot reach the opposite face of this 8 mm base.
    super::super::replace_extrusion_distance(world, "1");
    click_panel(world, "Distance");
    world.click_text("Through All");
    world.pump(Vec::new());
    assert_through_all_request(world);
}

#[when("the first auxiliary extent icon beside Distance A is clicked")]
fn select_auxiliary_extent(world: &mut ApplicationWorld) {
    use super::super::super::ribbon_appearance::icon_quads::icon_quads_inside;
    let label = world.run_in_window("Extrude", "DistanceA");
    let ([left, _], [width, _]) = diagnostics::window_geometry("Extrude").unwrap();
    let y = label.center()[1];
    let icons = icon_quads_inside(([left + width - 90., y - 13.], [left + width, y + 13.]));
    let [edit, extent] = icons.as_slice() else {
        panic!("expected the expression menu and auxiliary extent icons, got {icons:?}");
    };
    assert!(edit.0[0] < extent.0[0]);
    let [minimum, maximum] = [extent.0, extent.1];
    let center = std::array::from_fn(|axis| (minimum[axis] + maximum[axis]) * 0.5);
    super::super::viewport_interaction::click_at(world, diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

#[then("direction shows Flipped and Boolean shows Cut")]
fn auxiliary_extent_sets_inward_cut(world: &mut ApplicationWorld) {
    assert_through_all_request(world);
    assert_top_opening(world, true);
    assert_opposite_opening(world);
    world.run_in_window("Extrude", "Cut");
}

#[then("the displayed distance is parenthesized")]
fn retained_distance_is_parenthesized(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "(10.00mm)");
}

#[then("the opening is retained")]
fn auxiliary_cut_is_saved(world: &mut ApplicationWorld) {
    accepted_through_cut(world);
}

#[then("the circular sketch and the uncut top face return")]
fn undo_auxiliary_cut(world: &mut ApplicationWorld) {
    assert_original_shape(world);
    assert_top_opening(world, false);
}

fn assert_through_all_request(world: &mut ApplicationWorld) {
    let draft = world
        .state()
        .extrude_draft_preview
        .expect("certified Through All");
    assert_eq!(
        draft.reference_extent,
        Some(oblikovati_gui::GuiExtrudeReferenceExtent::ThroughAll)
    );
    assert_eq!(
        draft.direction,
        oblikovati_gui::GuiExtrudeDirection::Negative
    );
    assert_eq!(
        draft.operation,
        Some(oblikovati_model::feature::sketch_prism::PrismBoolean::Cut)
    );
}

#[then("Join is unavailable with this extent")]
fn only_cut(world: &mut ApplicationWorld) {
    click_panel(world, "Cut");
    world.expect_text("Join");
    assert!(!world.run_is_enabled("Join"));
    world.key(ViewportPhysicalKey::Escape, false);
    world.pump(Vec::new());
}

#[then("the preview crosses the complete thickness")]
fn through_preview(world: &mut ApplicationWorld) {
    assert_visible_replacement(world);
    assert_top_opening(world, true);
    assert_opposite_opening(world);
}

#[then("the circular opening exits the opposite face and the part still has one solid body")]
fn accepted_through_cut(world: &mut ApplicationWorld) {
    cut_result(world);
    assert_opposite_opening(world);
}

fn assert_opposite_opening(world: &mut ApplicationWorld) {
    let triangles = &world.app().window.scene_triangles;
    let cap = triangles
        .iter()
        .filter(|triangle| triangle.positions.iter().all(|point| point[2].abs() < 1e-5));
    assert!(
        !cap.clone()
            .any(|triangle| contains_xy(triangle.positions, [2., 1.5]))
    );
    assert!(
        cap.clone()
            .any(|triangle| contains_xy(triangle.positions, [2.3, 1.5]))
    );
}
