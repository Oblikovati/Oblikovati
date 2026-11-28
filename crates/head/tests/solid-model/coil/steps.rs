// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Create Coil editing and validation scenarios (coil-02..coil-07).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiCoilPreview;
use oblikovati_model::feature::sketch_coil::{CoilHandedness, CoilMethod};
use setup::{
    click_coil_button, focus_and_replace_coil_field, prepare_coil1_open_for_edit, reopen_coil1,
    scene_bounds,
};

#[given("Coil1 is open for editing")]
fn coil1_is_open_for_editing(world: &mut ApplicationWorld) {
    prepare_coil1_open_for_edit(world);
}

#[given("Coil1 is open with Pitch and Revolution selected")]
#[given("Coil1 uses Pitch and Revolution")]
fn coil1_open_with_pitch_and_rev(world: &mut ApplicationWorld) {
    prepare_coil1_open_for_edit(world);
    let preview = world.state().coil_preview.unwrap();
    assert_eq!(preview.request.method, CoilMethod::PitchAndRevolution);
}

#[given("Coil1 is open with a profile and axis selected")]
fn coil1_open_with_profile_and_axis(world: &mut ApplicationWorld) {
    prepare_coil1_open_for_edit(world);
    let panel = super::super::world::squeezed(&world.window_text("Coil"));
    assert!(
        panel.contains("1Profile") && panel.contains("1Axis"),
        "{panel}"
    );
}

#[given("Coil1 uses Pitch and Revolution with 15 mm pitch and 2.5 revolutions")]
fn coil1_uses_default_pitch_and_rev(world: &mut ApplicationWorld) {
    prepare_coil1_open_for_edit(world);
    let preview = world.state().coil_preview.unwrap();
    assert_eq!(preview.request.method, CoilMethod::PitchAndRevolution);
    assert_eq!(preview.request.pitch, "15 mm");
    assert_eq!(preview.request.revolutions, "2.5");
}

#[when("I open Method")]
fn open_coil_method_dropdown(world: &mut ApplicationWorld) {
    let label = world.run_in_window("Coil", "Method").clone();
    let (origin, size) = super::super::diagnostics::window_geometry("Coil").unwrap();
    let combo =
        super::super::diagnostics::pointer_pixels([origin[0] + size[0] * 0.72, label.center()[1]]);
    world.click(combo);
    world.pump(Vec::new());
}

#[then("Pitch and Revolution, Revolution and Height, Pitch and Height, and Spiral are offered")]
fn all_four_methods_offered(world: &mut ApplicationWorld) {
    for label in [
        "PitchandRevolution",
        "RevolutionandHeight",
        "PitchandHeight",
        "Spiral",
    ] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|row| world.frame.is_popup_row(row) && row.text == label),
            "expected {label} in Method dropdown, saw {}",
            world.text()
        );
    }
}

