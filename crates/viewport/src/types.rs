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
#[derive(Clone, Copy, Debug)]
pub struct ViewportCamera {
    pub eye: [f64; 3],
    pub target: [f64; 3],
    pub up: [f64; 3],
    pub scale: f64,
    pub near: f64,
    pub far: f64,
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
        }
    }
}
impl ViewportCamera {
    pub(crate) fn validate(&self) -> Result<(), String> {
        let vectors = [self.eye, self.target, self.up];
        if vectors.iter().flatten().any(|value| !value.is_finite())
            || self.eye == self.target
            || self.up == [0.; 3]
        {
            return Err(format!(
                "viewport camera {self:?}: expected finite distinct eye/target and nonzero up direction"
            ));
        }
        self.validate_magnitudes()?;
        self.validate_projection()
    }
    fn validate_magnitudes(&self) -> Result<(), String> {
        let direction: [f64; 3] = std::array::from_fn(|axis| self.target[axis] - self.eye[axis]);
        for vector in [direction, self.up] {
            let squared: f64 = vector.iter().map(|value| value * value).sum();
            if !squared.is_finite() || squared == 0. {
                return Err(format!(
                    "viewport camera {self:?}: expected finite nonzero squared direction magnitudes"
                ));
            }
        }
        Ok(())
    }
    fn validate_projection(&self) -> Result<(), String> {
        if !self.scale.is_finite()
            || self.scale <= 0.
            || !self.near.is_finite()
            || !self.far.is_finite()
            || self.near <= 0.
            || self.far <= self.near
        {
            return Err(format!(
                "viewport camera {self:?}: expected positive scale and finite 0 < near < far"
            ));
        }
        Ok(())
    }
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ViewportCapture {
    pub view_id: u64,
    pub scene_generation: u64,
    pub camera_generation: u64,
    pub surface_generation: u64,
    pub extent: [u32; 2],
    pub rgba: Vec<u8>,
}
/// A frame completes only for its observed surface; a resize requires a new presentation input.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ViewportPresentation {
    Completed(ViewportCapture),
    SurfaceChanged([u32; 2]),
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportEvent {
    Expose,
    Resized([u32; 2]),
    Focus(bool),
    CloseRequested,
}
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
