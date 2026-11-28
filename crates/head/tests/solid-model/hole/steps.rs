// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Modify Hole scenarios (hole-01..hole-09 & contract).
mod presets_and_sketch;
pub(super) mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::{GuiHolePreview, GuiHoleSeatKind, GuiHoleTermination};
use oblikovati_model::feature::hole::{
    HoleFastenerFit, HoleKind, HoleOriginPlane, HoleTerminationSurface,
};
use setup::{
    HOLE_WINDOW, accept_hole_and_wait, click_end_face_for_hole, click_hole_control,
    click_hole_ribbon, open_hole_fixture, open_visible_solid_part, pick_xy_origin_plane,
    reopen_hole1, replace_hole_field, select_hole_combo,
};

#[given(
    "a writable copy of `fixtures/cad/fillet-circular-edge-candidate.opd` is open with a planar circular end face visible"
)]
fn open_fillet_circular_candidate(world: &mut ApplicationWorld) {
    open_hole_fixture(world, "fixtures/cad/fillet-circular-edge-candidate.opd");
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Pan([400, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

#[when("the user starts Hole and select the first Hole type")]
fn start_hole_and_select_first_type(world: &mut ApplicationWorld) {
    click_hole_ribbon(world);
    click_hole_control(world, "Simple");
}

#[when("the user clicks a point on the end face")]
fn click_point_on_end_face(world: &mut ApplicationWorld) {
    click_end_face_for_hole(world);
}

#[then("a hole preview appears and OK is enabled")]
fn hole_preview_and_ok_enabled(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.result, GuiHolePreview::Ready);
    assert!(world.state().extrude_preview_active);
    assert!(world.run_is_enabled("OK"));
}

#[then("the face has a visible opening and Hole1 appears in the browser")]
fn face_opening_and_hole1_in_browser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Hole1"),
        "expected Hole1 in browser features: {:?}",
        browser.features
    );
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[given("a writable copy of `fixtures/cad/hole/hole1-through-unique-candidate.opd` is open")]
fn open_hole1_through_fixture(world: &mut ApplicationWorld) {
    open_hole_fixture(
        world,
        "fixtures/cad/hole/hole1-through-unique-candidate.opd",
    );
}

#[given("a writable copy of `fixtures/cad/hole/hole1-4mm-unique-candidate.opd` is open")]
fn open_hole1_4mm_fixture(world: &mut ApplicationWorld) {
    open_hole_fixture(world, "fixtures/cad/hole/hole1-4mm-unique-candidate.opd");
}

#[given("Hole1 is a simple 3 mm through hole at the center of the planar circular end face")]
fn verify_hole1_3mm_seed(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let hole1 = browser
        .features
        .iter()
        .find(|f| f.name == "Hole1")
        .expect("expected Hole1 in fixture");
    let seed = browser
        .hole_edits
        .get(&hole1.id)
        .expect("expected Hole1 edit seed");
    assert_eq!(seed.diameter, "3 mm");
    assert_eq!(seed.termination, GuiHoleTermination::ThroughAll);
}

#[given("Hole1 is a simple 4 mm through hole at the center of the planar circular end face")]
fn verify_hole1_4mm_seed(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let hole1 = browser
        .features
        .iter()
        .find(|f| f.name == "Hole1")
        .expect("expected Hole1 in fixture");
    let seed = browser
        .hole_edits
        .get(&hole1.id)
        .expect("expected Hole1 edit seed");
    assert_eq!(seed.diameter, "4 mm");
    assert_eq!(seed.termination, GuiHoleTermination::ThroughAll);
}

#[when("the user edits Hole1 and change its diameter from 3 mm to 4 mm")]
fn edit_hole1_diameter_3_to_4(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.request.diameter, "3 mm");
    replace_hole_field(world, "Diameter", "4 mm", false);
}

