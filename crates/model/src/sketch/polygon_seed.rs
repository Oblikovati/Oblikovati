// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, require_position};
/// Regular polygon seed; its points require the regularity constraints on creation.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchPolygonSeed {
    pub center: [f64; 2],
    pub through: [f64; 2],
    pub sides: usize,
    pub inscribed: bool,
}
impl SketchPolygonSeed {
    /// Resolve authored vertices in Go order without reserving identities.
    /// Example: `seed.vertices()?` before creating a constrained polygon.
    pub fn vertices(self) -> Result<Vec<[f64; 2]>, SketchError> {
        let (base, radius, step, offset) = self.polar_seed()?;
        let mut vertices = Vec::new();
        vertices.try_reserve_exact(self.sides).map_err(|cause| {
            SketchError::Definition(format!(
                "polygon sides {}: expected an allocatable vertex count: {cause}",
                self.sides
            ))
        })?;
        for index in 0..self.sides {
            let angle = base - offset + step * index as f64;
            let point = [
                self.center[0] + radius * angle.cos(),
                self.center[1] + radius * angle.sin(),
            ];
            require_position(point)?;
            vertices.push(point);
        }
        Ok(vertices)
    }
    pub(crate) fn boundary(self) -> Result<(Vec<[f64; 2]>, f64), SketchError> {
        let vertices = self.vertices()?;
        let delta = [
            vertices[0][0] - self.center[0],
            vertices[0][1] - self.center[1],
        ];
        // Go derives the construction radius from the first authored vertex (#3563, aa8297d1).
        let radius = (delta[0] * delta[0] + delta[1] * delta[1]).sqrt();
        super::circle::require_radius_value(radius)?;
        Ok((vertices, radius))
    }
    fn polar_seed(self) -> Result<(f64, f64, f64, f64), SketchError> {
        let vector = self.first_radius_vector()?;
        let radius = (vector[0] * vector[0] + vector[1] * vector[1]).sqrt();
        super::circle::require_radius_value(radius)?;
        let step = 2.0 * std::f64::consts::PI / self.sides as f64;
        let offset = if self.inscribed { 0.0 } else { step / 2.0 };
        let vertex_radius = if self.inscribed {
            radius
        } else {
            radius / offset.cos()
        };
        super::circle::require_radius_value(vertex_radius)?;
        Ok((vector[1].atan2(vector[0]), vertex_radius, step, offset))
    }
    fn first_radius_vector(self) -> Result<[f64; 2], SketchError> {
        require_position(self.center)?;
        require_position(self.through)?;
        if self.sides < 3 {
            return Err(SketchError::Definition(format!(
                "polygon sides {}: expected at least 3",
                self.sides
            )));
        }
        Ok([
            self.through[0] - self.center[0],
            self.through[1] - self.center[1],
        ])
    }
}
