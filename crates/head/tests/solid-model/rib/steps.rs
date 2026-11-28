// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Create Rib scenarios (rib-01..rib-05).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiRibPreview;
use oblikovati_model::feature::sketch_rib::{RibDirection, RibThicknessPlane};
use setup::{
    CREATE_FAILED_WINDOW, RANGE_WARNING_WINDOW, RIB_WINDOW, SKETCH_TIP_WINDOW, click_modal_control,
    click_rib_control, click_rib_ribbon, ensure_sketch_visible, open_disposable_rib_fixture,
    replace_rib_thickness, select_rib_combo, shell_block_top_face,
};

#[given("the part has no visible unadaptive sketch")]
fn part_has_no_visible_unadaptive_sketch(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .sketches
            .iter()
            .all(|s| !s.visible || s.consumed_by.is_some()),
        "expected no visible unadaptive sketch, saw {:?}",
        browser
            .sketches
            .iter()
            .map(|s| (&s.name, s.visible, s.consumed_by))
            .collect::<Vec<_>>()
    );
}

#[when("Rib is opened")]
fn rib_is_opened(world: &mut ApplicationWorld) {
    click_rib_ribbon(world);
}

#[then("the MCAD Software displays a tip requesting a visible unadaptive sketch or a new sketch")]
fn tip_requesting_visible_sketch(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(SKETCH_TIP_WINDOW).is_some(),
        "expected sketch-required tip modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(SKETCH_TIP_WINDOW));
    assert!(
        text.contains("visible") && text.contains("sketch"),
        "expected visible-sketch tip text, saw {text}"
    );
    assert!(
        super::super::diagnostics::window_geometry(RIB_WINDOW).is_none(),
        "expected Rib panel to stay closed without a visible sketch"
    );
}

#[given(expr = "a disposable copy of `{word}` is open")]
fn open_backtick_fixture(world: &mut ApplicationWorld, relative: String) {
    open_disposable_rib_fixture(world, &relative);
}

#[given("the user shells the block by removing its top face")]
fn user_shells_block_top_face(world: &mut ApplicationWorld) {
    shell_block_top_face(world);
}

#[given("the single-line Sketch2 on the XZ origin plane is visible")]
fn sketch2_on_xz_visible(world: &mut ApplicationWorld) {
    ensure_sketch_visible(world, "Sketch2");
    assert!(
        world
            .state()
            .rib_sketches
            .iter()
            .any(|s| s.name == "Sketch2" && s.curves.len() == 1),
        "expected single-line Sketch2 among rib candidates, saw {:?}",
        world.state().rib_sketches
    );
}

#[when("Rib is opened and acceptance is attempted with the displayed 1 mm thickness")]
fn open_rib_and_attempt_accept_1mm(world: &mut ApplicationWorld) {
    click_rib_ribbon(world);
    assert!(
        super::super::diagnostics::window_geometry(RIB_WINDOW).is_some(),
        "expected Rib dialog to open"
    );
    click_rib_control(world, "SelectProfile");
    let preview = world.state().rib_preview.expect("expected rib preview");
    assert_eq!(preview.request.thickness, "1 mm");
    assert_eq!(
        preview.request.direction,
        RibDirection::ParallelToSketchPlane
    );
    assert!(matches!(preview.result, GuiRibPreview::Failed(_)));
    click_rib_control(world, "OK");
}

#[then("the MCAD Software reports Create rib feature failed and offers Edit and Cancel")]
fn reports_create_rib_failed_edit_cancel(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(CREATE_FAILED_WINDOW).is_some(),
        "expected Create rib feature failed modal, saw {}",
        world.text()
    );
    let modal = super::super::world::squeezed(&world.window_text(CREATE_FAILED_WINDOW));
    assert!(modal.contains("Createribfeaturefailed"), "{modal}");
    assert!(
        modal.contains("Edit") && modal.contains("Cancel"),
        "{modal}"
    );
}

#[when("Edit is chosen and the other orientation is selected before acceptance is attempted")]
fn choose_edit_switch_orientation_accept(world: &mut ApplicationWorld) {
    click_modal_control(world, CREATE_FAILED_WINDOW, "Edit");
    assert!(super::super::diagnostics::window_geometry(CREATE_FAILED_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(RIB_WINDOW).is_some());
    click_rib_control(world, "NormaltoSketchPlane");
    let preview = world.state().rib_preview.expect("expected rib preview");
    assert_eq!(preview.request.direction, RibDirection::NormalToSketchPlane);
    assert!(matches!(preview.result, GuiRibPreview::Failed(_)));
    click_rib_control(world, "OK");
}

#[then("the MCAD Software again reports Create rib feature failed")]
fn again_reports_create_rib_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(CREATE_FAILED_WINDOW).is_some(),
        "expected Create rib feature failed modal on second orientation, saw {}",
        world.text()
    );
}