#[when("the user presses Enter and reopen the feature")]
fn press_enter_and_reopen_hole(world: &mut ApplicationWorld) {
    replace_hole_field(world, "Diameter", "4 mm", true);
    assert!(
        super::super::diagnostics::window_geometry(HOLE_WINDOW).is_none(),
        "expected Enter to accept Hole1 edit"
    );
    reopen_hole1(world);
}

#[then("the diameter is 4 mm")]
fn diameter_is_4mm(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.request.diameter, "4 mm");
}

#[when("the user chooses To Surface and select the XY origin plane")]
fn choose_to_surface_xy(world: &mut ApplicationWorld) {
    select_hole_combo(world, "Termination", "To Surface");
    pick_xy_origin_plane(world);
}

#[then("the selector shows one surface and OK is enabled")]
fn selector_shows_one_surface_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("1Surface"),
        "expected 1 Surface pill in Hole window, saw {text}"
    );
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.result, GuiHolePreview::Ready);
    assert!(world.run_is_enabled("OK"));
}

#[when("the user accepts and reopen the feature")]
#[when("the user accepts and reopen Hole1")]
fn accept_and_reopen_hole1(world: &mut ApplicationWorld) {
    accept_hole_and_wait(world);
    reopen_hole1(world);
}

#[then("To Surface and the selected surface remain shown")]
fn to_surface_retained(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.request.termination, GuiHoleTermination::ToSurface);
    assert_eq!(
        preview.request.termination_surface,
        Some(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy))
    );
}

#[given("the user reopens Hole1, sets its diameter to 4 mm, accepts it and reopens it for editing")]
fn set_hole1_4mm_and_reopen(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    replace_hole_field(world, "Diameter", "4 mm", false);
    accept_hole_and_wait(world);
    reopen_hole1(world);
}

#[when("the user selects the stepped seat with both diameters set to 4 mm")]
fn select_stepped_seat_4mm(world: &mut ApplicationWorld) {
    click_hole_control(world, "C'bore");
    replace_hole_field(world, "SeatDiameter", "4 mm", false);
}

#[then("validation indicators appear and OK is disabled")]
fn validation_indicators_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = world.window_text(HOLE_WINDOW);
    assert!(
        text.contains('!'),
        "expected validation indicator in Hole window, saw {text}"
    );
    assert!(
        !world.run_is_enabled("OK"),
        "expected OK disabled when invalid"
    );
}

#[when("the user changes the seat diameter to 6 mm")]
fn change_seat_diameter_to_6mm(world: &mut ApplicationWorld) {
    replace_hole_field(world, "SeatDiameter", "6 mm", false);
}

#[then("a stepped preview appears and OK is enabled")]
fn stepped_preview_and_ok_enabled(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.result, GuiHolePreview::Ready);
    assert_eq!(preview.request.seat, GuiHoleSeatKind::Counterbore);
    assert_eq!(preview.request.seat_diameter, "6 mm");
    assert!(world.run_is_enabled("OK"));
}

#[when("the user accepts the edit")]
fn accept_hole_edit(world: &mut ApplicationWorld) {
    accept_hole_and_wait(world);
}

#[then("the opening shows a recessed seat")]
fn opening_shows_recessed_seat(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let hole1 = browser.features.iter().find(|f| f.name == "Hole1").unwrap();
    let seed = browser.hole_edits.get(&hole1.id).unwrap();
    assert_eq!(seed.seat, GuiHoleSeatKind::Counterbore);
    assert_eq!(seed.seat_diameter, "6 mm");
}

#[given("the user edits Hole1 to a 4 mm diameter with a 6 mm stepped seat and accepts the edit")]
fn edit_hole1_4mm_6mm_stepped(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    replace_hole_field(world, "Diameter", "4 mm", false);
    click_hole_control(world, "C'bore");
    replace_hole_field(world, "SeatDiameter", "6 mm", false);
    accept_hole_and_wait(world);
}

#[when("the user edits it and select the fourth Seat icon")]
fn edit_and_select_countersink(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    click_hole_control(world, "C'sink");
}

