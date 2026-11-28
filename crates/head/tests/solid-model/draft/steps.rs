// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Draft scenarios (draft-01..draft-06, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiDraftPreview;
use oblikovati_model::feature::{
    draft::{DraftAngleMode, DraftMode, DraftPartingTool},
    hole::HoleOriginPlane,
};
use setup::{
    CREATE_FAILURE_WINDOW, DRAFT_WINDOW, EDIT_FAILURE_WINDOW, accept_draft_and_wait,
    click_draft_control, click_draft_ribbon, click_modal_control,
    click_shelled_rim_and_outer_y0_face, click_side_face_for_draft,
    click_top_face_for_pull_direction, pick_xz_origin_plane, reopen_face_draft1,
    replace_draft_angle, solid_volume,
};

#[given("the part contains a 2 mm inside shell with its top face removed")]
fn verify_shelled_hollow_box(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Shell1"),
        "expected Shell1 in browser: {:?}",
        browser.features
    );
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 11);
}

#[given("Face Draft is open on the shelled part")]
fn open_face_draft_on_shelled_part(world: &mut ApplicationWorld) {
    click_draft_ribbon(world);
}

#[when(
    "I select the planar rim at Z = 20 mm for Pull Direction and the outer face at Y = 0 mm for Faces"
)]
fn select_shelled_rim_and_y0_face(world: &mut ApplicationWorld) {
    click_shelled_rim_and_outer_y0_face(world);
}

#[when("I accept the 2.5 degree draft")]
fn accept_2_5_deg_draft_triggering_failure(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .draft_preview
        .expect("expected draft preview state");
    assert_eq!(preview.request.angle, "2.5 deg");
    click_draft_control(world, "OK");
    world.pump(Vec::new());
}

#[then("the MCAD Software reports that face draft creation failed")]
fn reports_face_draft_creation_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(CREATE_FAILURE_WINDOW).is_some(),
        "expected Create Face Draft modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(CREATE_FAILURE_WINDOW));
    assert!(
        text.contains("Facedraftcreationfailed"),
        "expected failure headline in Create Face Draft modal, saw {text}"
    );
}

#[then("the Face Draft dialog returns with the selection visible")]
fn face_draft_dialog_returns_with_selection(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(CREATE_FAILURE_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(DRAFT_WINDOW).is_some());
    let preview = world
        .state()
        .draft_preview
        .expect("expected retained draft preview");
    assert!(preview.request.pull_direction.is_some());
    assert_eq!(preview.request.faces.len(), 1);
    let text = super::super::world::squeezed(&world.window_text(DRAFT_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected selection visible in Draft dialog, saw {text}"
    );
}

#[then("the dialog closes without adding a Draft feature")]
fn dialog_closes_without_draft_feature(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(DRAFT_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        !browser
            .features
            .iter()
            .any(|f| f.name.starts_with("FaceDraft")),
        "expected no Draft feature added, saw {:?}",
        browser.features
    );
}

#[given("the part contains a rectangular extruded block")]
fn verify_rectangular_extruded_block(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
}

#[given("the user opens the Draft command in face-draft mode")]
#[given("the user opens the Draft command")]
fn user_opens_draft_command(world: &mut ApplicationWorld) {
    click_draft_ribbon(world);
}

#[when("I select its top face for Pull Direction and a side face for Faces")]
fn select_top_face_and_side_face(world: &mut ApplicationWorld) {
    click_top_face_for_pull_direction(world);
    click_side_face_for_draft(world);
    let preview = world.state().draft_preview.expect("expected draft preview");
    assert_eq!(preview.result, GuiDraftPreview::Ready);
}

#[when("I accept the 2.5 degree preview")]
fn accept_2_5_degree_preview(world: &mut ApplicationWorld) {
    let preview = world.state().draft_preview.expect("expected draft preview");
    assert_eq!(preview.request.angle, "2.5 deg");
    accept_draft_and_wait(world);
}

#[then("FaceDraft1 appears in the model browser")]
#[then("FaceDraft1 appears in the browser")]
fn face_draft1_appears_in_browser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "FaceDraft1"),
        "expected FaceDraft1 in browser: {:?}",
        browser.features
    );
}

