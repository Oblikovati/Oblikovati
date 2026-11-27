// SPDX-License-Identifier: GPL-2.0-only
//! Check the camera and the visible model during the native orientation transition.
use super::*;
use crate::HeadWindow;
use oblikovati_viewport::ViewportCamera;
use std::time::{Duration, Instant};

#[given("animated view transitions are enabled")]
fn enable_transitions(world: &mut ApplicationWorld) {
    open_options(world);
    click_row(world, "Use animated transitions");
    assert!(world.checked("Use animated transitions"));
    confirm(world);
    let start = world.app().window.native.view_camera().unwrap();
    world.sketch_baseline = Some(world.app().window.native.capture_window().unwrap());
    world.app().window.orientation_trace = Some(super::super::super::OrientationTrace {
        initial_top_direction: toward_top(start),
        ..Default::default()
    });
}

#[then("the camera moves through visible intermediate orientations toward the Top view")]
fn camera_intermediates(world: &mut ApplicationWorld) {
    let timeout = Instant::now() + Duration::from_secs(3);
    while Instant::now() < timeout {
        world.pump(Vec::new());
        if toward_top(world.app().window.native.view_camera().unwrap()) > 0.999999 {
            break;
        }
    }
    let trace = world.app().window.orientation_trace.as_ref().unwrap();
    let first = trace.initial_top_direction;
    let last = toward_top(trace.presentations.last().unwrap().camera);
    assert!(last > 0.999999, "camera stopped at {last:?}: expected Top");
    let observed: Vec<_> = trace
        .presentations
        .iter()
        .map(|frame| (toward_top(frame.camera), frame.bounds.is_some()))
        .collect();
    assert!(
        trace.presentations.iter().any(|frame| {
            let toward = toward_top(frame.camera);
            toward > first + 0.001 && toward < last - 0.001 && frame.bounds.is_some()
        }),
        "expected a visible intermediate frame between {first:?} and {last:?}, got {observed:?}"
    );
}

fn toward_top(camera: ViewportCamera) -> f32 {
    crate::view_orientation::view_axes(&camera).toward[1]
}

#[then("the model view and orientation control update consistently during the transition")]
fn control_tracks_camera(world: &mut ApplicationWorld) {
    let trace = world.app().window.orientation_trace.as_ref().unwrap();
    let expected = trace.presentations.last().unwrap().bounds;
    assert!(expected.is_some());
    assert!(
        trace
            .presentations
            .iter()
            .any(|frame| frame.bounds.is_some() && frame.bounds != expected),
        "expected the visibly shaded shape to change"
    );
    for frame in &trace.presentations {
        assert_eq!(
            frame.axes,
            Some(crate::view_orientation::view_axes(&frame.camera))
        );
    }
}

#[then("the camera settles with the block shown from the Top")]
fn animation_settles(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Top");
    top_face_on(world);
}

#[then("the active document and model geometry remain unchanged")]
fn original_solid_unchanged(world: &mut ApplicationWorld) {
    expect_same_part(world);
    let baseline = world.sketch_baseline.take().unwrap();
    let capture = world.app().window.native.capture_window().unwrap();
    assert_eq!(
        capture.scene_generation, baseline.scene_generation,
        "orientation is a view operation"
    );
    let trace = world.app().window.orientation_trace.take().unwrap();
    let path = super::super::world::reports_folder().join("orientation-intermediate.png");
    super::super::world::write_png(
        &path,
        &trace
            .intermediate
            .expect("expected a visible intermediate view"),
    )
    .unwrap();
}
