// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Create Emboss scenarios (emboss-01..emboss-07).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiEmbossPreview;
use oblikovati_model::feature::sketch_emboss::EmbossType;
use setup::{
    choose_emboss_operation, click_emboss_button, click_modal_button, click_top_planar_face,
    double_click_emboss1, focus_and_replace_emboss_field, open_solid_with_top_circle_sketch,
    open_solid_with_top_open_line_sketch, prepare_emboss1_raised, prepare_emboss1_recess,
    prepare_emboss1_wrapped_open, scene_bounds, start_emboss,
};

#[given("the available sketch contains an open line")]
fn available_sketch_has_open_line(world: &mut ApplicationWorld) {
    open_solid_with_top_open_line_sketch(world);
}

#[when("Emboss is started")]
fn emboss_is_started(world: &mut ApplicationWorld) {
    start_emboss(world);
}

#[then("Sketch Doctor identifies an open loop in the profile")]
fn sketch_doctor_identifies_open_loop(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry("Sketch Doctor").is_some(),
        "expected Sketch Doctor modal, saw {}",
        world.text()
    );
    let doctor = super::super::world::squeezed(&world.window_text("Sketch Doctor"));
    assert!(
        doctor.contains("OpenLoop") || doctor.contains("Openloop"),
        "expected Open Loop diagnostic in Sketch Doctor, saw {doctor}"
    );
    assert!(matches!(
        world.state().emboss_preview.map(|p| p.result),
        Some(GuiEmbossPreview::OpenLoop(_))
    ));
}

#[then("Cancel is selected to return to the model")]
fn cancel_sketch_doctor_returns_to_model(world: &mut ApplicationWorld) {
    click_modal_button(world, "Sketch Doctor", "Cancel");
    assert!(
        super::super::diagnostics::window_geometry("Sketch Doctor").is_none(),
        "expected Sketch Doctor modal to close"
    );
    assert!(
        super::super::diagnostics::window_geometry("Emboss").is_none(),
        "expected Emboss panel to close"
    );
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
}

#[given("a circular sketch on a planar face of a solid body")]
fn circular_sketch_on_planar_face(world: &mut ApplicationWorld) {
    open_solid_with_top_circle_sketch(world);
}

#[when("Emboss is started and the first operation is accepted at depth 1 mm")]
fn start_emboss_accept_first_op_1mm(world: &mut ApplicationWorld) {
    start_emboss(world);
    assert_eq!(
        world.state().emboss_preview.unwrap().request.emboss_type,
        EmbossType::EmbossFromFace
    );
    focus_and_replace_emboss_field(world, "Depth", "1 mm");
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::Ready
    );
    click_emboss_button(world, "OK");
    assert!(super::super::diagnostics::window_geometry("Emboss").is_none());
}

#[then("a raised circular feature appears on the face")]
fn raised_circular_feature_appears(world: &mut ApplicationWorld) {
    let (_, hi) = scene_bounds(world);
    assert!(
        hi[2] > 2.05,
        "expected raised emboss extending above Z=2.0 cm (to ~2.1 cm), saw Z_max={}",
        hi[2]
    );
}

#[then("Emboss1 appears in the model browser")]
fn emboss1_appears_in_browser(world: &mut ApplicationWorld) {
    world.expect_text("Emboss1");
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .features
            .iter()
            .any(|f| f.name == "Emboss1" && f.kind == "Emboss"),
        "expected Emboss1 in browser features"
    );
}

#[given("Emboss1 is a raised circular feature")]
fn emboss1_is_raised_circular(world: &mut ApplicationWorld) {
    prepare_emboss1_raised(world);
}

#[when("Emboss1 is double-clicked and the second operation is chosen at depth 1 mm")]
fn double_click_emboss1_choose_second_op(world: &mut ApplicationWorld) {
    double_click_emboss1(world);
    choose_emboss_operation(world, "EngravefromFace");
    focus_and_replace_emboss_field(world, "Depth", "1 mm");
    let preview = world.state().emboss_preview.unwrap();
    assert_eq!(preview.request.emboss_type, EmbossType::EngraveFromFace);
    assert_eq!(preview.result, GuiEmbossPreview::Ready);
}

#[then("the face displays a circular recess")]
fn face_displays_circular_recess(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("Emboss").is_none());
    let (_, hi) = scene_bounds(world);
    assert!(
        (hi[2] - 2.0).abs() < 0.02,
        "expected engraved recess flush at Z=2.0 cm, saw Z_max={}",
        hi[2]
    );
    let browser = world.state().browser.unwrap();
    let emboss = browser
        .features
        .iter()
        .find(|f| f.name == "Emboss1")
        .expect("expected Emboss1");
    assert_eq!(
        browser.emboss_edits[&emboss.id].emboss_type,
        EmbossType::EngraveFromFace
    );
    let has_recess_floor = world.app().window.scene_triangles.iter().any(|tri| {
        tri.positions
            .iter()
            .all(|p| (p[2] - 1.9).abs() < 0.02 && p[0] > 1.0 && p[0] < 3.0)
    });
    assert!(
        has_recess_floor,
        "expected circular recess floor triangles near Z=1.9 cm"
    );
}

#[given("the circular profile is selected in Emboss")]
fn circular_profile_selected_in_emboss(world: &mut ApplicationWorld) {
    open_solid_with_top_circle_sketch(world);
    start_emboss(world);
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::Ready
    );
}

#[when("I choose the third operation")]
fn choose_third_operation(world: &mut ApplicationWorld) {
    choose_emboss_operation(world, "Emboss/EngravefromPlane");
}

