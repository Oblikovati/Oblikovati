// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Shell scenarios (shell-01..shell-07).
pub(super) mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_model::feature::shell::{ShellDirection, ShellType};
use setup::{
    SHELL_WINDOW, accept_shell_and_wait, assert_shell_ready, click_rounded_face,
    click_shell_control, click_side_planar_face, click_top_planar_face, collect_faces,
    open_shell_ribbon, pan_clear_of_panel, reopen_shell, replace_override_thickness,
    replace_shell_field, solid_volume_and_faces,
};

#[given("the solid block has a top face")]
fn solid_block_has_top_face(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let faces = collect_faces(world);
    assert_eq!(faces.len(), 6, "expected a 6-face rectangular block");
}

#[when("the user opens Shell, selects the top face to remove, and accepts the 1 mm preview")]
fn open_shell_select_top_accept_1mm(world: &mut ApplicationWorld) {
    open_shell_ribbon(world);
    click_top_planar_face(world);
    assert_shell_ready(world);
    let preview = world.state().shell_v2_preview.unwrap();
    assert_eq!(preview.request.thickness, "1 mm");
    assert_eq!(preview.request.openings.len(), 1);
    accept_shell_and_wait(world);
}

#[then("the MCAD Software creates Shell1 and shows an open cavity")]
fn creates_shell1_with_open_cavity(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Shell1"),
        "expected Shell1 in browser, saw {:?}",
        browser.features
    );
    let (vol, faces) = solid_volume_and_faces(world);
    assert!(faces > 6, "expected cavity faces (>6), got {faces}");
    assert!(
        vol > 0.0 && vol < 20.0,
        "expected hollowed solid volume, got {vol}"
    );
}

#[when("the user reopens Shell1 and changes Thickness to 2 mm")]
fn reopen_shell1_change_thickness_2mm(world: &mut ApplicationWorld) {
    reopen_shell(world, "Shell1");
    assert_shell_ready(world);
    replace_shell_field(world, "Thickness", "2 mm");
    assert_shell_ready(world);
}

#[then("the preview shows thicker walls and a smaller opening")]
fn preview_shows_thicker_walls_and_smaller_opening(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell v2 preview");
    assert_eq!(preview.request.thickness, "2 mm");
    assert!(
        !world.app().window.scene_triangles.is_empty(),
        "expected non-empty preview geometry"
    );
}

#[then("accepting the edit updates the shell")]
fn accepting_edit_updates_shell(world: &mut ApplicationWorld) {
    let (vol_before, _) = solid_volume_and_faces(world);
    accept_shell_and_wait(world);
    let (vol_after, _) = solid_volume_and_faces(world);
    assert!(
        vol_after > vol_before + 1e-4,
        "expected thicker 2 mm shell volume ({vol_after}) > 1 mm shell volume ({vol_before})"
    );
}

#[given("the user creates Shell1 by removing the top face with 2 mm thickness")]
fn create_shell1_top_face_2mm(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    open_shell_ribbon(world);
    replace_shell_field(world, "Thickness", "2 mm");
    click_top_planar_face(world);
    assert_shell_ready(world);
    accept_shell_and_wait(world);
}

#[given("Shell1 is open for editing")]
fn shell1_open_for_editing(world: &mut ApplicationWorld) {
    reopen_shell(world, "Shell1");
    assert_shell_ready(world);
}

#[when(
    "the user selects the third direction, adds an override set, selects a side face, and enters 1 mm"
)]
fn select_third_dir_add_override_1mm(world: &mut ApplicationWorld) {
    click_shell_control(world, "Both");
    click_shell_control(world, "+");
    click_side_planar_face(world);
    replace_override_thickness(world, "1 mm");
    assert_shell_ready(world);
}

#[when("the user accepts and reopens the feature")]
fn accept_and_reopen_shell_feature(world: &mut ApplicationWorld) {
    accept_shell_and_wait(world);
    reopen_shell(world, "Shell1");
    assert_shell_ready(world);
}

#[then("the third direction is selected and the override set lists 1 Face at 1 mm")]
fn third_dir_and_override_retained(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview after reopen");
    assert_eq!(preview.request.direction, ShellDirection::Both);
    assert_eq!(preview.request.overrides.len(), 1);
    assert_eq!(preview.request.overrides[0].1, "1 mm");
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains("1Face") && text.contains("1mm"),
        "expected override 1 Face at 1 mm in Shell panel, saw {text}"
    );
}

#[then("the base thickness remains 2 mm")]
fn base_thickness_remains_2mm(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview");
    assert_eq!(preview.request.thickness, "2 mm");
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains("2mm"),
        "expected 2 mm in Shell panel, saw {text}"
    );
}

