// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Slot scenarios (slot-01..slot-08).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::{GuiHoleSeatKind, GuiHoleTermination, GuiSlotPreview};
use oblikovati_model::feature::{
    hole::{HoleFastenerFit, HoleKind, HoleOriginPlane, HoleTerminationSurface},
    slot::{SlotCenterMarks, SlotShape},
};
use setup::{
    SLOT_WINDOW, accept_slot_and_wait, click_slot_control, click_slot_ribbon,
    click_top_face_for_slot, create_baseline_slot1, reopen_slot1, replace_slot_field,
    select_slot_combo,
};

#[given("the rectangular solid has a visible planar end face")]
fn rectangular_solid_has_visible_face(world: &mut ApplicationWorld) {
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[when("the user starts Slot and sets its length to 4 mm and width to 2 mm")]
fn start_slot_4mm_by_2mm(world: &mut ApplicationWorld) {
    click_slot_ribbon(world);
    replace_slot_field(world, "Length", "4 mm", false);
    replace_slot_field(world, "Width", "2 mm", false);
}

#[when("the user clicks the face to place a position")]
fn click_face_to_place_slot(world: &mut ApplicationWorld) {
    click_top_face_for_slot(world);
}

#[then("the panel shows one position and a cut preview")]
fn slot_panel_shows_position_and_preview(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SLOT_WINDOW));
    assert!(
        text.contains("1Position"),
        "expected 1 Position pill in Slot panel, saw {text}"
    );
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.result, GuiSlotPreview::Ready);
    assert!(world.state().extrude_preview_active);
    assert!(world.run_is_enabled("OK"));
}

#[then("Slot1 appears in the browser and the cut is visible")]
fn slot1_appears_and_cut_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Slot1"),
        "expected Slot1 in browser features: {:?}",
        browser.features
    );
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[given("the user starts Slot and enters a 6 mm width")]
fn start_slot_and_enter_6mm_width(world: &mut ApplicationWorld) {
    click_slot_ribbon(world);
    replace_slot_field(world, "Length", "8 mm", false);
    replace_slot_field(world, "Width", "6 mm", false);
}

#[when("the user changes the length to 4 mm")]
fn change_slot_length_to_4mm(world: &mut ApplicationWorld) {
    replace_slot_field(world, "Length", "4 mm", false);
}

#[then("red validation indicators appear")]
fn red_validation_indicators_appear(world: &mut ApplicationWorld) {
    let text = world.window_text(SLOT_WINDOW);
    assert!(
        text.contains('!'),
        "expected red validation indicator in Slot window, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"));
}

#[when("the user changes the width to 2 mm")]
fn change_slot_width_to_2mm(world: &mut ApplicationWorld) {
    replace_slot_field(world, "Width", "2 mm", false);
}

#[then("those indicators disappear")]
fn validation_indicators_disappear(world: &mut ApplicationWorld) {
    let text = world.window_text(SLOT_WINDOW);
    assert!(
        !text.contains('!'),
        "expected validation indicators to clear in Slot window, saw {text}"
    );
}

#[given(
    "the user creates Slot1 on the top face near its center with rounded ends, 4 mm length, 2 mm width and 8 mm distance depth"
)]
fn create_slot1_rounded_4x2x8(world: &mut ApplicationWorld) {
    create_baseline_slot1(world);
}

#[given("the user opens Slot1 for editing")]
fn open_slot1_for_editing(world: &mut ApplicationWorld) {
    reopen_slot1(world);
}

#[given("rounded ends are selected")]
fn verify_rounded_ends_selected(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.shape, SlotShape::Rounded);
}

#[when("the user selects rectangular ends and enters a 45 degree rotation")]
fn select_rect_and_45_deg(world: &mut ApplicationWorld) {
    click_slot_control(world, "Rect");
    replace_slot_field(world, "Rotation", "45 deg", false);
}

#[when("the user selects distance termination")]
fn select_distance_termination(world: &mut ApplicationWorld) {
    select_slot_combo(world, "Termination", "Distance");
}

#[then("the preview is rectangular and rotated and the Depth field shows 8 mm")]
fn preview_rect_rotated_8mm(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.result, GuiSlotPreview::Ready);
    assert_eq!(preview.request.shape, SlotShape::Rectangular);
    assert_eq!(preview.request.rotation, "45 deg");
    assert_eq!(preview.request.termination, GuiHoleTermination::Distance);
    assert_eq!(preview.request.depth, "8 mm");
    assert!(world.state().extrude_preview_active);
}

#[when("the user accepts and reopens Slot1")]
fn accept_and_reopen_slot1(world: &mut ApplicationWorld) {
    accept_slot_and_wait(world);
    reopen_slot1(world);
}

#[then("those settings remain selected")]
fn rect_rotation_depth_retained(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.shape, SlotShape::Rectangular);
    assert_eq!(preview.request.rotation, "45 deg");
    assert_eq!(preview.request.termination, GuiHoleTermination::Distance);
    assert_eq!(preview.request.depth, "8 mm");
}

#[when("the user selects To Surface")]
fn select_slot_to_surface(world: &mut ApplicationWorld) {
    select_slot_combo(world, "Termination", "To Surface");
}

#[then("OK is disabled until a surface is selected")]
fn ok_disabled_until_surface_selected(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.termination, GuiHoleTermination::ToSurface);
    assert!(preview.request.termination_surface.is_none());
    assert!(!world.run_is_enabled("OK"));
}

