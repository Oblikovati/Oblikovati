// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
use oblikovati_model::sketch::SketchPlane;
impl NativeHeadWindow {
    /// Read an unsnapped sketch cursor using the current native camera.
    /// Example: `window.sketch_point([320,240], plane)?` before building a placement recipe.
    pub fn sketch_point(
        &self,
        pixel: [i32; 2],
        plane: SketchPlane,
    ) -> Result<Option<[f64; 2]>, String> {
        self.view.sketch_point(pixel, plane)
    }
    /// Locate a sketch witness on the current native surface.
    /// Example: `window.sketch_pixel(plane, [2.,3.])?` for an overlay anchor.
    pub fn sketch_pixel(&self, plane: SketchPlane, point: [f64; 2]) -> Result<[i32; 2], String> {
        self.view.sketch_pixel(plane, point)
    }
}