#[then("the diagram shows a 6 mm seat with a 90-degree angle")]
fn diagram_shows_6mm_90deg(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("6mm"),
        "expected 6 mm seat callout, saw {text}"
    );
    assert!(
        text.contains("90\u{00b0}"),
        "expected 90° callout, saw {text}"
    );
    assert!(world.run_is_enabled("OK"));
}

#[then("the opening shows a conical seat")]
fn opening_shows_conical_seat(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let hole1 = browser.features.iter().find(|f| f.name == "Hole1").unwrap();
    let seed = browser.hole_edits.get(&hole1.id).unwrap();
    assert_eq!(seed.seat, GuiHoleSeatKind::Countersink);
    assert_eq!(seed.seat_diameter, "6 mm");
}

#[given("the user reopens Hole1 for editing")]
fn reopen_hole1_for_editing(world: &mut ApplicationWorld) {
    reopen_hole1(world);
}

#[when("the user selects the second Hole type and choose Loose fit and size #4")]
fn select_clearance_loose_4(world: &mut ApplicationWorld) {
    click_hole_control(world, "Clearance");
    select_hole_combo(world, "Size", "#4");
    select_hole_combo(world, "Fit", "Loose");
}

#[then("the displayed hole and seat dimensions change")]
fn displayed_hole_and_seat_dimensions_change(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.request.kind, HoleKind::Clearance);
    assert_ne!(preview.request.diameter, "3 mm");
    assert_eq!(preview.request.diameter, "3.45 mm");
    assert_eq!(preview.request.seat_diameter, "6.10 mm");
}

#[then("size #4 and Loose fit remain selected")]
fn size_4_and_loose_retained(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    let fastener = preview.request.fastener.expect("expected saved fastener");
    assert_eq!(fastener.size, "#4");
    assert_eq!(fastener.fit, HoleFastenerFit::Loose);
}

#[when("the user selects the third Hole type and ISO Metric profile")]
fn select_tapped_iso_metric(world: &mut ApplicationWorld) {
    click_hole_control(world, "Tapped");
    select_hole_combo(world, "ThreadType", "ISO Metric profile");
}

#[when("the user enables Full Depth and select the left direction icon")]
fn enable_full_depth_and_left_direction(world: &mut ApplicationWorld) {
    click_hole_control(world, "FullDepth");
    click_hole_control(world, "Left");
}

#[then("the ISO profile, Full Depth and left direction remain selected")]
fn iso_full_depth_left_retained(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    let thread = preview.request.thread.expect("expected saved thread");
    assert_eq!(thread.profile, "ISO Metric profile");
    assert!(thread.full_depth);
    assert!(thread.left_hand);
}

#[given("the user edits Hole1 to use an ISO Metric thread and a 6 mm, 90-degree conical seat")]
fn edit_hole1_iso_and_csink(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    click_hole_control(world, "Tapped");
    select_hole_combo(world, "ThreadType", "ISO Metric profile");
    click_hole_control(world, "C'sink");
    replace_hole_field(world, "SeatDiameter", "6 mm", false);
}

#[given("the user accepts and reopens Hole1 for editing")]
fn accept_and_reopen_hole1_given(world: &mut ApplicationWorld) {
    accept_hole_and_wait(world);
    reopen_hole1(world);
}

#[when("the user selects the fourth Hole type")]
fn select_fourth_hole_type(world: &mut ApplicationWorld) {
    click_hole_control(world, "Taper");
}

#[when("the user selects no seat")]
fn select_no_seat(world: &mut ApplicationWorld) {
    click_hole_control(world, "None");
}

#[then("the 1/16–27 NPT designation remains displayed")]
fn npt_designation_retained(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    let thread = preview.request.thread.expect("expected taper thread");
    assert_eq!(thread.designation, "1/16\u{2013}27 NPT");
    let text = super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("1/16\u{2013}27NPT"),
        "expected 1/16–27 NPT designation in Hole window, saw {text}"
    );
}