#[then("the dialog shows a Taper field initially displaying 0,0")]
fn dialog_shows_taper_0_0(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Emboss"));
    assert!(panel.contains("Taper"), "expected Taper field in {panel}");
    assert!(
        panel.contains("0,0"),
        "expected Taper initially displaying 0,0 in {panel}"
    );
    let preview = world.state().emboss_preview.unwrap();
    assert_eq!(
        preview.request.emboss_type,
        EmbossType::EmbossEngraveFromPlane
    );
    assert_eq!(preview.request.taper, "0,0");
}

#[when("I return to the first operation and reverse direction")]
fn return_to_first_op_and_reverse(world: &mut ApplicationWorld) {
    choose_emboss_operation(world, "EmbossfromFace");
    let before = world.state().emboss_preview.unwrap();
    assert!(!before.request.direction_reversed);
    click_emboss_button(world, "FlipDirection");
}

#[then("the direction marker reverses on the selected profile")]
fn direction_marker_reverses(world: &mut ApplicationWorld) {
    let after = world.state().emboss_preview.unwrap();
    assert!(after.request.direction_reversed);
    let panel = super::super::world::squeezed(&world.window_text("Emboss"));
    assert!(
        panel.contains("-Normal"),
        "expected reversed direction marker in {panel}"
    );
}

#[given("Emboss1 is a circular recess on a planar face")]
fn emboss1_is_circular_recess(world: &mut ApplicationWorld) {
    prepare_emboss1_recess(world);
}

#[when("I edit it to use the third operation at zero taper and accept")]
fn edit_to_third_op_zero_taper_and_accept(world: &mut ApplicationWorld) {
    double_click_emboss1(world);
    choose_emboss_operation(world, "Emboss/EngravefromPlane");
    assert_eq!(world.state().emboss_preview.unwrap().request.taper, "0,0");
    click_emboss_button(world, "OK");
}

#[then("the MCAD Software reports \"Edit Emboss Feature Failed\"")]
fn reports_edit_emboss_feature_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry("Edit Emboss Feature Failed").is_some(),
        "expected Edit Emboss Feature Failed modal, saw {}",
        world.text()
    );
    let modal = super::super::world::squeezed(&world.window_text("Edit Emboss Feature Failed"));
    assert!(modal.contains("EditEmbossFeatureFailed"), "{modal}");
}

#[then("Edit returns to the Emboss settings")]
fn edit_returns_to_emboss_settings(world: &mut ApplicationWorld) {
    click_modal_button(world, "Edit Emboss Feature Failed", "Edit");
    assert!(
        super::super::diagnostics::window_geometry("Edit Emboss Feature Failed").is_none(),
        "expected failure modal to close"
    );
    assert!(
        super::super::diagnostics::window_geometry("Emboss").is_some(),
        "expected Emboss settings panel to remain open"
    );
}

#[given("Emboss1 is open for editing with a circular profile")]
fn emboss1_open_for_editing(world: &mut ApplicationWorld) {
    prepare_emboss1_recess(world);
    double_click_emboss1(world);
}

#[when("I select the raised operation and enable Wrap to Face")]
fn select_raised_and_enable_wrap(world: &mut ApplicationWorld) {
    choose_emboss_operation(world, "EmbossfromFace");
    click_emboss_button(world, "WraptoFace");
}

#[then("a Face selector appears and OK is unavailable until a face is selected")]
fn face_selector_appears_ok_disabled(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Emboss"));
    assert!(
        panel.contains("SelectFace"),
        "expected Face selector in {panel}"
    );
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::MissingWrapFace
    );
    assert!(!world.run_is_enabled("OK"));
}

#[when("I select the planar face and accept at 1 mm depth")]
fn select_planar_face_and_accept_1mm(world: &mut ApplicationWorld) {
    click_top_planar_face(world);
    let panel = super::super::world::squeezed(&world.window_text("Emboss"));
    assert!(
        panel.contains("1Face"),
        "expected selected face in Emboss panel, saw {panel}"
    );
    focus_and_replace_emboss_field(world, "Depth", "1 mm");
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::Ready
    );
    assert!(world.run_is_enabled("OK"));
    click_emboss_button(world, "OK");
    assert!(super::super::diagnostics::window_geometry("Emboss").is_none());
}

#[given("Emboss1 is open with the raised, wrapped definition")]
fn emboss1_open_raised_wrapped(world: &mut ApplicationWorld) {
    prepare_emboss1_wrapped_open(world);
}

#[when("I replace its depth with zero and click OK")]
fn replace_depth_zero_and_click_ok(world: &mut ApplicationWorld) {
    focus_and_replace_emboss_field(world, "Depth", "0");
    assert!(matches!(
        world.state().emboss_preview.map(|p| p.result),
        Some(GuiEmbossPreview::RangeError(_))
    ));
    click_emboss_button(world, "OK");
}

#[then("the MCAD Software reports that the expression is out of the expected range")]
fn reports_expression_out_of_range(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry("Emboss Warning").is_some(),
        "expected Emboss Warning modal, saw {}",
        world.text()
    );
    let modal = super::super::world::squeezed(&world.window_text("Emboss Warning"));
    assert!(
        modal.contains("Theexpressionisoutoftheexpectedrange"),
        "expected out-of-range warning text, saw {modal}"
    );
}

#[when("I dismiss the warning, replace zero with 1 and click Cancel")]
fn dismiss_warning_replace_1_and_cancel(world: &mut ApplicationWorld) {
    click_modal_button(world, "Emboss Warning", "OK");
    assert!(super::super::diagnostics::window_geometry("Emboss Warning").is_none());
    focus_and_replace_emboss_field(world, "Depth", "1");
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::Ready
    );
    click_emboss_button(world, "Cancel");
}

#[then("the Emboss edit dialog closes")]
fn emboss_edit_dialog_closes(_world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("Emboss").is_none());
}
