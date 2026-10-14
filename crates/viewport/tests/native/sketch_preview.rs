// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{SketchPlacementRecipe, SketchPlane};
use oblikovati_viewport::ViewportSketchPreview;
pub(super) fn packet() -> ViewportSketchPreview {
    ViewportSketchPreview {
        plane: SketchPlane::xy(),
        recipe: SketchPlacementRecipe::straight_slot([[-5., 0.], [5., 0.]], 4.).unwrap(),
    }
}
#[test]
fn native_sketch_preview_draws_clears_and_keeps_model_generation() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_window("Slot preview", [320, 240]).unwrap();
    view.submit(&ViewportScene {
        background: [0.; 3],
        triangles: vec![],
    })
    .unwrap();
    let baseline = view.redraw().unwrap();
    let packet = packet();
    assert!(view.set_sketch_preview(Some(&packet)).unwrap());
    assert!(view.capture().is_err());
    let drawn = view.redraw().unwrap();
    assert_eq!(drawn.scene_generation, baseline.scene_generation);
    assert!(drawn.overlay_generation > baseline.overlay_generation);
    assert!(
        drawn
            .rgba
            .as_chunks::<4>()
            .0
            .iter()
            .filter(|p| p[0] > 200 && p[1] > 200 && p[2] > 200)
            .count()
            > 50
    );
    assert!(!view.set_sketch_preview(Some(&packet)).unwrap());
    assert_eq!(view.capture().unwrap(), drawn);
    verify_line_styles(&view, &drawn);
    verify_invalid_and_resize(&mut view, &packet);
    assert!(view.set_sketch_preview(None).unwrap());
    assert!(!view.set_sketch_preview(None).unwrap());
    let cleared = view.redraw().unwrap();
    assert_eq!(
        cleared.rgba,
        [0, 0, 0, 255].repeat(cleared.extent[0] as usize * cleared.extent[1] as usize)
    );
    view.close();
    assert!(view.set_sketch_preview(None).is_err());
    assert!(runtime.validation_errors().is_empty());
}
fn verify_invalid_and_resize(view: &mut Viewport, packet: &ViewportSketchPreview) {
    let mut invalid = packet.clone();
    invalid.recipe.points[0][0] = f64::NAN;
    assert!(view.set_sketch_preview(Some(&invalid)).is_err());
    assert!(!view.set_sketch_preview(Some(packet)).unwrap());
    view.resize([400, 300]).unwrap();
    view.poll_events(64).unwrap();
    assert!(view.set_sketch_preview(Some(packet)).unwrap());
    view.redraw().unwrap();
}

fn verify_line_styles(view: &Viewport, capture: &ViewportCapture) {
    let [left, y] = view.sketch_pixel(SketchPlane::xy(), [-4., 0.]).unwrap();
    let [right, _] = view.sketch_pixel(SketchPlane::xy(), [4., 0.]).unwrap();
    let lit: Vec<_> = (left..right)
        .map(|x| {
            (-1..=1).any(|dy| {
                let offset = ((y + dy) as usize * capture.extent[0] as usize + x as usize) * 4;
                capture.rgba[offset] > 200
            })
        })
        .collect();
    assert!(lit.iter().any(|lit| *lit), "construction line absent");
    assert!(
        lit.iter().any(|lit| !*lit),
        "construction line lacks native dashes"
    );
    let [_, y] = view.sketch_pixel(SketchPlane::xy(), [0., 2.]).unwrap();
    assert!(
        (left..right).all(|x| (-1..=1).any(|dy| capture.rgba
            [((y + dy) as usize * capture.extent[0] as usize + x as usize) * 4]
            > 200)),
        "boundary line has gaps"
    );
}

#[test]
fn preview_resamples_after_zoom_and_never_changes_fit_bounds() {
    use oblikovati_viewport::ViewportNavigation;
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime
        .open_window("Preview fit bounds", [320, 240])
        .unwrap();
    view.submit(&scene()).unwrap();
    let initial = view.camera_state().unwrap();
    view.navigate(ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    let expected = view.camera_state().unwrap();
    view.set_camera(initial).unwrap();
    let mut preview = packet();
    preview.recipe = SketchPlacementRecipe::straight_slot([[-500., 0.], [500., 0.]], 50.).unwrap();
    view.set_sketch_preview(Some(&preview)).unwrap();
    view.navigate(ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    assert_eq!(view.camera_state().unwrap(), expected);
    let before = view.redraw().unwrap();
    view.navigate(ViewportNavigation::Zoom {
        factor: 2.,
        anchor: None,
    })
    .unwrap();
    let after = view.redraw().unwrap();
    assert!(after.overlay_generation > before.overlay_generation);
    assert_eq!(after.scene_generation, before.scene_generation);
    let mut invalid = packet();
    invalid.plane = SketchPlane::new(
        [f64::MAX, 0., 0.],
        [1., 0., 0.],
        [0., 1., 0.],
        Default::default(),
    )
    .unwrap();
    assert!(
        view.set_sketch_preview(Some(&invalid))
            .unwrap_err()
            .contains("f32")
    );
    assert_eq!(view.capture().unwrap(), after);
    view.close();
    assert!(runtime.validation_errors().is_empty());
}