fn choose_method(world: &mut ApplicationWorld, popup_text: &str) {
    let item = world
        .frame
        .runs
        .iter()
        .find(|row| world.frame.is_popup_row(row) && row.text == popup_text)
        .map(|row| row.center());
    let center = match item {
        Some(pos) => pos,
        None => {
            open_coil_method_dropdown(world);
            world
                .frame
                .runs
                .iter()
                .find(|row| world.frame.is_popup_row(row) && row.text == popup_text)
                .unwrap_or_else(|| panic!("expected {popup_text} in Method dropdown"))
                .center()
        }
    };
    world.click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

#[when("I select Revolution and Height")]
fn select_rev_and_height(world: &mut ApplicationWorld) {
    choose_method(world, "RevolutionandHeight");
}

#[then("Revolution and Height fields are shown")]
fn rev_and_height_fields_shown(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Coil"));
    assert!(
        panel.contains("Revolution") && panel.contains("Height"),
        "{panel}"
    );
    assert_eq!(
        world.state().coil_preview.unwrap().request.method,
        CoilMethod::RevolutionAndHeight
    );
}

#[when("I select Pitch and Height")]
fn select_pitch_and_height(world: &mut ApplicationWorld) {
    choose_method(world, "PitchandHeight");
}

#[then("Pitch and Height fields are shown")]
fn pitch_and_height_fields_shown(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Coil"));
    assert!(
        panel.contains("Pitch") && panel.contains("Height"),
        "{panel}"
    );
    assert_eq!(
        world.state().coil_preview.unwrap().request.method,
        CoilMethod::PitchAndHeight
    );
}

#[when("I select Spiral")]
fn select_spiral_method(world: &mut ApplicationWorld) {
    choose_method(world, "Spiral");
}

#[then("Pitch and Revolution fields remain and the preview becomes an outward spiral")]
fn spiral_fields_and_preview(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Coil"));
    assert!(
        panel.contains("Pitch") && panel.contains("Revolution"),
        "{panel}"
    );
    let preview = world.state().coil_preview.unwrap();
    assert_eq!(preview.request.method, CoilMethod::Spiral);
    assert_eq!(preview.result, GuiCoilPreview::Ready);
    let (lo, hi) = scene_bounds(world);
    assert!(
        hi[0] > 4.0 || lo[0] < -4.0,
        "expected outward spiral X expansion > 4 cm, got X=[{}, {}]",
        lo[0],
        hi[0]
    );
}

#[when("I accept and reopen Coil1")]
fn accept_and_reopen_coil1(world: &mut ApplicationWorld) {
    click_coil_button(world, "OK");
    assert!(super::super::diagnostics::window_geometry("Coil").is_none());
    reopen_coil1(world);
}

#[then("Spiral remains selected")]
fn spiral_remains_selected(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().coil_preview.unwrap().request.method,
        CoilMethod::Spiral
    );
}

#[when("I select right-hand rotation")]
fn select_right_hand_rotation(world: &mut ApplicationWorld) {
    click_coil_button(world, "Right-hand");
}

#[then("the helix preview changes direction")]
fn helix_preview_changes_direction(world: &mut ApplicationWorld) {
    let preview = world.state().coil_preview.unwrap();
    assert_eq!(preview.request.handedness, CoilHandedness::Right);
    assert_eq!(preview.result, GuiCoilPreview::Ready);
}

#[when("I enable Close Start and Close End")]
fn enable_close_start_and_end(world: &mut ApplicationWorld) {
    click_coil_button(world, "CloseStart");
    click_coil_button(world, "CloseEnd");
}

#[then("each end exposes Flat Angle and Transition Angle fields")]
fn end_angle_fields_exposed(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Coil"));
    for label in [
        "StartFlatAngle",
        "StartTransitionAngle",
        "EndFlatAngle",
        "EndTransitionAngle",
    ] {
        assert!(panel.contains(label), "expected {label} in {panel}");
    }
    let preview = world.state().coil_preview.unwrap();
    assert!(preview.request.close_start && preview.request.close_end);
    assert_eq!(preview.result, GuiCoilPreview::Ready);
}

#[then("right-hand rotation and both end options remain selected")]
fn right_hand_and_ends_retained(world: &mut ApplicationWorld) {
    let preview = world.state().coil_preview.unwrap();
    assert_eq!(preview.request.handedness, CoilHandedness::Right);
    assert!(preview.request.close_start && preview.request.close_end);
}

#[when("I enter zero for Pitch")]
#[when("I replace Pitch with zero")]
fn replace_pitch_with_zero(world: &mut ApplicationWorld) {
    focus_and_replace_coil_field(world, "Pitch", "0");
}

#[then("the field shows an error indicator and OK is disabled")]
fn field_shows_error_and_ok_disabled(world: &mut ApplicationWorld) {
    let preview = world.state().coil_preview.unwrap();
    assert!(matches!(preview.result, GuiCoilPreview::Failed(_)));
    assert!(world.window_text("Coil").contains('!'));
    assert!(!world.run_is_enabled("OK"));
}

