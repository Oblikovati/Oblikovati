// SPDX-License-Identifier: GPL-2.0-only
//! Selection lock changes the camera pivot using the selected B-rep entity.
use super::super::viewport_pan::camera;
use super::appearance::{choose_mode, open_menu};
use super::*;
use crate::HeadWork;

fn select_visible_face(world: &mut ApplicationWorld) {
    world.click_text("Select");
    world.click_text("Faces");
    world.pump(Vec::new());
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    let point = super::super::solid_visibility::visible_fixture_point(&capture, &windows)
        .expect("expected a visible face patch");
    world.click(point);
    world.pump(Vec::new());
    assert!(world.state().model_face_chosen, "expected a selected face");
}
fn close_to(actual: [f64; 3], expected: [f64; 3]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "expected {expected}, got {actual}"
        );
    }
}
#[test]
fn selection_lock_uses_current_face_and_reports_a_missing_selection() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    select_visible_face(&mut world);
    open_menu(&mut world);
    choose_mode(&mut world, "Lock to Current Selection");
    assert!(world.state().orientation_selection_locked);
    let first = world
        .app()
        .owner
        .orientation_selection_pivot()
        .unwrap()
        .unwrap();
    close_to(first, [2., 1.5, 2.]);
    press_part(&mut world, |parts| parts.step_right);
    close_to(camera(&mut world).target, first);
    select_visible_face(&mut world);
    let second = world
        .app()
        .owner
        .orientation_selection_pivot()
        .unwrap()
        .unwrap();
    assert_ne!(
        first, second,
        "the next face must change the selection pivot"
    );
    press_part(&mut world, |parts| parts.step_up);
    close_to(camera(&mut world).target, second);
    world.click(AWAY);
    world.pump(Vec::new());
    assert!(!world.state().orientation_selection_available);
    let before = camera(&mut world);
    press_part(&mut world, |parts| parts.step_left);
    assert_eq!(camera(&mut world), before);
    assert!(
        world
            .text()
            .contains("expectedacurrentselectedmodelfaceorbody")
    );
    open_menu(&mut world);
    choose_mode(&mut world, "Lock to Current Selection");
    assert!(!world.state().orientation_selection_locked);
    let unlocked = camera(&mut world);
    press_part(&mut world, |parts| parts.step_left);
    assert_eq!(camera(&mut world).target, unlocked.target);
    assert_ne!(camera(&mut world).eye, unlocked.eye);
    world.close();
}

fn assert_block_fits_document_area(world: &mut ApplicationWorld) {
    use crate::HeadWindow;
    let (_, [left, top, right, bottom]) = world.app().window.orientation_fit_area().unwrap();
    for x in [0., 4.] {
        for y in [0., 3.] {
            for z in [0., 2.] {
                let pixel = world.app().window.model_pixel([x, y, z]).unwrap();
                assert!(
                    pixel[0] >= left as i32
                        && pixel[0] < right as i32
                        && pixel[1] >= top as i32
                        && pixel[1] < bottom as i32,
                    "model point {:?} projected to {pixel:?}, area {left},{top}..{right},{bottom}",
                    [x, y, z]
                );
            }
        }
    }
}
#[test]
fn locked_fit_keeps_the_selected_pivot_and_the_block_visible() {
    for perspective in [false, true] {
        let mut world = ApplicationWorld::new();
        open_block(&mut world);
        select_visible_face(&mut world);
        open_menu(&mut world);
        choose_mode(&mut world, "Lock to Current Selection");
        if perspective {
            open_menu(&mut world);
            choose_mode(&mut world, "Perspective");
        }
        open_options(&mut world);
        world.click_text("Fit to view on change");
        confirm(&mut world);
        let pivot = world
            .app()
            .owner
            .orientation_selection_pivot()
            .unwrap()
            .unwrap();
        press_part(&mut world, |parts| parts.step_right);
        close_to(camera(&mut world).target, pivot);
        assert_block_fits_document_area(&mut world);
        let visible = solid_extent(&mut world);
        assert!(
            visible[0] > 40 && visible[1] > 40,
            "fitted solid extent {visible:?}: expected a usable view"
        );
        world.close();
    }
}

#[test]
fn selection_lock_survives_a_document_save_and_application_reopen() {
    use oblikovati_api::{client::Client, wire::OpenDocumentArgs};
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    select_visible_face(&mut world);
    let history = world.state().undo_labels;
    open_menu(&mut world);
    choose_mode(&mut world, "Lock to Current Selection");
    assert_eq!(
        world.state().undo_labels,
        history,
        "preference is not a document edit"
    );
    let document = world.active_id();
    let path = world.save_part_to_file(document, "selection-lock.opd");
    world.reopen_application();
    assert!(
        world.state().orientation_selection_locked,
        "preference must survive restart"
    );
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
    select_visible_face(&mut world);
    press_part(&mut world, |parts| parts.step_right);
    close_to(camera(&mut world).target, [2., 1.5, 2.]);
    world.close();
}

#[test]
fn a_selected_body_uses_its_own_named_bounds() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    super::super::workspace_layout::browser_feature::click_branch_arrow(&mut world, "Solid Bodies");
    world.click_text("Solid1");
    world.pump(Vec::new());
    assert!(world.state().orientation_selection_available);
    open_menu(&mut world);
    choose_mode(&mut world, "Lock to Current Selection");
    press_part(&mut world, |parts| parts.step_right);
    close_to(camera(&mut world).target, [2., 1.5, 1.]);
    world.close();
}
