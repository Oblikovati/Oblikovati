// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Thread external scenarios (thread-01..thread-07, #3918).
mod internal_steps;
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiThreadPreview;
use oblikovati_model::feature::thread::ThreadHandedness;
use setup::{
    EDIT_FAILED_MODAL, THREAD_WINDOW, accept_thread_and_wait, click_thread_control,
    click_thread_ribbon_raw, click_threadable_face, create_tapered_thread1, inspect_matching_faces,
    open_thread_combo, open_thread_panel, overlay_min_coord, overlay_span_extent,
    popup_option_texts, reopen_thread1, replace_thread_field, select_thread_combo, set_checkbox,
    thread_overlay_triangles,
};

#[given("the rectangular block has no cylindrical or conical faces")]
fn block_has_no_threadable_faces(world: &mut ApplicationWorld) {
    assert!(!world.state().has_threadable_faces);
}

#[when("the user opens Thread")]
fn user_opens_thread(world: &mut ApplicationWorld) {
    click_thread_ribbon_raw(world);
}

#[then("the MCAD Software reports that it could not find cylindrical or conical faces to thread")]
fn reports_no_threadable_faces(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.text());
    assert!(
        text.contains("Couldnotfindcylindricalorconicalfacestothread"),
        "expected no-threadable-faces notice, saw {}",
        world.text()
    );
}

#[given("Revolution1 produces a tapered solid with an outer conical face")]
fn revolution_has_outer_conical_face(world: &mut ApplicationWorld) {
    assert!(world.state().has_threadable_faces);
    let cones = inspect_matching_faces(world, |info| info.conical && !info.internal);
    assert!(!cones.is_empty(), "expected outer conical face");
}

#[given("the resulting tapered solid has an eligible outer face")]
fn tapered_solid_has_eligible_outer_face(world: &mut ApplicationWorld) {
    let cones = inspect_matching_faces(world, |info| info.conical && !info.internal);
    assert!(!cones.is_empty());
}

#[when("the user opens Thread and selects that face")]
fn open_thread_and_select_outer_conical_face(world: &mut ApplicationWorld) {
    open_thread_panel(world);
    click_threadable_face(world, |info| info.conical && !info.internal);
}

#[then("the panel shows ISO Taper External, size R 6, and a 10 mm depth preview")]
fn panel_shows_iso_taper_r6_10mm(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.result, GuiThreadPreview::Ready);
    assert_eq!(preview.request.spec.thread_type, "ISO Taper External");
    assert_eq!(preview.request.spec.size, "R 6");
    assert_eq!(preview.request.depth, "10 mm");
    assert!(!preview.request.full_depth);
    let overlays = thread_overlay_triangles(world);
    assert!(!overlays.is_empty(), "expected 10 mm depth preview bands");
    let span = overlay_span_extent(&overlays);
    assert!(
        span < 2.0,
        "expected 10 mm (~1.0 cm) partial preview span, got {span}"
    );
}

#[when("the user accepts the settings")]
#[when("the user accepts")]
fn accept_thread_settings(world: &mut ApplicationWorld) {
    if super::super::diagnostics::window_geometry("Split").is_some() {
        super::solid_split::accept_split_from_shared(world);
        return;
    }
    accept_thread_and_wait(world);
}

#[then("Thread1 appears in the browser and the thread is displayed at one end")]
fn thread1_appears_and_displayed_at_one_end(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Thread1"));
    let overlays = thread_overlay_triangles(world);
    assert!(!overlays.is_empty(), "expected visible committed thread");
    let span = overlay_span_extent(&overlays);
    assert!(
        span < 2.0,
        "expected partial thread at one end (< 2 cm), got {span}"
    );
}

#[when("the user reopens Thread1, chooses the left-hand direction and full depth, and accepts")]
fn reopen_choose_left_hand_full_depth_accept(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    click_thread_control(world, "LeftHand");
    set_checkbox(world, "Full Depth", true);
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.request.spec.handedness, ThreadHandedness::LeftHand);
    assert!(preview.request.full_depth);
    accept_thread_and_wait(world);
}

#[then("the displayed thread extends along the selected face")]
fn displayed_thread_extends_along_face(world: &mut ApplicationWorld) {
    let overlays = thread_overlay_triangles(world);
    assert!(!overlays.is_empty());
    let span = overlay_span_extent(&overlays);
    assert!(
        span > 3.5,
        "expected full-depth thread extending along ~4.9 cm face, got {span}"
    );
}

