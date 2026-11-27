// SPDX-License-Identifier: GPL-2.0-only
//! Customize Tools while a live Shell draft owns the modeling command.
use super::super::super::{
    capture_framebuffer, solid_visibility, workspace_layout::browser_feature,
};
use super::super::*;

#[when("the user opens the Options panel title context menu")]
fn open_options_title_menu(world: &mut ApplicationWorld) {
    let title = diagnostics::panel_caption_run(&world.frame, "Options")
        .expect("expected the Options title in the Tools ribbon");
    world.right_click(diagnostics::pointer_pixels(title.center()));
    world.expect_text("Customize User Commands");
}

#[then("the Shell properties and model preview remain visible behind the dialog")]
#[then("the Shell properties and model preview remain visible")]
fn shell_draft_remains(world: &mut ApplicationWorld) {
    let thickness = world.run_in_window("Shell", "Thickness").center();
    if let Some((position, size)) = diagnostics::window_geometry("Customize") {
        assert!(
            (0..2).any(|axis| {
                thickness[axis] < position[axis] || thickness[axis] >= position[axis] + size[axis]
            }),
            "expected Shell properties visible beside Customize"
        );
    }
    browser_feature::expect_solid_visible(world);
}

#[then("the Part Tools target is empty without a pending marker")]
fn empty_applied_tools_target(world: &mut ApplicationWorld) {
    assert_eq!(transaction_steps::target_label(world), "Part|Tools");
    transaction_steps::target_is_empty(world);
    world.save_capture("shell-with-empty-user-commands");
}

#[then("the customization dialog closes without adding a User Commands panel")]
fn staged_toolbar_is_discarded(world: &mut ApplicationWorld) {
    customization_closed(world);
    no_user_commands(world);
}

#[then("the target list contains 2D Sketches")]
fn staged_sketches_are_visible(world: &mut ApplicationWorld) {
    sketches_in_target(world);
}

#[when("the user selects a face of the block")]
fn select_shell_face(world: &mut ApplicationWorld) {
    world.click_text("Select");
    world.click_text("Faces");
    world.pump(Vec::new());
    let capture = capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    let point = solid_visibility::visible_fixture_point(&capture, &windows)
        .expect("expected an exposed patch of the block");
    world.click(point);
    world.pump(Vec::new());
}

#[then("the Shell Remove field reports 1 Face")]
fn shell_records_one_opening(world: &mut ApplicationWorld) {
    world.run_in_window("Shell", "1openingfacesselected");
}

#[then("the model preview shows the selected face removed")]
fn shell_shows_the_opening(world: &mut ApplicationWorld) {
    assert!(
        world.state().extrude_preview_active,
        "expected a certified Shell preview, window shows {:?}",
        world.window_text("Shell")
    );
    let capture = capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    assert!(solid_visibility::visible_preview_fill(&capture, &windows));
}

#[when("the user cancels Shell")]
fn cancel_shell(world: &mut ApplicationWorld) {
    let cancel = world.run_in_window("Shell", "Cancel").center();
    world.click(diagnostics::pointer_pixels(cancel));
    world.pump(Vec::new());
}

#[then("the Shell properties and preview close")]
#[then("the command properties and preview close")]
fn shell_closes(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Shell").is_none());
    assert!(!world.state().extrude_preview_active);
}

#[then("the original solid block remains visible")]
fn original_solid_remains(world: &mut ApplicationWorld) {
    browser_feature::expect_solid_visible(world);
}
