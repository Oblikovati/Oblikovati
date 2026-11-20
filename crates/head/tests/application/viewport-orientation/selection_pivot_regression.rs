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
    let [left, top, right, bottom] = world.app().window.orientation_fit_frame().unwrap().area;
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

fn display_distant_sketch(world: &mut ApplicationWorld) {
    use oblikovati_model::{
        document_sketch::DocumentSketchId,
        sketch::{Sketch2D, SketchEntityId, SketchPlane},
    };
    let mut geometry = Sketch2D::default();
    geometry
        .add_defining_point(SketchEntityId(1), [0., 12.])
        .unwrap();
    geometry
        .add_defining_point(SketchEntityId(2), [0., 16.])
        .unwrap();
    geometry
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    let sketch = oblikovati_viewport::ViewportSketchDisplay {
        working_scale: 1.,
        sketch: DocumentSketchId(99),
        plane: SketchPlane::xy(),
        geometry,
        blocks: Vec::new(),
        line_color: None,
    };
    world
        .app()
        .window
        .native
        .view
        .set_sketch_display(&[sketch])
        .unwrap();
}

#[test]
fn selection_fit_includes_a_distant_visible_sketch_and_the_selected_solid() {
    use crate::HeadWindow;
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    select_visible_face(&mut world);
    open_menu(&mut world);
    choose_mode(&mut world, "Lock to Current Selection");
    open_options(&mut world);
    world.click_text("Fit to view on change");
    confirm(&mut world);
    display_distant_sketch(&mut world);
    let bounds = world.app().window.orientation_fit_frame().unwrap().bounds;
    assert!(
        bounds[1][1] >= 16.,
        "display bounds {bounds:?} must include the sketch"
    );
    press_part(&mut world, |parts| parts.step_right);
    close_to(camera(&mut world).target, [2., 1.5, 2.]);
    assert_block_fits_document_area(&mut world);
    let [left, top, right, bottom] = world.app().window.orientation_fit_frame().unwrap().area;
    for point in [[0., 12., 0.], [0., 16., 0.]] {
        let pixel = world.app().window.model_pixel(point).unwrap();
        assert!(
            pixel[0] >= left as i32
                && pixel[0] < right as i32
                && pixel[1] >= top as i32
                && pixel[1] < bottom as i32,
            "sketch point {point:?} at {pixel:?} must fit"
        );
    }
    world
        .app()
        .window
        .native
        .view
        .set_sketch_display(&[])
        .unwrap();
    world.sketch_baseline = Some(super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
    display_distant_sketch(&mut world);
    let first = world.app().window.model_pixel([0., 12., 0.]).unwrap();
    let second = world.app().window.model_pixel([0., 16., 0.]).unwrap();
    world.expect_segment_drawn(first, second);
    world.close();
}