#[given("Shell is open with Allow Approximation enabled in its advanced properties")]
fn shell_open_with_allow_approximation(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    open_shell_ribbon(world);
    click_shell_control(world, "AllowApproximation");
    assert!(
        world.checked("Allow Approximation"),
        "expected Allow Approximation checked"
    );
}

#[when("the user opens Type")]
fn open_approx_type_dropdown(world: &mut ApplicationWorld) {
    let row = world.run_in_window(SHELL_WINDOW, "Type").clone();
    let (origin, size) = super::super::diagnostics::window_geometry(SHELL_WINDOW).unwrap();
    let combo =
        super::super::diagnostics::pointer_pixels([origin[0] + size[0] * 0.68, row.center()[1]]);
    world.click(combo);
    world.pump(Vec::new());
}

#[then("Never too thin, Never too thick, and Mean are offered")]
fn approx_type_choices_offered(world: &mut ApplicationWorld) {
    let popup_text: String = world
        .frame
        .runs
        .iter()
        .filter(|run| world.frame.is_popup_row(run))
        .map(|run| super::super::world::squeezed(&run.text))
        .collect::<Vec<_>>()
        .join("|");
    for expected in ["Nevertoothin", "Nevertoothick", "Mean"] {
        assert!(
            popup_text.contains(expected),
            "expected {expected} in Type dropdown popup, saw {popup_text}"
        );
    }
}

#[when("the user chooses Never too thin and selects Specified tolerance")]
fn choose_never_too_thin_and_specified_tolerance(world: &mut ApplicationWorld) {
    let item = world
        .frame
        .runs
        .iter()
        .find(|run| {
            world.frame.is_popup_row(run)
                && super::super::world::squeezed(&run.text) == "Nevertoothin"
        })
        .unwrap_or_else(|| panic!("expected Never too thin popup row, saw {}", world.text()))
        .center();
    world.click(super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
    click_shell_control(world, "Specifiedtolerance");
}

#[then("a percentage field appears with 10.0 percent displayed")]
fn percentage_field_displays_10_percent(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains("10.0%"),
        "expected percentage field with 10.0 % displayed, saw {text}"
    );
}

#[when("the user clears Allow Approximation")]
fn clear_allow_approximation(world: &mut ApplicationWorld) {
    click_shell_control(world, "AllowApproximation");
    assert!(!world.checked("Allow Approximation"));
}

#[then("the approximation type, tolerance, and percentage controls are disabled")]
fn approximation_controls_are_disabled(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Never too thin"));
    assert!(!world.run_is_enabled("Optimized"));
    assert!(!world.run_is_enabled("Specified tolerance"));
    assert!(!world.run_is_enabled("10.0 %"));
}

#[given(
    "the user edits Shell1, selects the third direction, adds a 1 mm side-face override and accepts"
)]
fn edit_shell1_both_override_accept(world: &mut ApplicationWorld) {
    reopen_shell(world, "Shell1");
    select_third_dir_add_override_1mm(world);
    accept_shell_and_wait(world);
}

#[given("Shell1 is reopened with thickness overrides and advanced properties visible")]
fn reopen_shell1_with_overrides_and_advanced_visible(world: &mut ApplicationWorld) {
    reopen_shell(world, "Shell1");
    assert_shell_ready(world);
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains("ThicknessOverrideSets") && text.contains("AdvancedProperties"),
        "expected overrides and advanced properties visible, saw {text}"
    );
}

#[when("the user selects the second Shell Type")]
fn select_second_shell_type(world: &mut ApplicationWorld) {
    let before_tris = world.app().window.scene_triangles.len();
    click_shell_control(world, "Rounded");
    assert_shell_ready(world);
    let after_tris = world.app().window.scene_triangles.len();
    assert_ne!(
        before_tris, after_tris,
        "expected Rounded preview mesh triangle count ({after_tris}) to differ from Sharp ({before_tris})"
    );
}

#[then("the preview changes and the panel shows the basic behavior controls")]
fn preview_changes_and_basic_controls_shown(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview");
    assert_eq!(preview.request.options.shell_type, ShellType::Rounded);
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(text.contains("Behavior"), "{text}");
    assert!(text.contains("ShellType"), "{text}");
    assert!(text.contains("Direction"), "{text}");
    assert!(text.contains("Remove"), "{text}");
    assert!(text.contains("Thickness"), "{text}");
    assert!(
        !text.contains("ThicknessOverrideSets") && !text.contains("AdvancedProperties"),
        "expected overrides and advanced properties hidden for second Shell Type, saw {text}"
    );
}

