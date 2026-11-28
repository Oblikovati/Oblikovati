// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Surface Trim scenarios (surface-trim-01..05, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiSurfaceTrimPreview;
use oblikovati_model::feature::surface_trim::SurfaceTrimSide;
use setup::{
    TRIM_WINDOW, click_positive_y_surface_region, click_trim_control,
    create_zero_offset_vertical_surface, current_surface_area, edit_trim1_feature,
    expect_remove_side, hide_original_solid, highlighted_region_mean_y, open_trim_surface_ribbon,
    pick_browser_xz_origin_plane, trim_control_enabled,
};

#[given("the user creates a zero-offset surface from a vertical block side face crossing XZ Plane")]
fn create_zero_offset_surface_step(world: &mut ApplicationWorld) {
    create_zero_offset_vertical_surface(world);
    let area = current_surface_area(world);
    assert!(
        (area - 5.0).abs() < 1e-3,
        "expected initial vertical surface area ~5.0 cm^2, got {area}"
    );
}

#[given("the user hides the original solid")]
fn hide_original_solid_step(world: &mut ApplicationWorld) {
    hide_original_solid(world);
}

#[given("Trim Surface is open without selections")]
fn trim_surface_open_without_selections(world: &mut ApplicationWorld) {
    open_trim_surface_ribbon(world);
    assert!(
        !trim_control_enabled(world, "OK"),
        "expected OK disabled without selections"
    );
    assert!(
        !trim_control_enabled(world, "Apply"),
        "expected Apply disabled without selections"
    );
}

#[when("the user selects XZ Plane as the cutting tool and clicks a surface region")]
fn select_xz_plane_and_click_surface_region(world: &mut ApplicationWorld) {
    pick_browser_xz_origin_plane(world);
    click_positive_y_surface_region(world);
}

#[then("the region is highlighted and OK and Apply become available")]
fn region_highlighted_and_ok_apply_enabled(world: &mut ApplicationWorld) {
    // Clicking y > 0 with XZ Plane normal (0, -1, 0) selects remove_side = Negative (y > 0 half).
    expect_remove_side(world, SurfaceTrimSide::Negative);
    let mean_y = highlighted_region_mean_y(world)
        .expect("expected translucent red highlight triangles for removal region");
    assert!(
        mean_y > 0.3,
        "expected highlighted removal region on y > 0 side, got mean_y={mean_y}"
    );
    assert!(
        trim_control_enabled(world, "OK"),
        "expected OK enabled after selecting cutting tool and removal region"
    );
    assert!(
        trim_control_enabled(world, "Apply"),
        "expected Apply enabled after selecting cutting tool and removal region"
    );
}

#[given(
    "the user opens Trim Surface, selects XZ Plane as the cutting tool and selects a surface region for removal"
)]
fn open_trim_and_select_xz_and_region(world: &mut ApplicationWorld) {
    open_trim_surface_ribbon(world);
    pick_browser_xz_origin_plane(world);
    click_positive_y_surface_region(world);
    expect_remove_side(world, SurfaceTrimSide::Negative);
}

#[when("the user clicks the control beside Remove")]
fn click_switch_beside_remove(world: &mut ApplicationWorld) {
    click_trim_control(world, "Flip");
}

#[then("the highlighted region selection changes")]
fn highlighted_region_selection_changes(world: &mut ApplicationWorld) {
    expect_remove_side(world, SurfaceTrimSide::Positive);
    let mean_y = highlighted_region_mean_y(world)
        .expect("expected highlighted removal region triangles after flip");
    assert!(
        mean_y < -0.3,
        "expected flipped highlight to move to y < 0 side, got mean_y={mean_y}"
    );
}

#[given(
    "the user clicks the control beside Remove twice, restoring the original highlighted region"
)]
fn click_switch_twice_restoring_original(world: &mut ApplicationWorld) {
    click_trim_control(world, "Flip");
    expect_remove_side(world, SurfaceTrimSide::Positive);
    click_trim_control(world, "Flip");
    expect_remove_side(world, SurfaceTrimSide::Negative);
    let mean_y = highlighted_region_mean_y(world).expect("expected restored highlight triangles");
    assert!(
        mean_y > 0.3,
        "expected restored highlight on y > 0 side, got mean_y={mean_y}"
    );
}

