// SPDX-License-Identifier: GPL-2.0-only
//! Test Zoom Selected against the rendered pixel and the face hit by that pixel's view ray.
use super::*;
use crate::{HeadWindow, HeadWork};

type Edges = Vec<[[f64; 3]; 2]>;

fn visible_face_ray(world: &mut ApplicationWorld) -> ([i32; 2], Edges) {
    let point = visible_face_middle(world);
    let line = world.app().window.view_line(point).unwrap();
    let edges = world.app().owner.face_edges_on_line(line).unwrap();
    assert!(
        !edges.is_empty(),
        "rendered pixel {point:?}: expected an exact face hit"
    );
    (point, edges)
}
pub(super) fn assert_unoccluded(point: [i32; 2]) {
    for [left, top, right, bottom] in super::super::diagnostics::window_occlusions() {
        assert!(
            (point[0] as f32) < left
                || point[0] as f32 >= right
                || (point[1] as f32) < top
                || point[1] as f32 >= bottom,
            "click {point:?} covered by window {left},{top}..{right},{bottom}"
        );
    }
}
pub(super) fn assert_face_is_framed(world: &mut ApplicationWorld, edges: &Edges) {
    let [left, top, right, bottom] = world
        .app()
        .window
        .orientation_fit_frame()
        .unwrap()
        .area
        .map(|x| x as i32);
    let (low, high) = projected_bounds(world, edges);
    assert!(
        low[0] >= left && low[1] >= top && high[0] < right && high[1] < bottom,
        "face {low:?}..{high:?}, document area {:?}",
        [left, top, right, bottom]
    );
    assert!(
        high[0] - low[0] > (right - left) / 2 || high[1] - low[1] > (bottom - top) / 2,
        "face {low:?}..{high:?}: expected it to fill the document area"
    );
}
fn projected_bounds(world: &mut ApplicationWorld, edges: &Edges) -> ([i32; 2], [i32; 2]) {
    let mut low = [i32::MAX; 2];
    let mut high = [i32::MIN; 2];
    for point in edges.iter().flatten() {
        let pixel = world.app().window.model_pixel(*point).unwrap();
        for axis in 0..2 {
            low[axis] = low[axis].min(pixel[axis]);
            high[axis] = high[axis].max(pixel[axis]);
        }
    }
    (low, high)
}
fn prepare_small_block(world: &mut ApplicationWorld) {
    open_block(world);
    shrink_the_block(world);
}
#[test]
fn preselected_face_is_the_rendered_face_and_zoom_frames_its_boundary() {
    let mut world = ApplicationWorld::new();
    prepare_small_block(&mut world);
    world.click_text("Select");
    world.click_text("Faces");
    let (point, expected) = visible_face_ray(&mut world);
    click_pump(&mut world, point);
    assert!(world.state().model_face_chosen);
    assert_eq!(world.app().owner.selected_browser_outline(), expected);
    choose_from_zoom_menu(&mut world, "Zoom Selected");
    assert!(world.state().model_face_chosen);
    assert_face_is_framed(&mut world, &expected);
    world.close();
}
#[test]
fn face_picked_after_start_is_framed_without_leaving_it_selected() {
    let mut world = ApplicationWorld::new();
    prepare_small_block(&mut world);
    let (point, expected) = visible_face_ray(&mut world);
    assert!(!world.state().model_face_chosen);
    choose_from_zoom_menu(&mut world, "Zoom Selected");
    world.expect_status("select an entity to zoom");
    click_pump(&mut world, point);
    assert!(!world.state().model_face_chosen);
    assert_face_is_framed(&mut world, &expected);
    world.expect_ready();
    world.close();
}

#[test]
fn zoom_selected_frames_the_picked_face_in_perspective() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    world
        .app()
        .window
        .set_view_projection(oblikovati_gui::GuiViewProjection::Perspective);
    world.pump(Vec::new());
    shrink_the_block(&mut world);
    let (point, expected) = visible_face_ray(&mut world);
    let previous = camera(&mut world);
    choose_from_zoom_menu(&mut world, "Zoom Selected");
    click_pump(&mut world, point);
    assert!(camera(&mut world).perspective);
    assert_face_is_framed(&mut world, &expected);
    assert!(world.app().window.previous_view().unwrap());
    assert_eq!(camera(&mut world), previous);
    world.close();
}
