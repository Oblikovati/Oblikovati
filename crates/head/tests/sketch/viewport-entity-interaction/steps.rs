// SPDX-License-Identifier: GPL-2.0-only
//! Native selection and clearing of a sketch line (#4033).
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_model::sketch::{SketchPointInput, SketchPrimitiveCreation};

#[given("an editable sketch contains an unselected line and the sketch is ready for input")]
fn unselected_line(world: &mut ApplicationWorld) {
    add_isolated_line(world);
    assert!(world.state().sketch_selection.entities.is_empty());
    assert!(world.state().sketch_at_ready);
    world.last_capture = Some(super::super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}
fn add_isolated_line(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::New([-3., 1.]),
            SketchPointInput::New([3., 1.]),
        ],
        construction: false,
        centerline: false,
    };
    world
        .app()
        .owner
        .document_commands(edit.document)
        .create_sketch_primitive(edit.sketch, &line)
        .unwrap();
    world.pump(Vec::new());
}
fn add_overlap_point(world: &mut ApplicationWorld) -> oblikovati_model::sketch::SketchEntityId {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let point = world
        .app()
        .owner
        .document_commands(edit.document)
        .create_sketch_primitive(
            edit.sketch,
            &SketchPrimitiveCreation::Point { position: [0., 1.] },
        )
        .unwrap()
        .entity;
    world.pump(Vec::new());
    point
}
#[when("the user clicks the line away from its endpoints and constraint glyphs")]
fn click_line(world: &mut ApplicationWorld) {
    let position = world.line_point(0, 0.25);
    native_click(world, position);
}
#[then("the line becomes the selected sketch entity")]
fn line_selected(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = world.edited_geometry().lines()[0].id;
    assert_eq!(
        world.state().sketch_selection.entities,
        [(edit.sketch, line)]
    );
}
#[then("the viewport displays a distinct selected state")]
fn selected_visible(world: &mut ApplicationWorld) {
    use oblikovati_api::types::{TOKEN_SKETCH_GEOMETRY, TOKEN_SKETCH_SELECTED};
    let point = world.line_point(0, 0.75);
    let state = world.state();
    let palette = state.theme.as_ref().expect("active theme").palette();
    let normal = &palette[&TOKEN_SKETCH_GEOMETRY];
    let selected = &palette[&TOKEN_SKETCH_SELECTED];
    let colors = [
        [normal.r, normal.g, normal.b],
        [selected.r, selected.g, selected.b],
    ];
    let after = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let before = world.last_capture.as_ref().unwrap();
    assert!(
        selection_color_changed(before, &after, point, colors),
        "expected theme selection color near {point:?}"
    );
    world.save_capture("sketch-line-selected-4033");
}
fn selection_color_changed(
    before: &oblikovati_viewport::ViewportCapture,
    after: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
    colors: [[f32; 3]; 2],
) -> bool {
    let width = after.extent[0] as i32;
    (-2..=2).any(|y| {
        (-2..=2).any(|x| {
            let offset = (((at[1] + y) * width + at[0] + x) * 4) as usize;
            let first = &before.rgba[offset..offset + 3];
            let second = &after.rgba[offset..offset + 3];
            color_distance(first, colors[0]) < color_distance(first, colors[1])
                && color_distance(second, colors[1]) < color_distance(second, colors[0])
        })
    })
}
fn color_distance(pixel: &[u8], color: [f32; 3]) -> f32 {
    pixel
        .iter()
        .zip(color)
        .map(|(channel, target)| (f32::from(*channel) / 255. - target).powi(2))
        .sum()
}
#[when("the user clicks empty sketch space")]
fn click_empty(world: &mut ApplicationWorld) {
    native_click(world, [1100, 650]);
}
fn native_click(world: &mut ApplicationWorld, position: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::click(position);
    #[cfg(not(target_os = "linux"))]
    world.click(position);
    world.pump(Vec::new());
}
#[then("the line is no longer selected")]
fn selection_cleared(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_selection.entities.is_empty());
}

