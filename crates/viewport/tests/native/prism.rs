// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismMeshOptions, retain_sketch_prism_definition};
use oblikovati_model::{
    feature::{sketch_prism::SketchPrismDefinition, work_geometry::DatumLength},
    sketch::SketchPlane,
};
pub(super) fn verify_meshed_prism(runtime: &ViewportRuntime) {
    let tilted = tilted_frame();
    let mut view = runtime
        .open_window("Oblikovati placed kernel prism presentation", [320, 240])
        .unwrap();
    for plane in [
        SketchPlane::xy(),
        SketchPlane::xz(),
        SketchPlane::yz(),
        tilted,
        tilted.flipped_normal().unwrap(),
    ] {
        verify_placed_frame(&mut view, plane);
    }
    view.close();
}
fn verify_placed_frame(view: &mut Viewport, plane: SketchPlane) {
    let mut document = native_prism_document(plane);
    let definition = native_prism_definition();
    let first = present_document_prism(view, &document, &definition, plane);
    document.parameters.set_expression(1, "6 cm").unwrap();
    sketch_fixture::solve_selected_sketches(&mut document);
    let edited = present_document_prism(view, &document, &definition, plane);
    assert!(edited.scene_generation > first.scene_generation);
    assert!(green_pixels(&edited) > green_pixels(&first));
}
fn present_document_prism(
    view: &mut Viewport,
    document: &oblikovati_model::history::DocumentSnapshot,
    definition: &SketchPrismDefinition,
    plane: SketchPlane,
) -> ViewportCapture {
    let snapshot = retain_sketch_prism_definition(document, definition).unwrap();
    let analytic = snapshot.report().clone();
    view.set_camera(prism_camera(plane)).unwrap();
    view.submit(&prism_scene(&snapshot)).unwrap();
    let capture = complete_surface_frame(view);
    await_presented_pixels(view, &capture);
    verify_hole_pixels(&capture);
    assert_eq!(snapshot.report(), &analytic);
    capture
}
fn green_pixels(capture: &ViewportCapture) -> usize {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| **pixel == [0, 255, 0, 255])
        .count()
}
fn prism_camera(plane: SketchPlane) -> ViewportCamera {
    let center = plane.to_model([2., 1.5]).unwrap();
    let normal = plane.normal();
    ViewportCamera {
        eye: std::array::from_fn(|i| center[i] + normal[i] * 20.),
        target: std::array::from_fn(|i| center[i] + normal[i]),
        up: plane.recipe().y_axis,
        scale: 6.,
        ..ViewportCamera::default()
    }
}
fn verify_hole_pixels(capture: &ViewportCapture) {
    let [width, height] = capture.extent;
    assert_eq!(sample(capture, width / 2, height / 2), [0, 0, 255, 255]);
    assert_eq!(
        sample(capture, width * 3 / 10, height / 2),
        [0, 255, 0, 255]
    );
    assert_eq!(sample(capture, 0, 0), [0, 0, 255, 255]);
}
fn sample(capture: &ViewportCapture, x: u32, y: u32) -> [u8; 4] {
    let offset = ((y * capture.extent[0] + x) * 4) as usize;
    capture.rgba[offset..offset + 4].try_into().unwrap()
}

fn prism_scene(snapshot: &oblikovati_martensite::PrismProbeSnapshot) -> ViewportScene {
    let mesh = snapshot
        .presentation(PrismMeshOptions {
            chord_error: 0.05,
            angular_error: 0.5,
        })
        .unwrap();
    ViewportScene::from_prism(&mesh, [0., 0., 1.], [0., 1., 0., 1.]).unwrap()
}

fn tilted_frame() -> SketchPlane {
    SketchPlane::new(
        [10., -20., 30.],
        [0.6, 0.8, 0.],
        [0., 0., 1.],
        Default::default(),
    )
    .unwrap()
}

fn native_prism_document(plane: SketchPlane) -> oblikovati_model::history::DocumentSnapshot {
    let mut document = sketch_fixture::selected_sketch_document();
    sketch_fixture::add_fixed_sketch_hole(&mut document);
    sketch_fixture::solve_selected_sketches(&mut document);
    document
        .sketches
        .replace_plane(sketch_fixture::SKETCH, plane)
        .unwrap();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    document
}

fn native_prism_definition() -> SketchPrismDefinition {
    let mut selection = sketch_fixture::selected_sketch_boundary();
    selection
        .holes
        .push(sketch_fixture::selected_sketch_line_ring(111));
    SketchPrismDefinition::new(
        sketch_fixture::SKETCH,
        selection,
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
