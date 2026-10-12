// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_vulkan::{GlyphLayout, RasterImage, TextLayoutSource};
use rustycad_tkservice::{font::FontRect, graphic3d::*};
pub(super) struct FixedOverlayFont;
impl TextLayoutSource for FixedOverlayFont {
    fn layout(
        &self,
        text: &Graphic3dText,
        _: &dyn Graphic3dAspectsSource,
        _: &Graphic3dRenderingParams,
    ) -> Result<Option<GlyphLayout>, RenderError> {
        Ok(Some(fixed_overlay_glyph(text.height() as u32)?))
    }
}
fn fixed_overlay_glyph(point_size: u32) -> Result<GlyphLayout, RenderError> {
    Ok(GlyphLayout {
        positions: vec![[0., 0.], [4., 0.], [4., 6.], [0., 6.]],
        uv: vec![[0., 1.], [1., 1.], [1., 0.], [0., 0.]],
        indices: vec![0, 1, 2, 0, 2, 3],
        atlas: RasterImage::new([1, 1], vec![[1.; 4]])?,
        formatter_bounds: FontRect {
            left: 0.,
            right: 4.,
            top: 6.,
            bottom: 0.,
        },
        point_size,
    })
}
#[test]
fn status_changes_invalidate_only_overlay_generation_and_no_op_preserves_capture() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([128, 96]).unwrap();
    view.submit(&scene()).unwrap();
    let before = view.redraw().unwrap();
    assert!(view.set_status_text("Saved part.opd").unwrap());
    assert!(view.capture().is_err());
    let shown = view.redraw().unwrap();
    assert_eq!(shown.scene_generation, before.scene_generation);
    assert_eq!(shown.camera_generation, before.camera_generation);
    assert_eq!(shown.overlay_generation, 1);
    assert!(!view.set_status_text("Saved part.opd").unwrap());
    assert_eq!(view.capture().unwrap(), shown);
    verify_overlay_draw_policy(&recorder);
    assert!(view.set_status_text("").unwrap());
    assert_eq!(view.redraw().unwrap().overlay_generation, 2);
}
fn verify_overlay_draw_policy(recorder: &RecordingDevice) {
    let recorded = recorder.state.lock().unwrap();
    let frame = recorded.frames.last().unwrap();
    assert_eq!(frame.draws().len(), 2);
    assert!(frame.draws().last().unwrap().textured_style().is_some());
    let (test, write, _) = frame.layers().last().unwrap().depth_policy();
    assert!(!test && !write);
}
#[test]
fn invalid_status_and_exhausted_generation_preserve_the_previous_frame() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([128, 96]).unwrap();
    view.set_status_text("Saved").unwrap();
    let before = view.redraw().unwrap();
    assert!(
        view.set_status_text("bad\0label")
            .unwrap_err()
            .contains("expected UTF-8 text without NUL")
    );
    assert_eq!(view.capture().unwrap(), before);
    view.overlay = u64::MAX;
    assert!(view.set_status_text("New").is_err());
    view.overlay = before.overlay_generation;
    assert_eq!(view.capture().unwrap(), before);
    view.close();
    assert!(view.set_status_text("").is_err());
}
#[test]
fn clearing_status_removes_its_structure_before_view_retirement() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([128, 96]).unwrap();
    assert!(!view.set_status_text("").unwrap());
    view.set_status_text("First").unwrap();
    view.set_status_text("Second").unwrap();
    view.redraw().unwrap();
    assert_eq!(
        recorder
            .state
            .lock()
            .unwrap()
            .frames
            .last()
            .unwrap()
            .draws()
            .len(),
        1
    );
    view.set_status_text("").unwrap();
    view.redraw().unwrap();
    assert!(
        recorder
            .state
            .lock()
            .unwrap()
            .frames
            .last()
            .unwrap()
            .draws()
            .is_empty()
    );
    view.set_status_text("Retire").unwrap();
    view.close();
}
