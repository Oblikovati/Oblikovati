// SPDX-License-Identifier: GPL-2.0-only
//! Observe hover through actual pointer motion and viewport pixels.
use super::super::{capture_framebuffer, desktop_input};
use super::*;
use crate::HeadWindow;
use oblikovati_viewport::ViewportCapture;
use std::time::{Duration, Instant};

pub(in super::super) struct HoverObservation {
    idle: ViewportCapture,
    resting: ViewportCapture,
    highlighted: Option<ViewportCapture>,
}

const TOP: [f32; 3] = [0., 1., 0.];
const FRONT: [f32; 3] = [0., 0., 1.];
const RIGHT: [f32; 3] = [1., 0., 0.];
const EDGE: [[f32; 3]; 2] = [[0., 1., 0.94], [0., 0.94, 1.]];
const CORNER: [[f32; 3]; 3] = [[0.94, 1., 0.94], [0.94, 0.94, 1.], [1., 0.94, 0.94]];

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is active with the orientation control in an oblique view"
)]
fn oblique_control(world: &mut ApplicationWorld) {
    fixture::orientation_part_oblique(world);
    world.observed_view_camera = Some(world.app().window.native.view_camera().unwrap());
    desktop_input::hover(world, AWAY);
    let idle = settled_pixels(world);
    let (position, layout) = layout();
    // Hover Home to bring the control to full opacity without lighting any cube target.
    let home = std::array::from_fn(|axis| position[axis] + layout.home[axis]);
    desktop_input::hover(world, diagnostics::pointer_pixels(home));
    let resting = settled_pixels(world);
    world.orientation_hover = Some(HoverObservation {
        idle,
        resting,
        highlighted: None,
    });
}

#[given("the Top, Front and Right faces of the control are visible")]
#[given("the Top and Front faces and their shared edge are visible")]
#[given("the Top, Front and Right faces meet at a visible corner")]
fn faces_visible(world: &mut ApplicationWorld) {
    expect_control_shows(world, "TopFrontRight");
}

fn projected_pixel(world: &ApplicationWorld, point: [f32; 3]) -> [i32; 2] {
    let (origin, layout) = layout();
    let camera = world.observed_view_camera.unwrap();
    let axes = crate::view_orientation::view_axes(&camera);
    let projection = |axis: [f32; 3]| point.into_iter().zip(axis).map(|(p, a)| p * a).sum::<f32>();
    diagnostics::pointer_pixels(
        [
            origin[0] + layout.center[0] + layout.half * projection(axes.right),
            origin[1] + layout.center[1] - layout.half * projection(axes.up),
        ]
        .map(f32::round),
    )
}

fn hover(world: &mut ApplicationWorld, point: [f32; 3]) {
    desktop_input::hover(world, projected_pixel(world, point));
}

#[when("the pointer hovers over the central target of the Top face without clicking")]
fn hover_top(world: &mut ApplicationWorld) {
    hover(world, TOP);
}

#[when("the pointer moves to the central target of the Front face")]
fn hover_front(world: &mut ApplicationWorld) {
    hover(world, FRONT);
}

#[when("the pointer hovers over the Top side of the shared Top-Front edge target without clicking")]
fn hover_edge_on_top(world: &mut ApplicationWorld) {
    hover(world, EDGE[0]);
}

#[when("the pointer moves across the shared edge to the Front side of the same target")]
fn hover_edge_on_front(world: &mut ApplicationWorld) {
    hover(world, EDGE[1]);
}

#[when("the pointer hovers over the Top side of that corner target without clicking")]
fn hover_corner_on_top(world: &mut ApplicationWorld) {
    hover(world, CORNER[0]);
}

#[when("the pointer moves across the corner target onto its Front side")]
fn hover_corner_on_front(world: &mut ApplicationWorld) {
    hover(world, CORNER[1]);
}

#[then("the entire central Top face target is highlighted as one region")]
fn top_highlighted(world: &mut ApplicationWorld) {
    expect_samples(world, &face_samples(TOP), true);
}

#[then("the entire central Front face target is highlighted as one region")]
fn front_highlighted(world: &mut ApplicationWorld) {
    expect_samples(world, &face_samples(FRONT), true);
}

#[then("the Front and Right face targets are not highlighted")]
fn other_faces_resting(world: &mut ApplicationWorld) {
    expect_samples(world, &face_samples(FRONT), false);
    expect_samples(world, &face_samples(RIGHT), false);
}

#[then("the Top target returns to its resting appearance")]
fn top_resting(world: &mut ApplicationWorld) {
    expect_samples(world, &face_samples(TOP), false);
}

#[then("both visible sides of that edge target are highlighted together as one continuous segment")]
fn edge_highlighted(world: &mut ApplicationWorld) {
    for x in [-0.55, 0., 0.55] {
        expect_samples(world, &EDGE.map(|[_, y, z]| [x, y, z]), true);
    }
    remember_highlight(world);
}

