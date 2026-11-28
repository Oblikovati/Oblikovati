// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Surface Extend scenarios (extend-surface-01..06, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::{GuiExtendSurfaceExtentMode, GuiExtendSurfacePreview};
use oblikovati_model::feature::extend_surface::ExtendSurfaceMethod;
use setup::{
    EXTEND_WINDOW, assert_distance_10mm_and_stretch, click_extend_control,
    click_extend_control_with_index, create_1mm_offset_surface, double_click_extend1_in_browser,
    open_extend_surface_ribbon, ready_preview_area, replace_distance_field,
    select_right_boundary_edge, select_upper_boundary_edge, set_edge_chain, surface_y_span_at_z21,
};

#[given("the user creates a 1 mm offset surface from the top planar face")]
fn user_creates_1mm_offset_surface(world: &mut ApplicationWorld) {
    create_1mm_offset_surface(world);
}

#[given("Extend Surface is open and Edge Chain is cleared")]
fn extend_surface_open_edge_chain_cleared(world: &mut ApplicationWorld) {
    open_extend_surface_ribbon(world);
    set_edge_chain(world, false);
}

#[when("I select the upper boundary edge of the surface")]
fn select_upper_surface_boundary_edge(world: &mut ApplicationWorld) {
    select_upper_boundary_edge(world);
}

#[then("a 5 mm extension preview appears and OK becomes available")]
fn five_mm_extension_preview_and_ok_enabled(world: &mut ApplicationWorld) {
    let area = ready_preview_area(world);
    assert!(
        (area - 1200.0).abs() < 1.0,
        "expected 5 mm extension of 40x25 mm surface -> 1200 mm², got {area}"
    );
    let span = surface_y_span_at_z21(world);
    assert!(
        (span - 3.0).abs() < 0.05,
        "expected 5 mm extension y-span = 3.0 cm, got {span}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[given(
    "Extend Surface is open with one offset-surface boundary edge selected and Distance extent active"
)]
fn extend_surface_open_one_edge_distance_active(world: &mut ApplicationWorld) {
    open_extend_surface_ribbon(world);
    set_edge_chain(world, false);
    select_upper_boundary_edge(world);
    let preview = world
        .state()
        .extend_surface_preview
        .expect("expected 5 mm preview");
    assert_eq!(
        preview.request.extent_mode,
        GuiExtendSurfaceExtentMode::Distance
    );
    assert_eq!(preview.request.distance, Some(0.5));
    let area = ready_preview_area(world);
    assert!((area - 1200.0).abs() < 1.0);
}

#[when("I choose To")]
fn choose_to_extent(world: &mut ApplicationWorld) {
    click_extend_control(world, "To");
}

#[then(
    "a termination selector replaces the distance field and OK is disabled until a target is supplied"
)]
fn termination_selector_replaces_distance_and_ok_disabled(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .extend_surface_preview
        .expect("expected preview state");
    assert_eq!(preview.request.extent_mode, GuiExtendSurfaceExtentMode::To);
    assert!(preview.request.termination.is_none());
    assert_eq!(preview.result, GuiExtendSurfacePreview::Incomplete);
    let text = super::super::world::squeezed(&world.window_text(EXTEND_WINDOW));
    assert!(
        text.contains("Termination") && text.contains("SelectTermination"),
        "expected Termination selector in panel, saw {text}"
    );
    assert!(
        !text.contains("5mm"),
        "expected distance field replaced when To is active, saw {text}"
    );
    assert!(
        !world.run_is_enabled("OK"),
        "expected OK disabled until termination target is supplied"
    );
}

#[when("I choose Stretch and change the distance from 5 to 10 mm")]
fn choose_stretch_and_change_distance_to_10mm(world: &mut ApplicationWorld) {
    let before_area = ready_preview_area(world);
    let before_span = surface_y_span_at_z21(world);
    assert!(
        (before_area - 1200.0).abs() < 1.0 && (before_span - 3.0).abs() < 0.05,
        "expected 5 mm preview before enlarging: area={before_area}, span={before_span}"
    );
    click_extend_control(world, "Stretch");
    replace_distance_field(world, "10 mm");
}