#[when("I reopen it and change Draft Angle to 5 degrees")]
fn reopen_face_draft1_and_change_to_5_deg(world: &mut ApplicationWorld) {
    reopen_face_draft1(world);
    replace_draft_angle(world, "5 deg");
    let preview = world.state().draft_preview.expect("expected draft preview");
    assert_eq!(preview.request.angle, "5 deg");
    assert_eq!(preview.result, GuiDraftPreview::Ready);
    assert!(world.state().extrude_preview_active);
}

#[then("the preview changes and accepting the edit updates the face")]
fn preview_changes_and_accepting_updates_face(world: &mut ApplicationWorld) {
    let before_vol = solid_volume(world);
    accept_draft_and_wait(world);
    let after_vol = solid_volume(world);
    assert!(
        (after_vol - before_vol).abs() > 1e-3,
        "expected 5° draft edit to change solid volume ({before_vol} -> {after_vol})"
    );
}

#[given("the user opens the Draft command and selects the second Draft mode")]
fn open_draft_and_select_fixed_plane(world: &mut ApplicationWorld) {
    click_draft_ribbon(world);
    click_draft_control(world, "FixedPlane");
}

#[when("I choose the top face as Fixed Plane and a side face to draft")]
fn choose_top_face_fixed_plane_and_side_face(world: &mut ApplicationWorld) {
    click_top_face_for_pull_direction(world);
    click_side_face_for_draft(world);
    let preview = world.state().draft_preview.expect("expected draft preview");
    assert_eq!(preview.request.mode, DraftMode::FixedPlane);
    assert_eq!(preview.result, GuiDraftPreview::Ready);
}

#[given("the user opens the Draft command and selects the third Draft mode")]
fn open_draft_and_select_parting_tool(world: &mut ApplicationWorld) {
    click_draft_ribbon(world);
    click_draft_control(world, "PartingTool");
}

#[when("I select a pull direction and an origin work plane as Parting Tool")]
fn select_pull_direction_and_origin_plane(world: &mut ApplicationWorld) {
    click_top_face_for_pull_direction(world);
    pick_xz_origin_plane(world);
}

#[then("the Faces selector becomes active")]
fn faces_selector_becomes_active(world: &mut ApplicationWorld) {
    let preview = world.state().draft_preview.expect("expected draft preview");
    assert_eq!(preview.request.mode, DraftMode::PartingTool);
    assert!(preview.request.pull_direction.is_some());
    assert_eq!(
        preview.request.parting_tool,
        Some(DraftPartingTool::OriginPlane(HoleOriginPlane::Xz))
    );
    let text = super::super::world::squeezed(&world.window_text(DRAFT_WINDOW));
    assert!(
        text.contains("PickFaces..."),
        "expected active Faces selector pill in Draft dialog, saw {text}"
    );
}

#[when("I cancel the incomplete setup")]
fn cancel_incomplete_draft_setup(world: &mut ApplicationWorld) {
    click_draft_control(world, "Cancel");
}

#[then("no additional Draft feature appears")]
fn no_additional_draft_feature(world: &mut ApplicationWorld) {
    dialog_closes_without_draft_feature(world);
}

#[given("the rectangular block has a top planar face and an XZ origin work plane")]
fn verify_block_and_origin(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1);
    assert!(browser.has_origin);
}

#[when("I expand Draft options and choose Relative")]
fn expand_draft_options_and_choose_relative(world: &mut ApplicationWorld) {
    click_draft_control(world, "Options>>");
    click_draft_control(world, "Relative");
}