#[given("the sketch contains a curve and a sketch tool is requesting a curve selection")]
fn curve_and_dimension_tool(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let circle = SketchPrimitiveCreation::Circle {
        center: SketchPointInput::New([0., 0.]),
        radius: 3.,
        construction: false,
    };
    world
        .app()
        .owner
        .document_commands(edit.document)
        .create_sketch_primitive(edit.sketch, &circle)
        .unwrap();
    world.key(oblikovati_viewport::ViewportPhysicalKey::D, false);
    world.pump(Vec::new());
    assert!(world.state().dimension_tool_active);
    assert!(world.state().sketch_selection.entities.is_empty());
    world.record_initial_sketch();
    world.last_capture = Some(super::super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}
#[when("the user hovers over a candidate curve segment")]
fn hover_candidate_curve(world: &mut ApplicationWorld) {
    let position = world.sketch_pixel_of([3., 0.]);
    #[cfg(target_os = "linux")]
    super::super::x11_input::move_pointer(position);
    #[cfg(not(target_os = "linux"))]
    world.pump(vec![super::super::super::pointer_motion(position)]);
    world.pump(Vec::new());
}
#[then("the candidate segment receives pointer highlight feedback before it is selected")]
fn candidate_visible_before_selection(world: &mut ApplicationWorld) {
    use oblikovati_api::types::{TOKEN_SKETCH_CANDIDATE, TOKEN_SKETCH_GEOMETRY};
    let point = world.sketch_pixel_of([0., 3.]);
    let state = world.state();
    let palette = state.theme.as_ref().expect("active theme").palette();
    let normal = &palette[&TOKEN_SKETCH_GEOMETRY];
    let candidate = &palette[&TOKEN_SKETCH_CANDIDATE];
    let colors = [
        [normal.r, normal.g, normal.b],
        [candidate.r, candidate.g, candidate.b],
    ];
    let after = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let before = world.last_capture.as_ref().unwrap();
    assert!(
        selection_color_changed(before, &after, point, colors),
        "expected candidate color near {point:?}"
    );
    assert!(world.state().sketch_selection.entities.is_empty());
    world.save_capture("sketch-curve-candidate-4034");
}

#[cfg(target_os = "linux")]
#[test]
fn candidate_clears_on_pointer_exit_and_tool_cancellation() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    curve_and_dimension_tool(&mut world);
    hover_candidate_curve(&mut world);
    candidate_visible_before_selection(&mut world);
    let sample = world.sketch_pixel_of([0., 3.]);
    let baseline = world.last_capture.clone().unwrap();
    super::super::x11_input::move_pointer([1100, 650]);
    world.pump(Vec::new());
    let away = super::super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(same_neighborhood(&baseline, &away, sample));
    hover_candidate_curve(&mut world);
    super::super::x11_input::move_pointer([35, 45]);
    world.pump(Vec::new());
    let over_panel = super::super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(same_neighborhood(&baseline, &over_panel, sample));
    hover_candidate_curve(&mut world);
    world.escape();
    let cancelled = super::super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(same_neighborhood(&baseline, &cancelled, sample));
    world.close();
}
#[cfg(target_os = "linux")]
fn same_neighborhood(
    before: &oblikovati_viewport::ViewportCapture,
    after: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
) -> bool {
    let width = after.extent[0] as i32;
    (-2..=2).all(|y| {
        (-2..=2).all(|x| {
            let offset = (((at[1] + y) * width + at[0] + x) * 4) as usize;
            before.rgba[offset..offset + 3] == after.rgba[offset..offset + 3]
        })
    })
}

#[test]
#[ignore = "run in release mode on a native desktop to measure candidate query cost"]
fn measure_candidate_query_for_dense_sketch() {
    use crate::HeadSketchWork;
    use std::time::Instant;
    let mut world = dense_candidate_world();
    let app = world.app();
    let mut samples = Vec::with_capacity(120);
    for i in 0..140 {
        let start = Instant::now();
        let _ = app
            .owner
            .sketch_tool_candidate([600 + i % 200, 450], &app.window)
            .unwrap();
        if i >= 20 {
            samples.push(start.elapsed());
        }
    }
    samples.sort();
    eprintln!(
        "candidate query 200 segments: p50={:?}, p95={:?}, p99={:?}, max={:?}",
        samples[60], samples[113], samples[118], samples[119]
    );
    world.close();
}

fn dense_candidate_world() -> ApplicationWorld {
    let mut world = dense_sketch_world();
    world.key(oblikovati_viewport::ViewportPhysicalKey::D, false);
    world
}

fn dense_sketch_world() -> ApplicationWorld {
    use oblikovati_model::sketch::SketchPlacementRecipe;
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let points: Vec<[f64; 2]> = (0..=200)
        .map(|i| [i as f64 * 0.04 - 4., (i % 2) as f64 * 0.1])
        .collect();
    let recipe = SketchPlacementRecipe::line_chain(&points, false).unwrap();
    world
        .app()
        .owner
        .commit_sketch_placement(edit.document, edit.sketch, &recipe, &[])
        .unwrap();
    world
}

#[test]
#[ignore = "run in release mode on a native desktop to measure interactive frames"]
fn measure_candidate_frames_for_dense_sketch() {
    use oblikovati_viewport::{ViewportEvent, ViewportInput};
    use std::time::Instant;
    let mut world = dense_candidate_world();
    let app = world.app();
    let extent = app.window.native.drawable_extent().unwrap();
    eprintln!(
        "candidate frame device: {}, extent: {extent:?}",
        app.window.native.runtime.device_name()
    );
    assert_eq!(extent, [1440, 900]);
    let mut samples = Vec::with_capacity(120);
    for i in 0..140 {
        app.window.scripted = vec![ViewportEvent::Input(ViewportInput::Motion {
            position: [600 + i % 200, 450],
            modifiers: Default::default(),
        })];
        let start = Instant::now();
        let summary = crate::frames::run_frames(
            &mut app.window,
            crate::HeadOptions { frames: Some(1) },
            &mut app.owner,
        )
        .unwrap();
        assert_eq!(summary.completed, 1);
        if i >= 20 {
            samples.push(start.elapsed());
        }
    }
    samples.sort();
    eprintln!(
        "candidate frame 200 segments: p50={:?}, p95={:?}, p99={:?}, max={:?}, over_16_67ms={}",
        samples[60],
        samples[113],
        samples[118],
        samples[119],
        samples
            .iter()
            .filter(|elapsed| elapsed.as_secs_f64() > 1. / 60.)
            .count()
    );
    world.close();
}

#[path = "measure_perf.rs"]
mod measure_perf;

#[path = "overlap.rs"]
mod overlap;
#[cfg(target_os = "linux")]
#[path = "pick_cue.rs"]
mod pick_cue;
#[path = "pointer_move.rs"]
mod pointer_move;
#[path = "pointer_placement.rs"]
mod pointer_placement;

#[cfg(target_os = "linux")]
#[path = "ready_hover.rs"]
mod ready_hover;

#[cfg(target_os = "linux")]
#[path = "ready_hover_perf.rs"]
mod ready_hover_perf;
