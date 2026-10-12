// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismMeshOptions, retain_stored_sketch_prism};
use oblikovati_model::{
    feature::{FeatureId, sketch_prism::SketchPrismDefinition, work_geometry::DatumLength},
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
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
    let mut owner = native_prism_owner(plane);
    let first = present_document_prism(view, owner.snapshot(), plane);
    edit_prism_width(&mut owner);
    let edited = present_document_prism(view, owner.snapshot(), plane);
    assert!(edited.scene_generation > first.scene_generation);
    assert!(green_pixels(&edited) > green_pixels(&first));
    owner.undo().unwrap();
    let undone = present_document_prism(view, owner.snapshot(), plane);
    assert!(undone.scene_generation > edited.scene_generation);
    assert_eq!(undone.rgba, first.rgba);
    owner.redo().unwrap();
    let redone = present_document_prism(view, owner.snapshot(), plane);
    assert!(redone.scene_generation > undone.scene_generation);
    assert_eq!(redone.rgba, edited.rgba);
}
fn present_document_prism(
    view: &mut Viewport,
    document: &DocumentSnapshot,
    plane: SketchPlane,
) -> ViewportCapture {
    let snapshot = retain_stored_sketch_prism(document, FeatureId(1)).unwrap();
    let analytic = snapshot.report().clone();
    view.set_camera(prism_camera(plane)).unwrap();
    view.submit(&prism_scene(&snapshot)).unwrap();
    let capture = complete_surface_frame(view);
    await_presented_pixels(view, &capture);
    verify_hole_pixels(&capture);
    assert_eq!(snapshot.report(), &analytic);
    capture
}
fn edit_prism_width(owner: &mut DocumentEditHistory) {
    let token = owner.begin("Width").unwrap();
    owner
        .edit(&token, |document, _| {
            document.parameters.set_expression(1, "6 cm")?;
            sketch_fixture::solve_selected_sketches(document);
            Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    owner.commit(&token, false).unwrap();
}
fn native_prism_owner(plane: SketchPlane) -> DocumentEditHistory {
    let document = native_prism_document(plane);
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    let mut owner = DocumentEditHistory::new(document, ids).unwrap();
    let token = owner.begin("Prism inputs").unwrap();
    let id = owner
        .create_sketch_prism(&token, "Prism", native_prism_definition())
        .unwrap();
    assert_eq!(id, FeatureId(1));
    owner.commit(&token, false).unwrap();
    owner
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
    // The camera fixes vertical span; a horizontal fraction can leave the body on wide windows.
    assert_eq!(
        sample(capture, width / 2, height * 3 / 10),
        [0, 255, 0, 255]
    );
    assert_eq!(sample(capture, 0, 0), [0, 0, 255, 255]);
}
#[test]
fn prism_hole_pixels_remain_valid_when_the_drawable_is_wide() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let plane = SketchPlane::xy();
    let owner = native_prism_owner(plane);
    let snapshot = retain_stored_sketch_prism(owner.snapshot(), FeatureId(1)).unwrap();
    for extent in [[320, 240], [960, 240]] {
        let mut view = runtime.open_offscreen(extent).unwrap();
        view.set_camera(prism_camera(plane)).unwrap();
        view.submit(&prism_scene(&snapshot)).unwrap();
        let capture = view.redraw().unwrap();
        assert_eq!(capture.extent, extent);
        verify_hole_pixels(&capture);
    }
    assert!(runtime.validation_errors().is_empty());
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