#[given(
    "the user creates Thread1 on the resulting outer conical face with a compatible external tapered thread"
)]
fn create_compatible_tapered_thread1(world: &mut ApplicationWorld) {
    create_tapered_thread1(world);
}

#[given("the user opens Thread1, enables full depth, accepts and reopens it for editing")]
fn enable_full_depth_accept_reopen(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    set_checkbox(world, "Full Depth", true);
    accept_thread_and_wait(world);
    reopen_thread1(world);
}

#[when("the user disables full depth and enters 20 mm depth and 5 mm offset")]
fn disable_full_depth_enter_20_and_5(world: &mut ApplicationWorld) {
    set_checkbox(world, "Full Depth", false);
    replace_thread_field(world, "Depth", "20 mm");
    replace_thread_field(world, "Offset", "5 mm");
}

#[then("the preview shows a shorter threaded region separated from the end of the face")]
fn preview_shows_shorter_offset_region(world: &mut ApplicationWorld) {
    let overlays = thread_overlay_triangles(world);
    assert!(!overlays.is_empty(), "expected partial offset preview");
    let span = overlay_span_extent(&overlays);
    assert!(
        (1.5..=2.5).contains(&span),
        "expected ~20 mm (2.0 cm) partial thread span, got {span}"
    );
    let min_x = overlay_min_coord(&overlays, 0);
    assert!(
        min_x > 0.35,
        "expected 5 mm (0.5 cm) offset separating thread from x=0 end, got min_x={min_x}"
    );
}

#[when("the user disables Display thread in model and accepts")]
fn disable_display_and_accept(world: &mut ApplicationWorld) {
    set_checkbox(world, "Display thread in model", false);
    accept_thread_and_wait(world);
}

#[then("Thread1 remains in the browser and the thread display is absent from the model")]
fn thread1_remains_display_absent(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Thread1"));
    assert!(
        thread_overlay_triangles(world).is_empty(),
        "expected no thread display overlay when Display thread in model is unchecked"
    );
}

#[when("the user reopens Thread1")]
#[given("the user reopens Thread1 for editing")]
#[given("Thread1 is reopened for editing")]
#[given("Thread1 is open for editing")]
fn reopen_thread1_step(world: &mut ApplicationWorld) {
    reopen_thread1(world);
}

#[then("depth is 20 mm, offset is 5 mm, and Display thread in model is unchecked")]
fn depth_20_offset_5_display_unchecked(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.request.depth, "20 mm");
    assert_eq!(preview.request.offset, "5 mm");
    assert!(!preview.request.display_in_model);
    assert!(!preview.request.full_depth);
}

#[given(
    "the user edits Thread1 to disable Full Depth, set depth to 20 mm and offset to 5 mm, disable Display thread in model, and accept"
)]
fn edit_thread1_20mm_5mm_hidden(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    set_checkbox(world, "Full Depth", false);
    replace_thread_field(world, "Depth", "20 mm");
    replace_thread_field(world, "Offset", "5 mm");
    set_checkbox(world, "Display thread in model", false);
    accept_thread_and_wait(world);
}

#[when("the user selects JIS Pipe Threads")]
fn select_jis_pipe_threads(world: &mut ApplicationWorld) {
    select_thread_combo(world, "Thread Type", "JIS Pipe Threads");
}

#[then("size 13.157, designation G 1/4 and class A are shown")]
fn shows_jis_13_157_g1_4_a(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.request.spec.size, "13.157");
    assert_eq!(preview.request.spec.designation, "G 1/4");
    assert_eq!(preview.request.spec.class, "A");
}

#[when("the user selects size 16.662")]
fn select_size_16_662(world: &mut ApplicationWorld) {
    select_thread_combo(world, "Size", "16.662");
}

#[then("designation changes to G 3/8 and class A remains selected")]
fn designation_g3_8_class_a(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.request.spec.designation, "G 3/8");
    assert_eq!(preview.request.spec.class, "A");
}

#[then("the class list offers A and B")]
fn class_list_offers_a_and_b(world: &mut ApplicationWorld) {
    open_thread_combo(world, "Class");
    let options = popup_option_texts(world);
    assert!(
        options.iter().any(|o| o == "A") && options.iter().any(|o| o == "B"),
        "expected class list to offer A and B, saw {options:?}"
    );
}

