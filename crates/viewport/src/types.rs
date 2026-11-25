// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Copy, Debug)]
pub struct ViewportOptions {
    pub validation: bool,
    pub physical_device: Option<usize>,
}
impl Default for ViewportOptions {
    fn default() -> Self {
        Self {
            validation: true,
            physical_device: None,
        }
    }
}
#[derive(Clone, Copy, Debug)]
pub struct ViewportTriangle {
    pub positions: [[f32; 3]; 3],
    pub color: [f32; 4],
}
#[derive(Clone, Debug)]
pub struct ViewportScene {
    pub background: [f64; 3],
    pub triangles: Vec<ViewportTriangle>,
}
impl ViewportScene {
    pub(crate) fn validate(&self) -> Result<(), String> {
        if self
            .background
            .iter()
            .any(|value| !value.is_finite() || !(0.0..=1.0).contains(value))
        {
            return Err(format!(
                "viewport background {:?}: expected three finite linear RGB channels in 0..=1",
                self.background
            ));
        }
        for (index, triangle) in self.triangles.iter().enumerate() {
            triangle.validate(index)?;
        }
        Ok(())
    }
}
impl ViewportTriangle {
    fn validate(&self, index: usize) -> Result<(), String> {
        if self
            .positions
            .iter()
            .flatten()
            .any(|value| !value.is_finite())
        {
            return Err(format!(
                "viewport triangle {index} positions {:?}: expected finite presentation coordinates",
                self.positions
            ));
        }
        self.validate_color(index)
    }
    fn validate_color(&self, index: usize) -> Result<(), String> {
        if self
            .color
            .iter()
            .any(|value| !value.is_finite() || !(0.0..=1.0).contains(value))
        {
            return Err(format!(
                "viewport triangle {index} color {:?}: expected finite linear RGBA channels in 0..=1",
                self.color
            ));
        }
        Ok(())
    }
}
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ViewportCamera {
    pub eye: [f64; 3],
    pub target: [f64; 3],
    pub up: [f64; 3],
    pub scale: f64,
    pub near: f64,
    pub far: f64,
    /// Draw with depth foreshortening; `scale` stays the view's height at the target.
    pub perspective: bool,
}
impl Default for ViewportCamera {
    fn default() -> Self {
        Self {
            eye: [0., 0., 20.],
            target: [0.; 3],
            up: [0., 1., 0.],
            scale: 20.,
            near: 1.,
            far: 100.,
            perspective: false,
        }
    }
}
impl ViewportCamera {
    /// The angle that shows `scale` of height at the target, in degrees: switching projection
    /// keeps what lies at the target the same size on screen.
    #[cfg(any(target_os = "linux", target_os = "windows", test))]
    pub(crate) fn perspective_field_of_view(&self) -> f64 {
        let delta: [f64; 3] = std::array::from_fn(|axis| self.eye[axis] - self.target[axis]);
        let distance = delta.iter().map(|value| value * value).sum::<f64>().sqrt();
        2. * (self.scale / 2. / distance).atan().to_degrees()
    }
    pub(crate) fn validate(&self) -> Result<(), String> {
        oblikovati_model::part_design_view::validate_part_camera(&self.part_view_camera())
            .map_err(|cause| format!("viewport camera {self:?}: {cause}"))
    }
    /// Copy the complete viewport into a part view; example: `camera.part_view_camera()`.
    pub fn part_view_camera(&self) -> oblikovati_api::types::PartDesignViewCamera {
        use oblikovati_api::types::{
            ORTHOGRAPHIC_PROJECTION, PERSPECTIVE_PROJECTION, Point, Vector,
        };
        oblikovati_api::types::PartDesignViewCamera {
            eye: Point {
                x: self.eye[0],
                y: self.eye[1],
                z: self.eye[2],
            },
            target: Point {
                x: self.target[0],
                y: self.target[1],
                z: self.target[2],
            },
            up: Vector {
                x: self.up[0],
                y: self.up[1],
                z: self.up[2],
            },
            scale: self.scale,
            near: self.near,
            far: self.far,
            projection: if self.perspective {
                PERSPECTIVE_PROJECTION
            } else {
                ORTHOGRAPHIC_PROJECTION
            },
        }
    }
    /// Restore an authored part view; example: `ViewportCamera::from_part_view_camera(&saved)?`.
    pub fn from_part_view_camera(
        camera: &oblikovati_api::types::PartDesignViewCamera,
    ) -> Result<Self, String> {
        oblikovati_model::part_design_view::validate_part_camera(camera)?;
        Ok(Self {
            eye: [camera.eye.x, camera.eye.y, camera.eye.z],
            target: [camera.target.x, camera.target.y, camera.target.z],
            up: [camera.up.x, camera.up.y, camera.up.z],
            scale: camera.scale,
            near: camera.near,
            far: camera.far,
            perspective: camera.projection == oblikovati_api::types::PERSPECTIVE_PROJECTION,
        })
    }
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ViewportCapture {
    pub view_id: u64,
    pub scene_generation: u64,
    pub camera_generation: u64,
    pub surface_generation: u64,
    pub overlay_generation: u64,
    pub extent: [u32; 2],
    pub rgba: Vec<u8>,
}
/// GUI publication result; `ViewportGuiSubmission::SurfaceChanged(extent)` requests a new packet.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportGuiSubmission {
    Accepted,
    SurfaceChanged([u32; 2]),
}
/// A frame completes only for its observed surface; a resize requires a new presentation input.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ViewportPresentation {
    Completed(ViewportCapture),
    SurfaceChanged([u32; 2]),
}
/// Borrowed color pixels from an accepted viewport frame, in top-left row order.
#[derive(Clone, Copy)]
pub enum ViewportPreviewPixels<'a> {
    Rgb(&'a [[u8; 3]]),
    Rgba(&'a [[u8; 4]]),
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportEvent {
    Input(crate::ViewportInput),
    Expose,
    Resized([u32; 2]),
    Focus(bool),
    Mapped(bool),
    CloseRequested,
}
#[cfg(any(target_os = "linux", target_os = "windows", test))]
pub(crate) fn valid_extent(extent: [u32; 2]) -> Result<[i32; 2], String> {
    if extent
        .iter()
        .any(|value| *value == 0 || *value > i32::MAX as u32)
    {
        return Err(format!(
            "viewport extent {extent:?}: expected two positive 31-bit dimensions"
        ));
    }
    Ok(extent.map(|value| value as i32))
}

#[cfg(test)]
#[path = "../tests/validation.rs"]
mod tests;