#[when("the user cancels")]
fn user_cancels_rib_failure(world: &mut ApplicationWorld) {
    if super::super::diagnostics::window_geometry("Ruled Surface").is_some() {
        let center = world.run_in_window("Ruled Surface", "Cancel").center();
        world.click(super::super::diagnostics::pointer_pixels(center));
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    if super::super::diagnostics::window_geometry("Shell").is_some() {
        let center = world.run_in_window("Shell", "Cancel").center();
        world.click(super::super::diagnostics::pointer_pixels(center));
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    click_modal_control(world, CREATE_FAILED_WINDOW, "Cancel");
}

#[then("the dialog closes without a Rib feature in the browser")]
fn dialog_closes_without_rib_feature(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(CREATE_FAILED_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(RIB_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().all(|f| f.kind != "Rib"),
        "expected no Rib feature in browser, saw {:?}",
        browser.features
    );
}

#[given("the open-line sketch is visible and the Rib dialog is open without a selected profile")]
fn open_line_visible_rib_open_no_profile(world: &mut ApplicationWorld) {
    let second_sketch = world
        .state()
        .browser
        .expect("expected browser")
        .sketches
        .get(1)
        .expect("expected second sketch")
        .name
        .clone();
    ensure_sketch_visible(world, &second_sketch);
    click_rib_ribbon(world);
    assert!(super::super::diagnostics::window_geometry(RIB_WINDOW).is_some());
    let preview = world.state().rib_preview.expect("expected rib preview");
    assert_eq!(preview.result, GuiRibPreview::MissingProfile);
}

#[when("the user opens the Boss tab")]
fn user_opens_boss_tab(world: &mut ApplicationWorld) {
    click_rib_control(world, "Boss");
}

#[then("Centers, Select All, Diameter, Offset and Draft Angle controls are shown")]
fn boss_controls_shown(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text(RIB_WINDOW));
    for required in ["Centers", "SelectAll", "Diameter", "Offset", "DraftAngle"] {
        assert!(
            panel.contains(required),
            "expected {required} on Boss tab, saw {panel}"
        );
    }
}

#[when("the user opens the Draft tab")]
fn user_opens_draft_tab(world: &mut ApplicationWorld) {
    click_rib_control(world, "Draft");
}

#[then("Hold Thickness offers At Top and At Root and a Draft Angle field is shown")]
fn draft_controls_shown(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text(RIB_WINDOW));
    for required in ["HoldThickness", "AtTop", "AtRoot", "DraftAngle", "T(AtTop)"] {
        assert!(
            panel.contains(required),
            "expected {required} on Draft tab, saw {panel}"
        );
    }
    world.observed_rows = Some(panel);
}

#[when("the user selects At Root")]
fn user_selects_at_root(world: &mut ApplicationWorld) {
    click_rib_control(world, "AtRoot");
}

#[then("the diagram changes to illustrate thickness held at the root")]
fn diagram_changes_to_at_root(world: &mut ApplicationWorld) {
    let before = world.observed_rows.clone().unwrap_or_default();
    let after = super::super::world::squeezed(&world.window_text(RIB_WINDOW));
    assert!(
        before.contains("T(AtTop)"),
        "expected At Top diagram before"
    );
    assert!(
        after.contains("T(AtRoot)") && !after.contains("T(AtTop)"),
        "expected diagram to change to At Root, saw {after}"
    );
    let preview = world.state().rib_preview.expect("expected rib preview");
    assert_eq!(preview.request.thickness_plane, RibThicknessPlane::AtRoot);
}

#[then("OK remains disabled without a profile")]
fn ok_remains_disabled_without_profile(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled("OK"),
        "expected OK to stay disabled without a profile"
    );
}

#[given("Sketch3 is visible and attached to the block side face on Y=0")]
fn sketch3_visible_on_side_face(world: &mut ApplicationWorld) {
    ensure_sketch_visible(world, "Sketch3");
    let sketch = world
        .state()
        .browser
        .expect("expected browser")
        .sketches
        .into_iter()
        .find(|s| s.name == "Sketch3")
        .expect("expected Sketch3");
    assert!(sketch.visible, "expected Sketch3 visible");
}

#[given("its open line runs from (20, 0, 10) mm on the face to (50, 0, 10) mm beyond its edge")]
fn sketch3_open_line_extends_beyond_edge(world: &mut ApplicationWorld) {
    assert!(
        world
            .state()
            .rib_sketches
            .iter()
            .any(|s| s.name == "Sketch3" && !s.curves.is_empty()),
        "expected open line in Sketch3, saw {:?}",
        world.state().rib_sketches
    );
}