#[then("the dialog closes and the large vertical surface is no longer visible in the canvas")]
fn dialog_closes_and_surface_trimmed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(TRIM_WINDOW).is_none(),
        "expected Trim Surface dialog to close after OK"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Trim1"),
        "expected Trim1 in model browser: {:?}",
        browser.features
    );
    let area = current_surface_area(world);
    assert!(
        (area - 2.5).abs() < 1e-3,
        "expected trimmed surface area ~2.5 cm^2 (reduced from 5.0 cm^2), got {area}"
    );
    // Original full vertical surface spanned y in [-1.25, +1.25]; removed y > 0 half must be gone.
    let max_y = world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|t| t.positions)
        .map(|p| p[1])
        .fold(f32::NEG_INFINITY, f32::max);
    assert!(
        max_y <= 0.02,
        "expected removed y > 0 half to be gone from canvas (max_y <= 0.02), got max_y={max_y}"
    );
}

#[then("Trim Surface remains open with Cutting Tool active and OK and Apply disabled")]
fn trim_remains_open_reset_after_apply(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(TRIM_WINDOW).is_some(),
        "expected Trim Surface dialog to stay open after Apply"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Trim1"),
        "expected Trim1 committed by Apply: {:?}",
        browser.features
    );
    let text = super::super::world::squeezed(&world.window_text(TRIM_WINDOW));
    assert!(
        text.contains("SelectCuttingTool"),
        "expected Cutting Tool active and cleared after Apply, saw {text}"
    );
    assert!(
        !trim_control_enabled(world, "OK"),
        "expected OK disabled after Apply reset"
    );
    assert!(
        !trim_control_enabled(world, "Apply"),
        "expected Apply disabled after Apply reset"
    );
}

#[given("the user creates Trim1 using XZ Plane and a selected removal region")]
fn create_trim1_with_xz_plane(world: &mut ApplicationWorld) {
    open_trim_surface_ribbon(world);
    pick_browser_xz_origin_plane(world);
    click_positive_y_surface_region(world);
    expect_remove_side(world, SurfaceTrimSide::Negative);
    click_trim_control(world, "OK");
    assert!(
        super::super::diagnostics::window_geometry(TRIM_WINDOW).is_none(),
        "expected Trim Surface to close after creating Trim1"
    );
}

#[when("the user edits Trim1, clicks the control beside Remove, and cancels")]
fn edit_trim1_flip_and_cancel(world: &mut ApplicationWorld) {
    edit_trim1_feature(world);
    expect_remove_side(world, SurfaceTrimSide::Negative);
    click_trim_control(world, "Flip");
    expect_remove_side(world, SurfaceTrimSide::Positive);
    click_trim_control(world, "Cancel");
    assert!(
        super::super::diagnostics::window_geometry(TRIM_WINDOW).is_none(),
        "expected Trim Surface to close on Cancel"
    );
}

#[then("reopening Trim1 restores the highlighted removal region and enables OK")]
fn reopening_trim1_restores_highlight_and_ok(world: &mut ApplicationWorld) {
    edit_trim1_feature(world);
    let preview = world
        .state()
        .surface_trim_preview
        .expect("expected Surface Trim preview on reopen");
    assert_eq!(preview.result, GuiSurfaceTrimPreview::Ready);
    assert_eq!(preview.request.remove_side, Some(SurfaceTrimSide::Negative));
    let mean_y = highlighted_region_mean_y(world)
        .expect("expected original removal region highlighted on reopen");
    assert!(
        mean_y > 0.3,
        "expected restored original removal region on y > 0 side, got mean_y={mean_y}"
    );
    assert!(
        trim_control_enabled(world, "OK"),
        "expected OK enabled on reopened Trim1"
    );
}