#[then("the extension preview becomes larger")]
fn extension_preview_becomes_larger(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .extend_surface_preview
        .expect("expected 10 mm Stretch preview");
    assert_eq!(preview.request.method, ExtendSurfaceMethod::Stretch);
    assert_eq!(preview.request.distance, Some(1.0));
    let area = ready_preview_area(world);
    assert!(
        (area - 1400.0).abs() < 1.0,
        "expected 10 mm extension area = 1400 mm² (> 1200 mm²), got {area}"
    );
    let span = surface_y_span_at_z21(world);
    assert!(
        (span - 3.5).abs() < 0.05,
        "expected 10 mm extension y-span = 3.5 cm (> 3.0 cm), got {span}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[given(
    "Extend Surface is open with one offset-surface boundary edge selected and a 10 mm Stretch preview"
)]
fn extend_surface_open_with_10mm_stretch_preview(world: &mut ApplicationWorld) {
    extend_surface_open_one_edge_distance_active(world);
    choose_stretch_and_change_distance_to_10mm(world);
    extension_preview_becomes_larger(world);
}

#[then("the dialog closes and the enlarged surface remains visible")]
fn dialog_closes_and_enlarged_surface_visible(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(EXTEND_WINDOW).is_none(),
        "expected Extend Surface dialog to close after OK"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Extend1"),
        "expected Extend1 in browser, saw {:?}",
        browser.features
    );
    assert_eq!(
        browser.surface_bodies.len(),
        1,
        "expected enlarged Srf1 to replace previous surface body in place"
    );
    let span = surface_y_span_at_z21(world);
    assert!(
        (span - 3.5).abs() < 0.05,
        "expected committed enlarged surface y-span = 3.5 cm, got {span}"
    );
}

#[given(
    "the user creates Extend1 from one offset-surface boundary edge with 10 mm Distance and Stretch"
)]
fn user_creates_extend1_10mm_stretch(world: &mut ApplicationWorld) {
    extend_surface_open_with_10mm_stretch_preview(world);
    click_extend_control(world, "OK");
    world.pump(Vec::new());
    dialog_closes_and_enlarged_surface_visible(world);
}

#[when("I edit it, select Extend, and cancel")]
fn edit_extend1_select_extend_and_cancel(world: &mut ApplicationWorld) {
    double_click_extend1_in_browser(world);
    assert_distance_10mm_and_stretch(world);
    // Click the "Extend" method button inside the More section (not the header title).
    click_extend_control_with_index(world, "Extend", true);
    let preview = world
        .state()
        .extend_surface_preview
        .expect("expected preview after switching to Extend");
    assert_eq!(preview.request.method, ExtendSurfaceMethod::Extend);
    click_extend_control(world, "Cancel");
    world.pump(Vec::new());
    assert!(
        super::super::diagnostics::window_geometry(EXTEND_WINDOW).is_none(),
        "expected Extend Surface dialog to close after Cancel"
    );
}

#[then("reopening Extend1 shows 10 mm and Stretch selected")]
fn reopening_extend1_shows_10mm_and_stretch(world: &mut ApplicationWorld) {
    double_click_extend1_in_browser(world);
    assert_distance_10mm_and_stretch(world);
    let text = super::super::world::squeezed(&world.window_text(EXTEND_WINDOW));
    assert!(
        text.contains("10mm") && text.contains("Stretch"),
        "expected 10 mm and Stretch selected in reopened Extend1, saw {text}"
    );
}

#[given("Extend Surface is open with Edge Chain checked")]
fn extend_surface_open_edge_chain_checked(world: &mut ApplicationWorld) {
    open_extend_surface_ribbon(world);
    set_edge_chain(world, true);
}

#[when("I select the right edge of the rectangular boundary")]
fn select_right_rectangular_boundary_edge(world: &mut ApplicationWorld) {
    select_right_boundary_edge(world);
}

#[then("the visible rectangular perimeter is highlighted and OK becomes available")]
fn rectangular_perimeter_highlighted_and_ok_enabled(world: &mut ApplicationWorld) {
    let state = world.state();
    let preview = state
        .extend_surface_preview
        .expect("expected chained perimeter preview");
    assert!(preview.request.edge_chain);
    assert_eq!(
        preview.request.edges.len(),
        4,
        "expected all 4 rectangular boundary edges chained, saw {}",
        preview.request.edges.len()
    );
    assert_eq!(
        state.selected_edge_count, 4,
        "expected all 4 perimeter edges selected for viewport highlighting, saw {}",
        state.selected_edge_count
    );
    let outlines = crate::HeadWork::selected_edge_outlines(&mut world.app().owner);
    assert_eq!(
        outlines.len(),
        4,
        "expected all 4 perimeter edges highlighted in viewport outlines, saw {}",
        outlines.len()
    );
    assert!(
        outlines.iter().all(|group| !group.is_empty()),
        "expected every perimeter edge outline to contain visible segments"
    );
    let area = ready_preview_area(world);
    assert!(
        (area - 1750.0).abs() < 1.0,
        "expected 5 mm perimeter extension area = 1750 mm², got {area}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}
