// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_display::{close_display, open_display};
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId, SketchPlacementRecipe, SketchPlane},
};
use oblikovati_viewport::{ViewportSketchDisplay, ViewportSketchPreview};
#[test]
fn transient_point_uses_the_same_marker_pixels_as_the_committed_point() {
    let (runtime, mut view) = open_display("Point preview");
    let position = [1., -1.];
    let preview = point_packet(position);
    let baseline = view.redraw().unwrap();
    assert!(view.set_sketch_preview(Some(&preview)).unwrap());
    let drawn = view.redraw().unwrap();
    assert_eq!(drawn.scene_generation, baseline.scene_generation);
    assert!(drawn.overlay_generation > baseline.overlay_generation);
    let mask = marker_pixels(&drawn);
    assert!(mask.iter().filter(|pixel| **pixel).count() > 2);
    verify_point_preview_cache(&mut view, &preview, &drawn);
    assert!(view.set_sketch_preview(None).unwrap());
    assert_eq!(view.redraw().unwrap().rgba, baseline.rgba);
    view.set_sketch_display(&[committed_point(&preview, position)])
        .unwrap();
    assert_eq!(marker_pixels(&view.redraw().unwrap()), mask);
    assert!(view.sketch_display_declines().unwrap().is_empty());
    close_display(&runtime, &mut view);
}
fn point_packet(position: [f64; 2]) -> ViewportSketchPreview {
    ViewportSketchPreview::Recipe {
        recipe: SketchPlacementRecipe::point(position).unwrap(),
        plane: SketchPlane::new(
            [1., 2., 0.],
            [0., 1., 0.],
            [-1., 0., 0.],
            Default::default(),
        )
        .unwrap(),
        solve_options: Default::default(),
    }
}
fn committed_point(preview: &ViewportSketchPreview, position: [f64; 2]) -> ViewportSketchDisplay {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), position).unwrap();
    ViewportSketchDisplay {
        sketch: DocumentSketchId(1),
        plane: preview.plane(),
        geometry,
    }
}
fn marker_pixels(capture: &ViewportCapture) -> Vec<bool> {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .map(|pixel| pixel[0] > 100)
        .collect()
}
fn verify_point_preview_cache(
    view: &mut Viewport,
    preview: &ViewportSketchPreview,
    drawn: &ViewportCapture,
) {
    assert!(!view.set_sketch_preview(Some(preview)).unwrap());
    assert_eq!(view.capture().unwrap(), *drawn);
    let mut invalid = preview.clone();
    super::sketch_preview::recipe_parts(&mut invalid).0.points[0][0] = f64::NAN;
    assert!(view.set_sketch_preview(Some(&invalid)).is_err());
    assert_eq!(view.capture().unwrap(), *drawn);
}
