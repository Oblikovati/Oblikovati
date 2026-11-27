// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, SketchPlane, cross_product, normalize_direction, vector_length};

/// Select the sketch axis fixed by a resolved reference direction; example: `SketchFrameAxis::Y`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SketchFrameAxis {
    X,
    Y,
}

/// One evaluated axis reference; example: `SketchFrameAlignment { direction: [1., 0., 0.], axis: SketchFrameAxis::X, reverse: false }`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchFrameAlignment {
    pub direction: [f64; 3],
    pub axis: SketchFrameAxis,
    pub reverse: bool,
}
impl SketchPlane {
    /// Apply independent origin and alignment changes; example: `frame.with_frame_edit(Some([1., 2., 0.]), None)?`.
    pub fn with_frame_edit(
        self,
        origin: Option<[f64; 3]>,
        alignment: Option<SketchFrameAlignment>,
    ) -> Result<Self, SketchError> {
        let origin = origin.unwrap_or(self.recipe().origin);
        if let Some(alignment) = alignment {
            return self.with_orientation(
                alignment.direction,
                alignment.axis,
                alignment.reverse,
                Some(origin),
            );
        }
        let recipe = self.recipe();
        Self::new(origin, recipe.x_axis, recipe.y_axis, self.options())
    }
    /// Reframe from a current resolved direction without a stale fallback; example: `plane.with_orientation([0.0, 0.0, 1.0], SketchFrameAxis::Y, false, None)?`.
    pub fn with_orientation(
        self,
        direction: [f64; 3],
        axis: SketchFrameAxis,
        reverse: bool,
        origin: Option<[f64; 3]>,
    ) -> Result<Self, SketchError> {
        let projected = reference_projection(self, direction)?;
        let selected = projected.map(|value| if reverse { -value } else { value });
        let (x_axis, y_axis) = oriented_frame_axes(self.normal(), selected, axis)?;
        Self::new(
            origin.unwrap_or(self.recipe().origin),
            x_axis,
            y_axis,
            self.options(),
        )
    }
}

fn reference_projection(plane: SketchPlane, direction: [f64; 3]) -> Result<[f64; 3], SketchError> {
    let unit = normalize_direction(direction)?;
    // The double cross avoids subtracting nearly equal normal components (#3736).
    let tangent = cross_product(plane.normal(), unit);
    let sine = vector_length(tangent);
    let minimum = plane.options().angular_deviation;
    if sine <= minimum {
        return Err(SketchError::Definition(format!(
            "sketch orientation direction {direction:?} against normal {:?}, sine {sine:?}: expected an in-plane projection above angular deviation {minimum:?}",
            plane.normal()
        )));
    }
    normalize_direction(cross_product(tangent, plane.normal()))
}

fn oriented_frame_axes(
    normal: [f64; 3],
    selected: [f64; 3],
    axis: SketchFrameAxis,
) -> Result<([f64; 3], [f64; 3]), SketchError> {
    match axis {
        SketchFrameAxis::X => Ok((
            selected,
            normalize_direction(cross_product(normal, selected))?,
        )),
        SketchFrameAxis::Y => Ok((
            normalize_direction(cross_product(selected, normal))?,
            selected,
        )),
    }
}