#[when("I cancel")]
fn cancel_coil_panel(world: &mut ApplicationWorld) {
    if super::super::diagnostics::window_geometry("Draft").is_some() {
        let center = world.run_in_window("Draft", "Cancel").center();
        world.click(super::super::diagnostics::pointer_pixels(center));
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    click_coil_button(world, "Cancel");
}

#[then("the previous coil remains displayed")]
#[then("the previously accepted coil remains displayed")]
fn previous_coil_remains_displayed(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("Coil").is_none());
    assert_eq!(world.state().browser.unwrap().bodies.len(), 2);
}

#[when("I set Taper to 5 degrees and accept")]
fn set_taper_5_deg_and_accept(world: &mut ApplicationWorld) {
    focus_and_replace_coil_field(world, "Taper", "5 deg");
    assert_eq!(
        world.state().coil_preview.unwrap().result,
        GuiCoilPreview::Ready
    );
    click_coil_button(world, "OK");
    assert!(super::super::diagnostics::window_geometry("Coil").is_none());
}

#[then("reopening Coil1 shows a 5 degree taper")]
fn reopen_shows_5_deg_taper(world: &mut ApplicationWorld) {
    reopen_coil1(world);
    let preview = world.state().coil_preview.unwrap();
    assert!(
        preview.request.taper.contains('5'),
        "{}",
        preview.request.taper
    );
    assert_eq!(preview.result, GuiCoilPreview::Ready);
}

#[when("I reverse the axis direction")]
fn reverse_coil_axis(world: &mut ApplicationWorld) {
    click_coil_button(world, "FlipAxis");
}

#[then("the preview extends to the opposite side of the profile")]
fn preview_extends_opposite_side(world: &mut ApplicationWorld) {
    let preview = world.state().coil_preview.unwrap();
    assert!(preview.request.axis_reversed);
    assert_eq!(preview.result, GuiCoilPreview::Ready);
    let (lo, _) = scene_bounds(world);
    assert!(
        lo[1] < -1.0,
        "expected reversed coil Y_min < -1.0 cm, got {}",
        lo[1]
    );
}

#[then("the reversed preview and 5 degree taper remain")]
fn reversed_and_5_deg_taper_remain(world: &mut ApplicationWorld) {
    let preview = world.state().coil_preview.unwrap();
    assert!(preview.request.axis_reversed);
    assert!(
        preview.request.taper.contains('5'),
        "{}",
        preview.request.taper
    );
    assert_eq!(preview.result, GuiCoilPreview::Ready);
}

#[when("I clear the profile selection")]
fn clear_coil_profile(world: &mut ApplicationWorld) {
    click_coil_button(world, "ClearProfile");
}

#[then("the profile control requests a selection and OK is disabled")]
fn profile_requests_selection_ok_disabled(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Coil"));
    assert!(panel.contains("SelectProfiles"), "{panel}");
    assert_eq!(
        world.state().coil_preview.unwrap().result,
        GuiCoilPreview::MissingProfile
    );
    assert!(!world.run_is_enabled("OK"));
}

#[then("the preview disappears, Pitch shows an error, and OK is disabled")]
#[then("the preview disappears, Revolution shows an error, and OK is disabled")]
fn preview_disappears_error_ok_disabled(world: &mut ApplicationWorld) {
    field_shows_error_and_ok_disabled(world);
    let (lo, hi) = scene_bounds(world);
    assert!(
        hi[1] < 3.5 && lo[1] > -0.5,
        "expected coil preview hidden (base block only), saw Y=[{}, {}]",
        lo[1],
        hi[1]
    );
}

#[when("I cancel and reopen Coil1")]
fn cancel_and_reopen_coil1(world: &mut ApplicationWorld) {
    cancel_coil_panel(world);
    reopen_coil1(world);
}

#[then("Pitch shows 15 mm")]
fn pitch_shows_15_mm(world: &mut ApplicationWorld) {
    assert_eq!(world.state().coil_preview.unwrap().request.pitch, "15 mm");
}

#[when("I replace Revolution with zero")]
fn replace_revolution_with_zero(world: &mut ApplicationWorld) {
    focus_and_replace_coil_field(world, "Revolution", "0");
}

#[then("the Coil panel closes")]
fn coil_panel_closes(_world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("Coil").is_none());
}