#[given("the user reopens Thread1 and configures JIS Pipe Threads, size 16.662 and class B")]
fn reopen_and_configure_jis_16_662_b(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    select_thread_combo(world, "Thread Type", "JIS Pipe Threads");
    select_thread_combo(world, "Size", "16.662");
    select_thread_combo(world, "Class", "B");
}

#[when("the user enables thread display and accepts the edit")]
fn enable_display_and_accept_edit(world: &mut ApplicationWorld) {
    set_checkbox(world, "Display thread in model", true);
    click_thread_control(world, "OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the MCAD Software reports Edit Thread Feature Failed")]
fn reports_edit_thread_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(EDIT_FAILED_MODAL).is_some(),
        "expected Edit Thread Feature Failed modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(EDIT_FAILED_MODAL));
    assert!(text.contains("EditThreadFeatureFailed"), "{text}");
}

#[then("the edit closes and Thread1 remains in the browser")]
fn edit_closes_thread1_remains(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(EDIT_FAILED_MODAL).is_none());
    assert!(super::super::diagnostics::window_geometry(THREAD_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Thread1"));
}

#[given("the user edits Thread1 to disable full depth, sets depth to 20 mm and accepts")]
fn edit_thread1_disable_full_depth_20mm(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    set_checkbox(world, "Full Depth", false);
    replace_thread_field(world, "Depth", "20 mm");
    accept_thread_and_wait(world);
}

#[when("the user enters zero for Depth")]
#[when("the user replaces Depth with zero")]
fn enter_zero_for_depth(world: &mut ApplicationWorld) {
    replace_thread_field(world, "Depth", "0");
}

#[then("the depth field shows an error indicator and OK is disabled")]
fn depth_error_indicator_ok_disabled(world: &mut ApplicationWorld) {
    let text = world.window_text(THREAD_WINDOW);
    assert!(
        text.contains('!'),
        "expected error indicator in Thread window, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"));
}

#[when("the user replaces zero with 20 mm")]
fn replace_zero_with_20mm(world: &mut ApplicationWorld) {
    replace_thread_field(world, "Depth", "20 mm");
}

#[then("the error indicator clears and OK becomes available")]
fn error_clears_and_ok_available(world: &mut ApplicationWorld) {
    let text = world.window_text(THREAD_WINDOW);
    assert!(!text.contains('!'), "expected error to clear, saw {text}");
    assert!(world.run_is_enabled("OK"));
}

#[given("the tapered face has a full-depth measurement of approximately 49.062 mm")]
fn tapered_face_measures_49_062mm(world: &mut ApplicationWorld) {
    let cones = inspect_matching_faces(world, |info| info.conical && !info.internal);
    let depth_mm = cones[0].1.full_depth_cm * 10.0;
    assert!(
        (depth_mm - 49.062).abs() < 0.05,
        "expected tapered face full depth ~49.062 mm, got {depth_mm:.4}"
    );
}

#[given("the user edits Thread1 to a 20 mm partial depth with thread display enabled and accepts")]
fn edit_thread1_20mm_display_enabled(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    set_checkbox(world, "Full Depth", false);
    replace_thread_field(world, "Depth", "20 mm");
    set_checkbox(world, "Display thread in model", true);
    accept_thread_and_wait(world);
}

#[when("the user enters a 99 mm offset and accepts")]
fn enter_99mm_offset_and_accept(world: &mut ApplicationWorld) {
    replace_thread_field(world, "Offset", "99 mm");
    accept_thread_and_wait(world);
}

#[then("Thread1 remains in the browser and no thread is visible on the face")]
fn thread1_remains_no_thread_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Thread1"));
    assert!(
        thread_overlay_triangles(world).is_empty(),
        "expected no thread visible when offset exceeds face length"
    );
}

#[then("the offset is 99 mm and thread display is checked")]
fn offset_99mm_and_display_checked(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.request.offset, "99 mm");
    assert!(preview.request.display_in_model);
}

#[when("the user restores the offset to 5 mm and accepts")]
fn restore_offset_5mm_and_accept(world: &mut ApplicationWorld) {
    replace_thread_field(world, "Offset", "5 mm");
    accept_thread_and_wait(world);
}

#[then("the partial thread display returns")]
fn partial_thread_display_returns(world: &mut ApplicationWorld) {
    assert!(
        !thread_overlay_triangles(world).is_empty(),
        "expected partial thread display to return after restoring 5 mm offset"
    );
}
