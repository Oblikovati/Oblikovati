// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;
use rustycad_tkmath::gp::Point3;
impl Viewport {
    /// Map a framebuffer pixel to the unsnapped sketch plane; edge-on planes return `None`.
    /// Example: `view.sketch_point([320,240], SketchPlane::xy())?`.
    pub fn sketch_point(
        &self,
        pixel: [i32; 2],
        plane: SketchPlane,
    ) -> Result<Option<[f64; 2]>, String> {
        self.require_projection_pixel(pixel)?;
        let [x, y, z, dx, dy, dz] = self
            .live()?
            .view
            .convert_with_proj(pixel[0], pixel[1])
            .map_err(|cause| format!("viewport pixel {pixel:?}: {cause}"))?;
        // UI conditioning only: retain Go aa8297d1 screenToSketch's DefaultTolerance.
        oblikovati_martensite::sketch_projection(plane, [x, y, z], [dx, dy, dz], 1e-9)
    }
    /// Project a sketch point to the kernel's integer framebuffer pixel convention.
    /// Example: `view.sketch_pixel(SketchPlane::xy(), [2.,3.])?` for a witness endpoint.
    pub fn sketch_pixel(&self, plane: SketchPlane, point: [f64; 2]) -> Result<[i32; 2], String> {
        let [x, y, z] = plane.to_model(point).map_err(|cause| cause.to_string())?;
        self.require_projection_extent()?;
        self.require_representable_projection(Point3::new(x, y, z))?;
        self.live()?
            .view
            .convert_world_to_pixels(x, y, z)
            .map(|(x, y)| [x, y])
            .map_err(|cause| cause.to_string())
    }
    fn require_projection_pixel(&self, pixel: [i32; 2]) -> Result<(), String> {
        let [_, height] = self.require_projection_extent()?;
        if (height - 1).checked_sub(pixel[1]).is_none() {
            return Err(format!(
                "viewport pixel {pixel:?}: expected representable inverted Y for height {height}"
            ));
        }
        Ok(())
    }
    fn require_projection_extent(&self) -> Result<[i32; 2], String> {
        self.live()?;
        let actual = self.target.as_ref().expect("live target").extent();
        if actual != self.extent {
            return Err(format!(
                "viewport extent {actual:?}: expected observed camera extent {:?}; poll events first",
                self.extent
            ));
        }
        crate::types::valid_extent(actual)
    }
    fn require_representable_projection(&self, point: Point3) -> Result<(), String> {
        let projected = self.live()?.view.camera().project(&point);
        let [width, height] = self.extent.map(f64::from);
        let pixel = [
            (projected.x() + 1.) * 0.5 * width,
            height - 1. - (projected.y() + 1.) * 0.5 * height,
        ];
        if pixel.iter().any(|value| {
            !value.is_finite()
                || *value < f64::from(i32::MIN) + 1.
                || *value > f64::from(i32::MAX) - 1.
        }) {
            return Err(format!(
                "viewport projected pixel {pixel:?}: expected finite signed 32-bit coordinates with rounding headroom"
            ));
        }
        Ok(())
    }
}
