// SPDX-License-Identifier: GPL-2.0-only
//! Curvature preview, Apply, Cancel and OK through the native settings controls.
use super::super::super::{diagnostics, world::CurvatureObservation};
use super::{
    ApplicationWorld,
    curvature::{current_comb, enable_curvature, native_click, native_click_text, open_curve_menu},
};
use cucumber::{given, then, when};
use oblikovati_gui::GuiCurvatureSettings;

const WINDOW: &str = "Curvature Settings";

#[given("the ellipse has Display Curvature enabled")]
#[given("the ellipse has a sparse curvature comb enabled")]
#[given("the ellipse has curvature display enabled")]
#[then("curvature display stays enabled")]
fn curvature_stays_enabled(world: &mut ApplicationWorld) {
    assert!(current_comb(world).is_some());
}

#[when("the user choose Setup Curvature Display from its context menu")]
#[when("the user open Setup Curvature Display")]
#[when("the user reopen Setup Curvature Display")]
#[when("the user reopen Curvature Settings")]
fn open_settings(world: &mut ApplicationWorld) {
    open_curve_menu(world);
    native_click_text(world, "Setup Curvature Display");
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    let next = settings(world);
    let comb = current_comb(world).expect("expected an active curvature comb");
    world.curvature_initial.get_or_insert(CurvatureObservation {
        sliders: next.preview,
        comb,
    });
}

#[given("the user enables Display Curvature on the ellipse and records the slider positions")]
fn record_initial_settings(world: &mut ApplicationWorld) {
    enable_curvature(world);
    world.remember_edited_sketch(Vec::new());
    open_settings(world);
    cancel(world);
}

#[given(
    "the user enables Display Curvature, records both slider positions, previews an upper-slider change and cancels it"
)]
fn preview_and_cancel_upper(world: &mut ApplicationWorld) {
    record_initial_settings(world);
    open_settings(world);
    advance_upper(world);
    expect_denser(world);
    cancel(world);
}

#[given("the prior upper-slider preview was cancelled")]
fn density_restored(world: &mut ApplicationWorld) {
    dialog_closed(world);
    expect_initial_comb(world);
}

#[given("the upper and lower slider positions are recorded")]
#[given("its original upper and lower thumb positions were recorded")]
fn original_positions_recorded(world: &mut ApplicationWorld) {
    assert!(world.curvature_initial.is_some());
}

#[then("Curvature Settings shows two Minimum-to-Maximum sliders without numeric values")]
fn slider_labels(world: &mut ApplicationWorld) {
    let text = world.window_text(WINDOW);
    assert_eq!(text.matches("Minimum").count(), 2, "window text {text:?}");
    assert_eq!(text.matches("Maximum").count(), 2, "window text {text:?}");
    assert!(
        !text.chars().any(|char| char.is_ascii_digit()),
        "window text {text:?}: expected no readouts"
    );
}

#[then("OK and Apply are initially disabled")]
#[then("OK and Apply are disabled")]
fn buttons_disabled(world: &mut ApplicationWorld) {
    for button in ["Apply", "OK"] {
        assert!(
            world.run(button).alpha < u8::MAX,
            "button {button}: expected disabled text"
        );
        if let Some(before) = world.state().curvature_settings {
            assert_eq!(before.applied, before.preview);
            native_click_text(world, button);
            assert_eq!(settings(world), before);
        }
    }
}

#[when("the user advance the upper slider")]
fn advance_upper(world: &mut ApplicationWorld) {
    click_slider(world, 0, 0.80);
}

#[when("the user advance only the lower slider")]
fn advance_lower(world: &mut ApplicationWorld) {
    click_slider(world, 1, 0.80);
}

#[when("the user move the upper slider back to its recorded original position")]
fn restore_upper(world: &mut ApplicationWorld) {
    let original = world.curvature_initial.unwrap().sliders.density;
    let (low, high, y) = slider_track(world, 0);
    let [mut x, y] =
        diagnostics::pointer_pixels([low + (high - low) * f32::from(original) / 100., y]);
    for _ in 0..16 {
        native_click(world, [x, y]);
        let actual = settings(world).preview.density;
        if actual == original {
            return;
        }
        x += (i32::from(original) - i32::from(actual)).signum();
    }
    panic!(
        "slider density {:?}: expected the original {original}",
        settings(world).preview
    );
}

#[then("the thumb advances and the comb becomes denser")]
fn expect_denser(world: &mut ApplicationWorld) {
    let initial = world.curvature_initial.unwrap();
    let view = settings(world);
    assert!(view.preview.density > initial.sliders.density);
    assert_eq!(view.preview.scale, initial.sliders.scale);
    assert!(current_comb(world).unwrap().intervals > initial.comb.intervals);
}