#[when("the user starts Rib and selects the Sketch3 line and the block as the solid")]
fn start_rib_select_sketch3_and_block(world: &mut ApplicationWorld) {
    let block_solid = world
        .state()
        .browser
        .expect("expected browser")
        .bodies
        .first()
        .expect("expected block solid")
        .name
        .clone();
    click_rib_ribbon(world);
    assert!(super::super::diagnostics::window_geometry(RIB_WINDOW).is_some());
    click_rib_control(world, "SelectProfile");
    if world.state().rib_sketches.len() > 1 {
        select_rib_combo(world, "Sketch", "Sketch3");
    }
    select_rib_combo(world, "Solids", &block_solid);
}

#[when("the user accepts with 1 mm thickness and Extend Profile enabled")]
fn accept_with_1mm_and_extend_profile(world: &mut ApplicationWorld) {
    let preview = world.state().rib_preview.expect("expected rib preview");
    assert_eq!(preview.request.thickness, "1 mm");
    assert!(preview.request.extend_profile);
    assert!(matches!(preview.result, GuiRibPreview::Failed(_)));
    click_rib_control(world, "OK");
}

#[then(r#"the MCAD Software reports "Create rib feature failed""#)]
fn reports_create_rib_feature_failed_quoted(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(CREATE_FAILED_WINDOW).is_some(),
        "expected Create rib feature failed modal, saw {}",
        world.text()
    );
}

#[then("cancelling the error returns to the model with the sketch visible")]
fn cancel_error_returns_with_sketch_visible(world: &mut ApplicationWorld) {
    click_modal_control(world, CREATE_FAILED_WINDOW, "Cancel");
    assert!(super::super::diagnostics::window_geometry(CREATE_FAILED_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(RIB_WINDOW).is_none());
    let sketch3 = world
        .state()
        .browser
        .expect("expected browser")
        .sketches
        .into_iter()
        .find(|s| s.name == "Sketch3")
        .expect("expected Sketch3");
    assert!(
        sketch3.visible,
        "expected Sketch3 to remain visible after cancel"
    );
}

#[given("the second sketch containing an open line is visible and the Rib dialog is open")]
fn second_sketch_visible_rib_open(world: &mut ApplicationWorld) {
    open_line_visible_rib_open_no_profile(world);
}

#[when("the user replaces Thickness with zero")]
fn replace_thickness_with_zero(world: &mut ApplicationWorld) {
    replace_rib_thickness(world, "0");
}

#[then("zero is displayed in red")]
fn zero_is_displayed_in_red(world: &mut ApplicationWorld) {
    let list = super::super::diagnostics::window_draw_list(RIB_WINDOW).expect("Rib draw list");
    let zero_run = world.frame.runs.iter().find(|run| {
        run.text == "0" && run.color[0] > 180 && run.color[1] < 100 && run.color[2] < 100
    });
    assert!(
        zero_run.is_some(),
        "expected red '0' glyph run in Rib window (list={list}), saw {:?}",
        world
            .frame
            .runs
            .iter()
            .filter(|r| r.text == "0")
            .map(|r| (&r.text, r.color, r.list))
            .collect::<Vec<_>>()
    );
}

#[when("the user clicks OK and then cancels while the dialog remains open")]
fn clicks_ok_and_cancels_while_open(world: &mut ApplicationWorld) {
    click_rib_control(world, "OK");
    assert!(
        super::super::diagnostics::window_geometry(RIB_WINDOW).is_some(),
        "expected Rib dialog to remain open after OK with zero thickness"
    );
    if super::super::diagnostics::window_geometry(RANGE_WARNING_WINDOW).is_none() {
        click_rib_control(world, "Cancel");
    }
}

#[then("the MCAD Software displays an out-of-range expression warning")]
fn displays_out_of_range_warning(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(RANGE_WARNING_WINDOW).is_some(),
        "expected Rib Warning modal, saw {}",
        world.text()
    );
    let modal = super::super::world::squeezed(&world.window_text(RANGE_WARNING_WINDOW));
    assert!(
        modal.contains("Theexpressionisoutoftheexpectedrange"),
        "expected out-of-range message, saw {modal}"
    );
}

#[when("the user dismisses the warning, replaces zero with 1, and clicks Cancel")]
fn dismiss_warning_restore_1_cancel(world: &mut ApplicationWorld) {
    click_modal_control(world, RANGE_WARNING_WINDOW, "OK");
    assert!(super::super::diagnostics::window_geometry(RANGE_WARNING_WINDOW).is_none());
    replace_rib_thickness(world, "1");
    click_rib_control(world, "Cancel");
}

#[then("the Rib dialog closes")]
fn rib_dialog_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(RIB_WINDOW).is_none(),
        "expected Rib dialog to close"
    );
}