#[when(
    "I select parting-tool mode, the top face for Pull Direction, the XZ plane for Parting Tool, and a side face"
)]
fn select_parting_tool_top_xz_side(world: &mut ApplicationWorld) {
    click_draft_control(world, "PartingTool");
    click_top_face_for_pull_direction(world);
    pick_xz_origin_plane(world);
    click_side_face_for_draft(world);
    let preview = world.state().draft_preview.expect("expected draft preview");
    assert_eq!(preview.request.mode, DraftMode::PartingTool);
    assert_eq!(preview.request.angle_mode, DraftAngleMode::Relative);
    assert_eq!(preview.result, GuiDraftPreview::Ready);
}

#[when("I reopen the feature and expand its options")]
fn reopen_face_draft1_and_expand_options(world: &mut ApplicationWorld) {
    reopen_face_draft1(world);
    click_draft_control(world, "Options>>");
}

#[then("the parting-tool controls and Relative angle setting are shown")]
fn parting_tool_and_relative_shown(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .draft_preview
        .expect("expected reopened draft preview");
    assert_eq!(preview.request.mode, DraftMode::PartingTool);
    assert_eq!(
        preview.request.parting_tool,
        Some(DraftPartingTool::OriginPlane(HoleOriginPlane::Xz))
    );
    assert_eq!(preview.request.angle_mode, DraftAngleMode::Relative);
    let text = super::super::world::squeezed(&world.window_text(DRAFT_WINDOW));
    assert!(
        text.contains("PartingTool") && text.contains("XZPlane") && text.contains("Relative"),
        "expected parting-tool controls and Relative setting, saw {text}"
    );
}

#[given(
    "the user creates a relative parting-tool draft using the top face for Pull Direction, the XZ origin plane for Parting Tool, a side face, and a 2.5 degree angle"
)]
fn create_relative_parting_tool_draft(world: &mut ApplicationWorld) {
    click_draft_ribbon(world);
    click_draft_control(world, "Options>>");
    click_draft_control(world, "Relative");
    select_parting_tool_top_xz_side(world);
    accept_2_5_degree_preview(world);
    face_draft1_appears_in_browser(world);
}

#[given("the user opens the resulting FaceDraft1 for editing")]
fn open_resulting_face_draft1(world: &mut ApplicationWorld) {
    reopen_face_draft1(world);
}

#[when("the angle is replaced with zero and OK is clicked")]
fn replace_angle_with_zero_and_click_ok(world: &mut ApplicationWorld) {
    replace_draft_angle(world, "0");
    click_draft_control(world, "OK");
    world.pump(Vec::new());
}

#[then("the MCAD Software reports that editing the face draft failed")]
fn reports_editing_face_draft_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(EDIT_FAILURE_WINDOW).is_some(),
        "expected Edit Face Draft failure modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(EDIT_FAILURE_WINDOW));
    assert!(
        text.contains("Editingthefacedraftfailed"),
        "expected edit failure message in Edit Face Draft modal, saw {text}"
    );
}

#[then("the expanded detail reports errors during the part update")]
fn expanded_detail_reports_errors_during_part_update(world: &mut ApplicationWorld) {
    click_modal_control(world, EDIT_FAILURE_WINDOW, "Details>>");
    let text = super::super::world::squeezed(&world.window_text(EDIT_FAILURE_WINDOW));
    assert!(
        text.contains("errorsduringthepartupdate"),
        "expected 'errors during the part update' in expanded detail, saw {text}"
    );
}

#[when("I cancel the failure and reopen FaceDraft1")]
fn cancel_failure_and_reopen_face_draft1(world: &mut ApplicationWorld) {
    click_modal_control(world, EDIT_FAILURE_WINDOW, "Cancel");
    assert!(super::super::diagnostics::window_geometry(EDIT_FAILURE_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(DRAFT_WINDOW).is_none());
    reopen_face_draft1(world);
}

#[then("the draft angle is 2.5 degrees")]
fn verify_draft_angle_is_2_5_deg(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .draft_preview
        .expect("expected reopened draft preview");
    assert_eq!(preview.request.angle, "2.5 deg");
}

#[then("the Draft dialog closes")]
fn draft_dialog_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DRAFT_WINDOW).is_none(),
        "expected Draft dialog to close"
    );
}
