// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId as Id, SketchPlane},
};
use oblikovati_viewport::{ViewportNavigation, ViewportSketchDisplay};
fn packet() -> ViewportSketchDisplay {
    let mut geometry = Sketch2D::default();
    for (id, point) in [(100, [-5., 0.]), (101, [5., 0.]), (102, [0., 0.])] {
        geometry.add_defining_point(Id(id), point).unwrap();
    }
    geometry.add_line(Id(1), [Id(100), Id(101)]).unwrap();
    geometry.add_circle(Id(2), Id(102), 3.).unwrap();
    geometry.add_point(Id(3), [0., 5.]).unwrap();
    ViewportSketchDisplay {
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        geometry,
    }
}
pub(super) fn open_display(title: &str) -> (ViewportRuntime, Viewport) {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_window(title, [320, 240]).unwrap();
    view.submit(&ViewportScene {
        background: [0.; 3],
        triangles: vec![],
    })
    .unwrap();
    (runtime, view)
}
pub(super) fn close_display(runtime: &ViewportRuntime, view: &mut Viewport) {
    view.close();
    assert!(view.set_sketch_display(&[]).is_err());
    assert!(view.sketch_display_declines().is_err());
    assert!(runtime.validation_errors().is_empty());
}
#[test]
fn committed_geometry_survives_preview_clear_and_reports_local_failures() {
    let (runtime, mut view) = open_display("Committed sketch");
    let packet = packet();
    assert!(
        view.set_sketch_display(std::slice::from_ref(&packet))
            .unwrap()
    );
    let baseline = view.redraw().unwrap();
    verify_standalone_point(&view, &baseline);
    let pixels = baseline.rgba.as_chunks::<4>().0;
    assert!(pixels.iter().filter(|pixel| pixel[0] > 100).count() > 50);
    assert!(view.sketch_display_declines().unwrap().is_empty());
    verify_cache_and_preview(&mut view, &packet, &baseline);
    verify_local_decline(&mut view, packet, &baseline);
    view.set_sketch_display(&[]).unwrap();
    assert_black(&view.redraw().unwrap());
    close_display(&runtime, &mut view);
}
fn verify_cache_and_preview(
    view: &mut Viewport,
    packet: &ViewportSketchDisplay,
    baseline: &ViewportCapture,
) {
    assert!(
        !view
            .set_sketch_display(std::slice::from_ref(packet))
            .unwrap()
    );
    assert_eq!(view.capture().unwrap(), *baseline);
    let preview = super::sketch_preview::packet();
    view.set_sketch_preview(Some(&preview)).unwrap();
    assert_ne!(view.redraw().unwrap().rgba, baseline.rgba);
    view.set_sketch_preview(None).unwrap();
    assert_eq!(view.redraw().unwrap().rgba, baseline.rgba);
}
fn verify_local_decline(
    view: &mut Viewport,
    mut packet: ViewportSketchDisplay,
    baseline: &ViewportCapture,
) {
    packet.geometry.add_line(Id(4), [Id(100); 2]).unwrap();
    view.set_sketch_display(&[packet]).unwrap();
    let declines = view.sketch_display_declines().unwrap();
    assert_eq!(declines.len(), 1);
    assert_eq!(
        (declines[0].sketch, declines[0].entity),
        (DocumentSketchId(1), Id(4))
    );
    assert_eq!(view.redraw().unwrap().rgba, baseline.rgba);
}
#[test]
fn committed_sketches_participate_in_fit_and_resample_after_zoom_resize() {
    let (runtime, mut view) = open_display("Sketch fit");
    view.set_sketch_display(&[with_origin(packet(), [1000., 0., 0.])])
        .unwrap();
    view.navigate(ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    let center = view.sketch_pixel(SketchPlane::xy(), [1000., 0.]).unwrap();
    assert!(
        (0..320).contains(&center[0]) && (0..240).contains(&center[1]),
        "{center:?}"
    );
    verify_preview_excluded_from_sketch_fit(&mut view);
    verify_zoom_resize(&mut view);
    close_display(&runtime, &mut view);
}
fn verify_preview_excluded_from_sketch_fit(view: &mut Viewport) {
    let fitted = view.camera_state().unwrap();
    view.set_sketch_preview(Some(&super::sketch_preview::packet()))
        .unwrap();
    view.navigate(ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    assert_eq!(view.camera_state().unwrap(), fitted);
    view.set_sketch_preview(None).unwrap();
}
fn verify_zoom_resize(view: &mut Viewport) {
    let before = view.redraw().unwrap();
    view.navigate(ViewportNavigation::Zoom {
        factor: 2.,
        anchor: None,
    })
    .unwrap();
    let zoomed = view.redraw().unwrap();
    assert!(zoomed.overlay_generation > before.overlay_generation);
    view.resize([400, 300]).unwrap();
    view.poll_events(64).unwrap();
    let resized = view.redraw().unwrap();
    assert!(resized.overlay_generation > zoomed.overlay_generation);
}
fn verify_standalone_point(view: &Viewport, capture: &ViewportCapture) {
    let [x, y] = view.sketch_pixel(SketchPlane::xy(), [0., 5.]).unwrap();
    let lit = (-5..=5)
        .flat_map(|dy| (-5..=5).map(move |dx| (x + dx, y + dy)))
        .filter(|(x, y)| {
            capture.rgba[(*y as usize * capture.extent[0] as usize + *x as usize) * 4] > 100
        })
        .count();
    assert!(lit > 2, "standalone marker absent at {x},{y}");
}
#[test]
fn unrenderable_positions_are_named_and_do_not_keep_old_geometry() {
    let (runtime, mut view) = open_display("Sketch render decline");
    let packet = packet();
    view.set_sketch_display(std::slice::from_ref(&packet))
        .unwrap();
    view.redraw().unwrap();
    view.set_sketch_display(&[with_origin(packet, [f64::MAX, 0., 0.])])
        .unwrap();
    let declines = view.sketch_display_declines().unwrap();
    assert_eq!(declines.len(), 4);
    assert!(declines.iter().all(|decline| decline.cause.contains("f32")));
    assert_black(&view.redraw().unwrap());
    close_display(&runtime, &mut view);
}
fn with_origin(mut packet: ViewportSketchDisplay, origin: [f64; 3]) -> ViewportSketchDisplay {
    packet.plane =
        SketchPlane::new(origin, [1., 0., 0.], [0., 1., 0.], Default::default()).unwrap();
    packet
}
fn assert_black(capture: &ViewportCapture) {
    assert!(
        capture
            .rgba
            .as_chunks::<4>()
            .0
            .iter()
            .all(|pixel| pixel == &[0, 0, 0, 255])
    );
}
#[test]
fn committed_line_roles_use_native_dash_patterns() {
    let (runtime, mut view) = open_display("Sketch line styles");
    let mut packet = packet();
    let solid = display_lit_count(&mut view, &packet);
    packet
        .geometry
        .set_curve_construction(&[Id(1)], true)
        .unwrap();
    let dashed = display_lit_count(&mut view, &packet);
    packet.geometry.set_line_centerline(Id(1), true).unwrap();
    let axis = display_lit_count(&mut view, &packet);
    assert!(
        dashed > 0 && dashed < solid && axis > 0 && axis < solid,
        "solid {solid}, dash {dashed}, axis {axis}"
    );
    close_display(&runtime, &mut view);
}
fn display_lit_count(view: &mut Viewport, packet: &ViewportSketchDisplay) -> usize {
    view.set_sketch_display(std::slice::from_ref(packet))
        .unwrap();
    let capture = view.redraw().unwrap();
    horizontal_lit_count(view, &capture)
}
fn horizontal_lit_count(view: &Viewport, capture: &ViewportCapture) -> usize {
    let [left, y] = view.sketch_pixel(SketchPlane::xy(), [-2., 0.]).unwrap();
    let [right, _] = view.sketch_pixel(SketchPlane::xy(), [2., 0.]).unwrap();
    (left..right)
        .filter(|x| {
            (-1..=1).any(|dy| {
                capture.rgba[((y + dy) as usize * capture.extent[0] as usize + *x as usize) * 4]
                    > 100
            })
        })
        .count()
}