#[then("the previously accepted Shell1 remains")]
fn previously_accepted_shell1_remains(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(SHELL_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    let seed = browser
        .shell_v2_edits
        .values()
        .next()
        .expect("expected Shell1 edit seed");
    assert_eq!(seed.direction, ShellDirection::Both);
    assert_eq!(seed.options.shell_type, ShellType::Sharp);
    assert_eq!(seed.overrides.len(), 1);
    assert_eq!(seed.overrides[0].thickness, "1 mm");
}

#[when("the user selects the second Shell Type and accepts the preview")]
fn select_second_shell_type_and_accept(world: &mut ApplicationWorld) {
    click_shell_control(world, "Rounded");
    assert_shell_ready(world);
    accept_shell_and_wait(world);
}

#[then("the model updates and reopening Shell1 shows that type selected")]
fn model_updates_and_reopen_shows_second_type(world: &mut ApplicationWorld) {
    reopen_shell(world, "Shell1");
    assert_shell_ready(world);
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview");
    assert_eq!(preview.request.options.shell_type, ShellType::Rounded);
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(!text.contains("ThicknessOverrideSets"), "{text}");
}

#[given("the user edits Shell1, selects the second Shell Type and accepts")]
fn edit_shell1_second_type_and_accept(world: &mut ApplicationWorld) {
    reopen_shell(world, "Shell1");
    click_shell_control(world, "Rounded");
    assert_shell_ready(world);
    accept_shell_and_wait(world);
}

#[given("Shell1 is reopened with the alternate Shell type")]
fn reopen_shell1_with_alternate_type(world: &mut ApplicationWorld) {
    reopen_shell(world, "Shell1");
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview");
    assert_eq!(preview.request.options.shell_type, ShellType::Rounded);
}

#[when("the user replaces its thickness with zero")]
fn replace_shell_thickness_with_zero(world: &mut ApplicationWorld) {
    replace_shell_field(world, "Thickness", "0");
}

#[then("an error indicator appears beside Thickness and OK is disabled")]
fn error_indicator_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains('!'),
        "expected error indicator beside Thickness, saw {text}"
    );
    assert!(
        !world.run_is_enabled("OK"),
        "expected OK button disabled on zero thickness"
    );
}

#[then("the previous shell remains")]
fn previous_shell_remains(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(SHELL_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    let seed = browser
        .shell_v2_edits
        .values()
        .next()
        .expect("expected Shell1 edit seed");
    assert_eq!(seed.thickness, "2 mm");
    assert_eq!(seed.options.shell_type, ShellType::Rounded);
}

#[given("the user opens Shell, sets Thickness to 1 mm and enables Allow Approximation")]
fn open_shell_1mm_allow_approximation(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    open_shell_ribbon(world);
    replace_shell_field(world, "Thickness", "1 mm");
    click_shell_control(world, "AllowApproximation");
    assert!(world.checked("Allow Approximation"));
}

#[given("the first Shell Type and first Direction are selected")]
fn first_shell_type_and_first_direction_selected(world: &mut ApplicationWorld) {
    let preview = world.state().shell_v2_preview;
    if let Some(p) = preview {
        assert_eq!(p.request.options.shell_type, ShellType::Sharp);
        assert_eq!(p.request.direction, ShellDirection::Inside);
    }
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(text.contains("Sharp") && text.contains("Inside"), "{text}");
}

#[when("the upper planar face is selected for removal")]
fn select_upper_planar_face_for_removal(world: &mut ApplicationWorld) {
    click_top_planar_face(world);
}

#[then("removes shows 1 Face")]
fn removes_shows_1_face(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected 1 Face in Shell panel, saw {text}"
    );
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview");
    assert_eq!(preview.request.openings.len(), 1);
}

#[when("the lower rounded face is clicked")]
fn click_lower_rounded_face(world: &mut ApplicationWorld) {
    click_rounded_face(world);
}

#[then("removes shows 3 Faces")]
fn removes_shows_3_faces(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains("3Faces"),
        "expected 3 Faces in Shell panel, saw {text}"
    );
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview");
    assert_eq!(preview.request.openings.len(), 3);
}

#[when("the user accepts and reopens Shell1")]
fn accept_and_reopen_shell1(world: &mut ApplicationWorld) {
    assert_shell_ready(world);
    accept_shell_and_wait(world);
    reopen_shell(world, "Shell1");
}

#[then("removes shows 3 Faces and Thickness shows 1 mm")]
fn removes_3_faces_and_thickness_1mm(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SHELL_WINDOW));
    assert!(
        text.contains("3Faces") && text.contains("1mm"),
        "expected 3 Faces and 1 mm in Shell panel, saw {text}"
    );
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell preview");
    assert_eq!(preview.request.openings.len(), 3);
    assert_eq!(preview.request.thickness, "1 mm");
}

#[then("the Shell property panel closes")]
fn shell_property_panel_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(SHELL_WINDOW).is_none(),
        "expected Shell property panel to close"
    );
}
