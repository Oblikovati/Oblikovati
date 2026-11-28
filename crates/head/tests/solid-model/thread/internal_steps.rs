// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Thread internal-bore scenarios (thread-08..thread-10a).
use super::super::super::*;
use super::setup::{
    THREAD_WINDOW, accept_thread_and_wait, click_threadable_face,
    create_internal_9mm_full_depth_thread1, inspect_matching_faces, open_thread_combo,
    open_thread_panel, popup_option_texts, reopen_thread1, select_thread_combo, set_checkbox,
    thread_overlay_triangles,
};
use cucumber::{given, then, when};

#[given("Revolution1 produces a solid with a stepped internal bore")]
fn revolution_has_stepped_internal_bore(world: &mut ApplicationWorld) {
    assert!(world.state().has_threadable_faces);
    let bores = inspect_matching_faces(world, |info| info.internal && !info.conical);
    assert!(bores.len() >= 2, "expected stepped internal bore faces");
}

#[given("the resulting counterbored hole has a 9 mm diameter upper wall 2 mm deep")]
#[given("the cylindrical inside wall has diameter 9 mm and depth 2 mm")]
fn nine_mm_internal_wall_present(world: &mut ApplicationWorld) {
    let walls = inspect_matching_faces(world, |info| {
        info.internal
            && !info.conical
            && (info.diameter_cm * 10.0 - 9.0).abs() < 0.25
            && (info.full_depth_cm * 10.0 - 2.0).abs() < 0.25
    });
    assert!(
        !walls.is_empty(),
        "expected 9 mm diameter, 2 mm deep internal wall"
    );
}

#[when("the user opens Thread and selects that wall")]
#[when("the user opens Thread and selects the inside wall")]
fn open_thread_and_select_9mm_wall(world: &mut ApplicationWorld) {
    open_thread_panel(world);
    click_threadable_face(world, |info| {
        info.internal
            && !info.conical
            && (info.diameter_cm * 10.0 - 9.0).abs() < 0.25
            && (info.full_depth_cm * 10.0 - 2.0).abs() < 0.25
    });
}

#[when("the user selects ANSI Metric M Profile, size 9, designation M9x1.25 and class 6H")]
fn select_ansi_metric_m9(world: &mut ApplicationWorld) {
    select_thread_combo(world, "Thread Type", "ANSI Metric M Profile");
    select_thread_combo(world, "Size", "9");
    select_thread_combo(world, "Designation", "M9x1.25");
    select_thread_combo(world, "Class", "6H");
}

#[then(
    "the panel shows one surface, ANSI Metric M Profile, size 9, designation M9x1.25 and class 6H"
)]
fn panel_shows_one_surface_ansi_m9(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THREAD_WINDOW));
    assert!(
        text.contains("1Surface"),
        "expected 1 Surface pill, saw {text}"
    );
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.request.spec.thread_type, "ANSI Metric M Profile");
    assert_eq!(preview.request.spec.size, "9");
    assert_eq!(preview.request.spec.designation, "M9x1.25");
    assert_eq!(preview.request.spec.class, "6H");
}

#[when("the user selects full depth")]
#[when("the user turns full depth back on")]
fn select_full_depth(world: &mut ApplicationWorld) {
    set_checkbox(world, "Full Depth", true);
}

#[then("the depth is shown as 2.000 mm")]
fn depth_shown_2_000_mm(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THREAD_WINDOW));
    assert!(
        text.contains("2.000mm"),
        "expected 2.000 mm in Thread panel, saw {text}"
    );
}

#[then("Thread1 appears in the browser and the upper wall displays a thread")]
#[then("Thread1 appears in the browser and the inside wall displays a thread")]
fn thread1_appears_and_wall_displays_thread(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Thread1"));
    assert!(
        !thread_overlay_triangles(world).is_empty(),
        "expected internal wall thread display"
    );
}

#[then("the same specification and full depth are shown")]
fn same_spec_and_full_depth_shown(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert_eq!(preview.request.spec.thread_type, "ANSI Metric M Profile");
    assert_eq!(preview.request.spec.size, "9");
    assert_eq!(preview.request.spec.designation, "M9x1.25");
    assert_eq!(preview.request.spec.class, "6H");
    assert!(preview.request.full_depth);
}