#[then("OK and Apply become enabled")]
#[then("OK and Apply are enabled")]
fn buttons_enabled(world: &mut ApplicationWorld) {
    for button in ["Apply", "OK"] {
        assert_eq!(
            world.run(button).alpha,
            u8::MAX,
            "button {button}: expected enabled text"
        );
    }
    if let Some(view) = world.state().curvature_settings {
        assert_ne!(view.applied, view.preview);
    }
}

#[when("the user click Cancel")]
#[when("the user click Cancel without making changes")]
fn cancel(world: &mut ApplicationWorld) {
    let curvature_open = world.state().curvature_settings.is_some();
    native_click_text(world, "Cancel");
    if curvature_open {
        dialog_closed(world);
    }
}

#[when("the user click Apply")]
fn apply_settings(world: &mut ApplicationWorld) {
    let candidate = world.state().curvature_settings;
    native_click_text(world, "Apply");
    let Some(candidate) = candidate else {
        return;
    };
    let applied = candidate.preview;
    assert_eq!(settings(world).applied, applied);
    let comb = current_comb(world).expect("expected an applied comb");
    world.curvature_applied = Some(CurvatureObservation {
        sliders: applied,
        comb,
    });
}

#[when("the user choose OK")]
fn accept_settings(world: &mut ApplicationWorld) {
    let curvature_open = world.state().curvature_settings.is_some();
    native_click_text(world, "OK");
    if curvature_open {
        dialog_closed(world);
    }
}

#[then("the dialog remains open with the denser comb visible")]
fn dialog_keeps_density(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    expect_denser(world);
}

#[then("the dialog closes and the denser comb remains")]
fn cancel_keeps_applied_density(world: &mut ApplicationWorld) {
    dialog_closed(world);
    let applied = world.curvature_applied.unwrap();
    assert_eq!(
        current_comb(world).unwrap().intervals,
        applied.comb.intervals
    );
}

#[then("the upper slider remains at its applied position")]
fn reopen_applied_density(world: &mut ApplicationWorld) {
    assert_eq!(
        settings(world).preview.density,
        world.curvature_applied.unwrap().sliders.density
    );
}

#[then("the lower slider remains at its original position")]
fn scale_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        settings(world).preview.scale,
        world.curvature_initial.unwrap().sliders.scale
    );
}

#[then("the upper slider stays at its earlier position")]
fn density_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        settings(world).preview.density,
        world.curvature_initial.unwrap().sliders.density
    );
}

#[then("both slider thumbs are at their original positions")]
#[then("the upper and lower thumbs have their earlier restored positions")]
fn positions_restored(world: &mut ApplicationWorld) {
    assert_eq!(
        settings(world).preview,
        world.curvature_initial.unwrap().sliders
    );
}

#[then("the dialog closes and the earlier sparse comb returns")]
#[then("the dialog closes and the earlier sparse comb is displayed")]
#[then("the earlier comb envelope returns with curvature still enabled")]
fn expect_initial_comb(world: &mut ApplicationWorld) {
    dialog_closed(world);
    let initial = world.curvature_initial.unwrap();
    let comb = current_comb(world).unwrap();
    assert_eq!(comb.intervals, initial.comb.intervals);
    assert_eq!(comb.envelope_ratio, initial.comb.envelope_ratio);
}

#[then("the dialog closes and curvature display remains enabled")]
fn reopened_without_edits(world: &mut ApplicationWorld) {
    dialog_closed(world);
    curvature_stays_enabled(world);
}

#[then("the comb envelope expands without a visible ellipse shape change")]
fn envelope_expands(world: &mut ApplicationWorld) {
    let initial = world.curvature_initial.unwrap();
    assert!(current_comb(world).unwrap().envelope_ratio > initial.comb.envelope_ratio);
    world.expect_edited_sketch_retained();
    super::dimension_lifecycle::assert_rendered_ellipse(world);
}

fn settings(world: &mut ApplicationWorld) -> GuiCurvatureSettings {
    world
        .state()
        .curvature_settings
        .expect("expected native Curvature Settings state")
}

fn dialog_closed(world: &mut ApplicationWorld) {
    assert!(world.state().curvature_settings.is_none());
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}

fn click_slider(world: &mut ApplicationWorld, index: usize, fraction: f32) {
    let (low, high, y) = slider_track(world, index);
    native_click(
        world,
        diagnostics::pointer_pixels([low + (high - low) * fraction, y]),
    );
}

fn slider_track(world: &ApplicationWorld, index: usize) -> (f32, f32, f32) {
    let minima: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Minimum")
        .collect();
    let maxima: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Maximum")
        .collect();
    assert_eq!((minima.len(), maxima.len()), (2, 2));
    (
        minima[index].maximum[0] + 12.,
        maxima[index].minimum[0] - 12.,
        minima[index].center()[1],
    )
}
