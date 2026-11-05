// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, require_position};
/// Authored rectangle seed constructors; these do not add constraints or certify a profile.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchRectangleSeed {
    Corners([[f64; 2]; 2]),
    Center([[f64; 2]; 2]),
    /// Placement retains the clicked corner direction; raw API Center uses absolute half-extents.
    DirectedCenter([[f64; 2]; 2]),
    ThreePoints([[f64; 2]; 3]),
}
impl SketchRectangleSeed {
    /// Resolve four ordered seed vertices before reserving any entity identities.
    /// Example: `SketchRectangleSeed::Corners([[0.0, 0.0], [3.0, 2.0]]).vertices()?;`.
    pub fn vertices(self) -> Result<[[f64; 2]; 4], SketchError> {
        let vertices = match self {
            Self::Corners(points) => rectangle_corners(points)?,
            Self::Center(points) => centered_rectangle(points, false)?,
            Self::DirectedCenter(points) => centered_rectangle(points, true)?,
            Self::ThreePoints(points) => three_point_rectangle(points)?,
        };
        for position in vertices {
            require_position(position)?;
        }
        Ok(vertices)
    }
}
fn rectangle_corners([first, opposite]: [[f64; 2]; 2]) -> Result<[[f64; 2]; 4], SketchError> {
    require_position(first)?;
    require_position(opposite)?;
    Ok([
        first,
        [opposite[0], first[1]],
        opposite,
        [first[0], opposite[1]],
    ])
}
fn centered_rectangle(
    [center, corner]: [[f64; 2]; 2],
    directed: bool,
) -> Result<[[f64; 2]; 4], SketchError> {
    require_position(center)?;
    require_position(corner)?;
    let half = [corner[0] - center[0], corner[1] - center[1]];
    // Go aa8297d1: AddRectangleByCenter uses absolute extents; CenterRectangleRecipe retains direction.
    let half = if directed { half } else { half.map(f64::abs) };
    require_position(half)?;
    rectangle_corners([
        [center[0] - half[0], center[1] - half[1]],
        [center[0] + half[0], center[1] + half[1]],
    ])
}
fn three_point_rectangle(points: [[f64; 2]; 3]) -> Result<[[f64; 2]; 4], SketchError> {
    for position in points {
        require_position(position)?;
    }
    let [first, second, third] = points;
    let normal = rectangle_normal(first, second)?;
    let offset = [third[0] - second[0], third[1] - second[1]];
    require_position(offset)?;
    let distance = offset[0] * normal[0] + offset[1] * normal[1];
    let depth = normal.map(|coordinate| coordinate * distance);
    require_position(depth)?;
    Ok([
        first,
        second,
        [second[0] + depth[0], second[1] + depth[1]],
        [first[0] + depth[0], first[1] + depth[1]],
    ])
}
fn rectangle_normal(first: [f64; 2], second: [f64; 2]) -> Result<[f64; 2], SketchError> {
    let perpendicular = [-(second[1] - first[1]), second[0] - first[0]];
    // Preserve the Go seed expression and operation order (aa8297d1, #3563).
    let length = (perpendicular[0] * perpendicular[0] + perpendicular[1] * perpendicular[1]).sqrt();
    if !length.is_finite() || length == 0.0 {
        return Err(SketchError::Definition(format!(
            "rectangle first edge {first:?} -> {second:?}, length {length}: expected a finite nonzero representable length"
        )));
    }
    // Retain the pinned Go reciprocal/multiply order for the placement seed.
    let inverse = 1.0 / length;
    let normal = perpendicular.map(|coordinate| coordinate * inverse);
    require_position(normal)?;
    Ok(normal)
}