#[given(
    "the user creates a full-depth Thread1 on the 9 mm counterbore wall using ANSI Metric M Profile, designation M9x1.25 and class 6H"
)]
#[given(
    "the user creates a full-depth internal Thread1 using ANSI Metric M Profile, size 9, designation M9x1.25 and class 6H"
)]
fn create_full_depth_internal_m9(world: &mut ApplicationWorld) {
    create_internal_9mm_full_depth_thread1(world);
    let count = thread_overlay_triangles(world).len();
    world.observed_rows = Some(count.to_string());
}

#[given(
    "the user creates and reopens a full-depth internal Thread1 using ANSI Metric M Profile, size 9, designation M9x1.25 and class 6H"
)]
fn create_and_reopen_full_depth_internal_m9(world: &mut ApplicationWorld) {
    create_internal_9mm_full_depth_thread1(world);
    reopen_thread1(world);
}

#[when("the user opens the designation list")]
fn open_designation_list(world: &mut ApplicationWorld) {
    open_thread_combo(world, "Designation");
}

#[when("the user opens Thread1 and opens the designation list")]
fn open_thread1_and_designation_list(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    open_thread_combo(world, "Designation");
}

#[then("M9x1.25, M9x1, M9x0.75 and M9x0.5 are offered")]
fn four_m9_designations_offered(world: &mut ApplicationWorld) {
    let options = popup_option_texts(world);
    for expected in ["M9x1.25", "M9x1", "M9x0.75", "M9x0.5"] {
        assert!(
            options.iter().any(|opt| opt == expected),
            "expected designation {expected:?} in {options:?}"
        );
    }
}

#[when("the user selects M9x0.75 and accepts with class 6H")]
fn select_m9x075_and_accept_6h(world: &mut ApplicationWorld) {
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == "M9x0.75")
        .unwrap_or_else(|| panic!("expected M9x0.75 popup option, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
    select_thread_combo(world, "Class", "6H");
    accept_thread_and_wait(world);
}

#[then("the thread display remains visible with more closely spaced bands")]
fn thread_display_more_closely_spaced_bands(world: &mut ApplicationWorld) {
    let after = thread_overlay_triangles(world).len();
    let before: usize = world
        .observed_rows
        .as_deref()
        .and_then(|s| s.parse().ok())
        .unwrap_or(0);
    assert!(
        after > before && after > 0,
        "expected denser M9x0.75 thread bands ({after}) than M9x1.25 ({before})"
    );
}

#[given("Thread1 is open for editing with a full depth of 2 mm")]
fn thread1_open_full_depth_2mm(world: &mut ApplicationWorld) {
    reopen_thread1(world);
    let preview = world
        .state()
        .thread_preview
        .expect("expected thread preview");
    assert!(preview.request.full_depth);
    depth_shown_2_000_mm(world);
}

#[when("the user turns off full depth")]
fn turn_off_full_depth(world: &mut ApplicationWorld) {
    set_checkbox(world, "Full Depth", false);
}

#[then("editable Depth and Offset fields are shown")]
fn editable_depth_and_offset_shown(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THREAD_WINDOW));
    assert!(
        text.contains("Depth") && text.contains("Offset"),
        "expected Depth and Offset fields when Full Depth is off, saw {text}"
    );
}

#[then("Depth shows 2 mm, Offset is hidden, and OK is enabled")]
fn depth_2mm_offset_hidden_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THREAD_WINDOW));
    assert!(
        text.contains("2.000mm") || text.contains("2mm"),
        "expected Depth to show 2 mm, saw {text}"
    );
    assert!(
        !text.contains("Offset"),
        "expected Offset to be hidden when Full Depth is on, saw {text}"
    );
    assert!(world.run_is_enabled("OK"));
}

#[then("the Thread panel closes")]
fn thread_panel_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry(THREAD_WINDOW).is_none(),
        "expected Thread panel to close"
    );
}
