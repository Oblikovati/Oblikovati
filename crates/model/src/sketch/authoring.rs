// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchEntityId, SketchError, SketchOffsetSpline2D, SketchSpline2D, SketchSplineKind,
    circle::require_radius_value, geometry::require_position,
};

/// A defining point is new in working units or shares an existing point in this sketch.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPointInput {
    New([f64; 2]),
    Owned(SketchEntityId),
}
/// Typed primitive inputs; curve construction roles are explicit and plain points have none.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchPrimitiveCreation {
    Point {
        position: [f64; 2],
    },
    Line {
        endpoints: [SketchPointInput; 2],
        construction: bool,
        centerline: bool,
    },
    Circle {
        center: SketchPointInput,
        radius: f64,
        construction: bool,
    },
    Ellipse {
        center: SketchPointInput,
        major_axis: [f64; 2],
        radii: [f64; 2],
        construction: bool,
    },
    Arc {
        center: SketchPointInput,
        endpoints: [SketchPointInput; 2],
        counter_clockwise: bool,
        construction: bool,
    },
    Spline {
        points: Vec<SketchPointInput>,
        kind: SketchSplineKind,
        closed: bool,
        construction: bool,
    },
    OffsetSpline {
        parent: SketchEntityId,
        distance: f64,
        construction: bool,
    },
}
impl SketchPrimitiveCreation {
    pub(crate) fn defining_points(&self) -> Vec<SketchPointInput> {
        match self {
            Self::Point { position } => vec![SketchPointInput::New(*position)],
            Self::Line { endpoints, .. } => endpoints.to_vec(),
            Self::Circle { center, .. } | Self::Ellipse { center, .. } => vec![*center],
            Self::Arc {
                center, endpoints, ..
            } => vec![*center, endpoints[0], endpoints[1]],
            Self::Spline { points, .. } => points.clone(),
            Self::OffsetSpline { .. } => Vec::new(),
        }
    }
    pub(crate) fn validate(&self, geometry: &Sketch2D) -> Result<(), SketchError> {
        self.validate_defining_points(geometry)?;
        self.validate_curve_inputs(geometry)
    }
    fn validate_defining_points(&self, geometry: &Sketch2D) -> Result<(), SketchError> {
        for point in self.defining_points() {
            match point {
                SketchPointInput::New(position) => require_position(position)?,
                SketchPointInput::Owned(id) => {
                    geometry.point(id)?;
                }
            }
        }
        Ok(())
    }
    fn validate_curve_inputs(&self, geometry: &Sketch2D) -> Result<(), SketchError> {
        if let Self::Circle { radius, .. } = self {
            require_radius_value(*radius)?;
        }
        if let Self::Ellipse {
            major_axis, radii, ..
        } = self
        {
            super::ellipse::require_ellipse_frame(*major_axis, *radii)?;
        }
        if let Self::Spline { points, kind, .. } = self {
            super::spline::require_spline_intent(points.len(), *kind)?;
        }
        if let Self::OffsetSpline {
            parent, distance, ..
        } = self
        {
            geometry.spline(*parent)?;
            super::spline::require_offset_distance(*distance)?;
        }
        Ok(())
    }
    pub(crate) fn append_curve(
        &self,
        geometry: &mut Sketch2D,
        id: SketchEntityId,
        points: &[SketchEntityId],
    ) -> Result<(), SketchError> {
        if matches!(self, Self::Point { .. }) {
            return Ok(());
        }
        self.append_geometry(geometry, id, points)?;
        geometry.set_curve_construction(&[id], self.construction())?;
        if let Self::Line { centerline, .. } = self {
            geometry.set_line_centerline(id, *centerline)?;
        }
        Ok(())
    }
    fn append_geometry(
        &self,
        geometry: &mut Sketch2D,
        id: SketchEntityId,
        points: &[SketchEntityId],
    ) -> Result<(), SketchError> {
        match self {
            Self::Point { .. } => return Ok(()),
            Self::Line { .. } => geometry.add_line(id, [points[0], points[1]])?,
            Self::Ellipse {
                major_axis, radii, ..
            } => geometry.add_ellipse(super::SketchEllipse2D::new(
                id,
                points[0],
                *major_axis,
                *radii,
            )?)?,
            Self::Circle { radius, .. } => geometry.add_circle(id, points[0], *radius)?,
            Self::Arc {
                counter_clockwise, ..
            } => geometry.add_arc(id, points[0], [points[1], points[2]], *counter_clockwise)?,
            Self::Spline { kind, closed, .. } => {
                geometry.add_spline(SketchSpline2D::new(id, points.to_vec(), *closed, *kind)?)?
            }
            Self::OffsetSpline {
                parent, distance, ..
            } => geometry.add_offset_spline(SketchOffsetSpline2D::new(id, *parent, *distance)?)?,
        }
        Ok(())
    }
    fn construction(&self) -> bool {
        match self {
            Self::Point { .. } => false,
            Self::Line { construction, .. }
            | Self::Circle { construction, .. }
            | Self::Ellipse { construction, .. }
            | Self::Arc { construction, .. }
            | Self::Spline { construction, .. }
            | Self::OffsetSpline { construction, .. } => *construction,
        }
    }
}