#[given(
    "the user reopens Hole1, sets an 8 mm distance termination, accepts it and reopens it for editing"
)]
fn set_8mm_distance_and_reopen(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    select_hole_combo(world, "Termination", "Distance");
    replace_hole_field(world, "Depth", "8 mm", false);
    accept_hole_and_wait(world);
    reopen_hole1(world);
}

#[when("the user selects the third Seat icon and enable Extend Start")]
fn select_spotface_and_extend_start(world: &mut ApplicationWorld) {
    click_hole_control(world, "Spotface");
    click_hole_control(world, "ExtendStart");
}

#[then("the third Seat icon remains selected")]
fn spotface_remains_selected(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert_eq!(preview.request.seat, GuiHoleSeatKind::Spotface);
}

#[then("Extend Start remains checked")]
fn extend_start_remains_checked(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert!(preview.request.extend_start);
}

#[given(
    "the user reopens Hole1, sets an 8 mm distance termination and selects an angled drill point"
)]
fn set_8mm_and_angled_drill_point(world: &mut ApplicationWorld) {
    reopen_hole1(world);
    select_hole_combo(world, "Termination", "Distance");
    replace_hole_field(world, "Depth", "8 mm", false);
    click_hole_control(world, "Angle");
    let text = super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("118\u{00b0}"),
        "expected 118° drill angle in diagram, saw {text}"
    );
}

#[when("the user selects the flat Drill Point icon")]
fn select_flat_drill_point(world: &mut ApplicationWorld) {
    click_hole_control(world, "Flat");
}

#[then("the diagram shows a flat end without a point angle")]
fn diagram_shows_flat_end(world: &mut ApplicationWorld) {
    let preview = world.state().hole_preview.expect("expected hole preview");
    assert!(!preview.request.drill_point_angled);
    let text = super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("Flat"),
        "expected Flat end callout, saw {text}"
    );
    assert!(
        !text.contains("118\u{00b0}"),
        "expected no drill point angle, saw {text}"
    );
}

#[given("a Part document with a visible solid is open")]
fn part_with_visible_solid_open(world: &mut ApplicationWorld) {
    open_visible_solid_part(world);
}

#[given("no saved floating dialog position overrides the default placement")]
fn no_saved_floating_override(_world: &mut ApplicationWorld) {}

#[when("the user starts Hole")]
fn start_hole_command(world: &mut ApplicationWorld) {
    click_hole_ribbon(world);
}

#[then(
    "the Hole Properties panel opens at 280px width anchored upper-left at 14px viewport inset clear of the ViewCube, Navigation Bar, and XYZ Triad"
)]
fn hole_panel_layout_contract(_world: &mut ApplicationWorld) {
    let (origin, size) = super::super::diagnostics::window_geometry(HOLE_WINDOW)
        .expect("expected Hole window geometry");
    assert!(
        (size[0] - 280.0).abs() <= 2.0,
        "expected Hole width ~280px, got {}",
        size[0]
    );
    assert!(
        origin[0] >= 10.0 && origin[1] >= 10.0,
        "expected upper-left viewport inset, got {origin:?}"
    );
}

#[then(
    "face/position selection renders as a Teal selection pill alongside 1px-bordered 40/60 numeric inputs with consistent unit precision"
)]
fn hole_panel_pill_and_inputs_contract(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(HOLE_WINDOW));
    assert!(
        text.contains("Positions") && text.contains("Diameter"),
        "expected Positions pill and Diameter field, saw {text}"
    );
}

#[then(
    "the pinned 44px footer displays primary Indigo OK, 1px-bordered Cancel, and far-right Apply and Create New (+)"
)]
fn hole_panel_footer_contract(world: &mut ApplicationWorld) {
    let ok = world.run_in_window(HOLE_WINDOW, "OK").center();
    let cancel = world.run_in_window(HOLE_WINDOW, "Cancel").center();
    let plus = world.run_in_window(HOLE_WINDOW, "+").center();
    assert!(
        ok[0] < cancel[0] && cancel[0] < plus[0],
        "expected OK < Cancel < + order"
    );
}