#[when("the user selects the XY origin plane")]
fn select_xy_origin_plane_for_slot(world: &mut ApplicationWorld) {
    setup::pick_xy_origin_plane(world);
}

#[then("the selected surface remains shown")]
fn slot_selected_surface_retained(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.termination, GuiHoleTermination::ToSurface);
    assert_eq!(
        preview.request.termination_surface,
        Some(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy))
    );
}

#[when("the user selects the second Seat icon")]
fn select_slot_counterbore_seat(world: &mut ApplicationWorld) {
    click_slot_control(world, "C'bore");
}

#[then("the diagram and Counterbore field show the seat dimensions")]
fn diagram_and_counterbore_field_show_seat(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.seat, GuiHoleSeatKind::Counterbore);
    let text = super::super::world::squeezed(&world.window_text(SLOT_WINDOW));
    assert!(
        text.contains("Counterbore") && text.contains("4mm"),
        "expected Counterbore field and seat width in Slot window, saw {text}"
    );
}

#[then("the counterbore seat remains selected")]
fn counterbore_seat_retained(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.seat, GuiHoleSeatKind::Counterbore);
}

#[when("the user selects the fourth Seat icon")]
fn select_slot_countersink_seat(world: &mut ApplicationWorld) {
    click_slot_control(world, "C'sink");
}

#[then("the Countersink field shows 90 degrees")]
fn countersink_field_shows_90_deg(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.seat, GuiHoleSeatKind::Countersink);
    assert_eq!(preview.request.seat_angle, "90 deg");
    let text = super::super::world::squeezed(&world.window_text(SLOT_WINDOW));
    assert!(
        text.contains("Countersink") && text.contains("90\u{00b0}"),
        "expected Countersink field and 90° callout in Slot window, saw {text}"
    );
}

#[then("the fourth Seat icon and angle remain selected")]
fn countersink_and_angle_retained(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.seat, GuiHoleSeatKind::Countersink);
    assert_eq!(preview.request.seat_angle, "90 deg");
}

#[when("the user chooses the second Hole type and Loose fit")]
fn choose_clearance_and_loose_fit(world: &mut ApplicationWorld) {
    click_slot_control(world, "Clearance");
    select_slot_combo(world, "Fit", "Loose");
}

#[then("Fastener fields appear and the displayed width changes")]
fn fastener_fields_appear_and_width_changes(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SLOT_WINDOW));
    assert!(
        text.contains("FastenerType") && text.contains("Fit"),
        "expected Fastener fields in Slot window, saw {text}"
    );
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert_eq!(preview.request.kind, HoleKind::Clearance);
    assert_ne!(preview.request.width, "2 mm");
    assert_eq!(preview.request.width, "3.45 mm");
}

#[when("the user selects the fourth Center Marks icon and accepts the edit")]
fn select_all_center_marks_and_accept(world: &mut ApplicationWorld) {
    click_slot_control(world, "All");
    accept_slot_and_wait(world);
}

#[when("the user reopens Slot1")]
fn reopen_slot1_when(world: &mut ApplicationWorld) {
    reopen_slot1(world);
}

#[then("Loose fit and the fourth Center Marks icon remain selected")]
fn loose_fit_and_center_marks_retained(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    let fastener = preview.request.fastener.expect("expected saved fastener");
    assert_eq!(fastener.fit, HoleFastenerFit::Loose);
    assert_eq!(preview.request.center_marks, SlotCenterMarks::All);
}

#[given("the user starts a new Slot with no position or target surface selected")]
fn start_new_slot_empty_part(world: &mut ApplicationWorld) {
    click_slot_ribbon(world);
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert!(preview.request.face.is_none());
}

#[when("the second End Mill icon is selected")]
fn select_point_end_mill(world: &mut ApplicationWorld) {
    click_slot_control(world, "Point");
}

#[then("a Point field appears showing 90 degrees")]
fn point_field_appears_90_deg(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert!(preview.request.end_mill_pointed);
    assert_eq!(preview.request.end_mill_angle, "90 deg");
    let text = super::super::world::squeezed(&world.window_text(SLOT_WINDOW));
    assert!(
        text.contains("Point") && text.contains("90deg"),
        "expected Point field with 90 deg in Slot window, saw {text}"
    );
}

#[when("the first End Mill icon is selected")]
fn select_flat_end_mill(world: &mut ApplicationWorld) {
    click_slot_control(world, "Flat");
}

#[then("the Point field is hidden")]
fn point_field_hidden(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert!(!preview.request.end_mill_pointed);
}

#[when("iMate and Extend Start are checked")]
fn check_imate_and_extend_start(world: &mut ApplicationWorld) {
    click_slot_control(world, "iMate");
    click_slot_control(world, "ExtendStart");
}

#[then("both checkboxes are selected")]
fn both_checkboxes_selected(world: &mut ApplicationWorld) {
    let preview = world.state().slot_preview.expect("expected slot preview");
    assert!(preview.request.imate);
    assert!(preview.request.extend_start);
}

#[then("OK remains disabled")]
fn ok_remains_disabled(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("OK"));
}

#[then("the Slot panel closes")]
fn slot_panel_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(SLOT_WINDOW).is_none(),
        "expected Slot panel to close"
    );
}