#[then(
    "all three visible sides of the corner target are highlighted together as one continuous segment"
)]
fn corner_highlighted(world: &mut ApplicationWorld) {
    expect_samples(world, &CORNER, true);
    remember_highlight(world);
}

#[then("the remaining face, edge and corner targets are not highlighted")]
fn other_targets_resting(world: &mut ApplicationWorld) {
    for face in [TOP, FRONT, RIGHT] {
        expect_samples(world, &face_samples(face), false);
    }
    // The Top-Right strip and the far Top-Front-Left corner.
    expect_samples(world, &[[0.94, 1., 0.], [-0.94, 1., 0.94]], false);
}

#[then("the model and camera do not move")]
fn stationary(world: &mut ApplicationWorld) {
    assert_eq!(
        world.app().window.native.view_camera().unwrap(),
        world.observed_view_camera.unwrap()
    );
    world.expect_active(world.previous_document.unwrap());
    let capture = capture_framebuffer(&mut world.app().window.native);
    let observation = world.orientation_hover.as_ref().unwrap();
    assert_eq!(
        capture.scene_generation,
        observation.resting.scene_generation
    );
    assert_eq!(
        capture.camera_generation,
        observation.resting.camera_generation
    );
}

#[then(
    "the same complete Top-Front segment stays highlighted without switching to a face-only highlight"
)]
#[then("the same complete Top-Front-Right corner segment stays highlighted")]
fn same_highlight(world: &mut ApplicationWorld) {
    let current = capture_framebuffer(&mut world.app().window.native);
    let before = world
        .orientation_hover
        .as_ref()
        .unwrap()
        .highlighted
        .as_ref()
        .unwrap();
    assert!(
        same_control_pixels(&current, before),
        "highlight shifted at the seam"
    );
    stationary(world);
}

#[when("the pointer leaves the orientation control")]
fn leave(world: &mut ApplicationWorld) {
    desktop_input::hover(world, AWAY);
}

#[then("the hover highlight clears")]
#[then("the whole edge highlight clears")]
#[then("the whole corner highlight clears")]
fn highlight_clears(world: &mut ApplicationWorld) {
    let current = settled_pixels(world);
    let idle = &world.orientation_hover.as_ref().unwrap().idle;
    assert!(
        same_control_pixels(&current, idle),
        "highlight remained after pointer exit"
    );
    stationary(world);
}

pub(super) fn settled_pixels(world: &mut ApplicationWorld) -> ViewportCapture {
    let until = Instant::now() + Duration::from_millis(120);
    while Instant::now() < until {
        world.pump(Vec::new());
    }
    capture_framebuffer(&mut world.app().window.native)
}

fn face_samples(face: [f32; 3]) -> [[f32; 3]; 4] {
    let tangents: Vec<_> = (0..3).filter(|&axis| face[axis] == 0.).collect();
    [(-0.55, -0.55), (-0.55, 0.55), (0.55, -0.55), (0.55, 0.55)].map(|(a, b)| {
        let mut point = face;
        point[tangents[0]] = a;
        point[tangents[1]] = b;
        point
    })
}

fn remember_highlight(world: &mut ApplicationWorld) {
    let current = capture_framebuffer(&mut world.app().window.native);
    world.orientation_hover.as_mut().unwrap().highlighted = Some(current);
}

fn expect_samples(world: &mut ApplicationWorld, samples: &[[f32; 3]], highlighted: bool) {
    let current = capture_framebuffer(&mut world.app().window.native);
    let resting = &world.orientation_hover.as_ref().unwrap().resting;
    for &sample in samples {
        let [x, y] = projected_pixel(world, sample).map(|value| value as usize);
        let actual = pixel(&current, x, y);
        let before = pixel(resting, x, y);
        assert_eq!(
            actual != before,
            highlighted,
            "sample {sample:?} at {x}, {y}: {before:?} to {actual:?}"
        );
    }
}

fn same_control_pixels(a: &ViewportCapture, b: &ViewportCapture) -> bool {
    let (origin, size) = control_geometry();
    let [left, top] = diagnostics::pointer_pixels(origin).map(|value| value as usize);
    let end = std::array::from_fn(|axis| origin[axis] + size[axis]);
    let [right, bottom] = diagnostics::pointer_pixels(end).map(|value| value as usize);
    (top..bottom).all(|y| (left..right).all(|x| pixel(a, x, y) == pixel(b, x, y)))
}

fn pixel(frame: &ViewportCapture, x: usize, y: usize) -> &[u8] {
    let offset = (y * frame.extent[0] as usize + x) * 4;
    &frame.rgba[offset..offset + 4]
}
