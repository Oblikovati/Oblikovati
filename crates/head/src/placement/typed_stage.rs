// SPDX-License-Identifier: GPL-2.0-only
//! What typed field values make of a stage's point: the point a typed length, angle or radius
//! reaches, whichever stage asked for it.
use super::*;
impl SketchPlacementState {
    /// Accept a point and move the fields on to the next stage's.
    pub(super) fn accept_stage_point(&mut self, point: [f64; 2]) -> Result<(), String> {
        self.accept_point(point)?;
        if matches!(self.tool, crate::SketchPlacementTool::ThreePointArc { .. }) {
            // An arc keeps no dimension of its chord: those values are spent on the end point.
            self.fields = Default::default();
            return Ok(());
        }
        let next = self.tool.stage_fields(self.accepted.len()).start;
        self.fields.advance_to_field(next);
        Ok(())
    }
    /// Typed values name a three-point rectangle's second point, whatever the pointer says.
    pub(super) fn typed_stage_point(&self, point: [f64; 2]) -> [f64; 2] {
        use crate::SketchPlacementTool::{
            ThreePointArc, ThreePointCenterRectangle, ThreePointRectangle,
        };
        let ([first], ThreePointRectangle | ThreePointCenterRectangle | ThreePointArc { .. }) =
            (self.accepted.as_slice(), self.tool)
        else {
            return point;
        };
        let delta = [point[0] - first[0], point[1] - first[1]];
        let [length, angle, ..] = self.fields.typed_values();
        let direction = angle.unwrap_or_else(|| delta[1].atan2(delta[0]));
        let reach = length
            .map(|length| length / self.first_axis_scale())
            .unwrap_or_else(|| delta[0].hypot(delta[1]));
        [
            first[0] + reach * direction.cos(),
            first[1] + reach * direction.sin(),
        ]
    }
    /// Enter finishes the shape from here; example: not at an arc's point, which needs a side.
    pub(crate) fn enter_finishes_shape(&self) -> bool {
        self.at_finishing_stage()
            && !matches!(self.tool, crate::SketchPlacementTool::ThreePointArc { .. })
    }
    /// Enter at a stage short of the end: its values name that stage's point.
    pub(crate) fn accept_typed_stage(&mut self) -> Result<bool, String> {
        use crate::SketchPlacementTool::{
            ThreePointArc, ThreePointCenterRectangle, ThreePointRectangle,
        };
        if matches!(self.tool, ThreePointArc { .. }) && self.accepted.len() == 2 {
            // A radius alone cannot say which side the arc bows to: ask for the end again.
            self.accepted.pop();
            self.fields = Default::default();
            return Ok(true);
        }
        if !matches!(
            self.tool,
            ThreePointRectangle | ThreePointCenterRectangle | ThreePointArc { .. }
        ) || self.accepted.len() != 1
        {
            return Ok(false);
        }
        self.fields.lock_typed();
        let cursor = self.cursor_point.unwrap_or(self.accepted[0]);
        let point = self.typed_stage_point(cursor);
        self.accept_stage_point(point)?;
        Ok(true)
    }
    /// A typed width sets the third point square off the first edge, on the pointer's side.
    pub(super) fn typed_third_point(&self) -> Option<[f64; 2]> {
        use crate::SketchPlacementTool::{ThreePointCenterRectangle, ThreePointRectangle};
        let [first, second] = self.accepted.as_slice() else {
            return None;
        };
        let scale = match self.tool {
            ThreePointRectangle => 1.,
            ThreePointCenterRectangle => 0.5,
            _ => return None,
        };
        let width = self.fields.typed_values()[2]? * scale;
        let along = [second[0] - first[0], second[1] - first[1]];
        let length = along[0].hypot(along[1]);
        let normal = [-along[1] / length, along[0] / length];
        let cursor = self.cursor_point.unwrap_or(*second);
        let offset = [cursor[0] - second[0], cursor[1] - second[1]];
        let side = if offset[0] * normal[0] + offset[1] * normal[1] < 0. {
            -1.
        } else {
            1.
        };
        Some([
            second[0] + normal[0] * width * side,
            second[1] + normal[1] * width * side,
        ])
    }
    /// Typed radius and direction set the start; an untyped field follows the pointer.
    pub(super) fn arc_start_at_radius(&self, point: [f64; 2]) -> [f64; 2] {
        let (crate::SketchPlacementTool::CenterPointArc, [center]) =
            (self.tool, self.accepted.as_slice())
        else {
            return point;
        };
        let [radius, angle, _] = self.fields.typed_values();
        let delta = [point[0] - center[0], point[1] - center[1]];
        let length = delta[0].hypot(delta[1]);
        if let Some(angle) = angle {
            let radius = radius.unwrap_or(length);
            return [
                center[0] + radius * angle.cos(),
                center[1] + radius * angle.sin(),
            ];
        }
        let Some(radius) = radius.filter(|_| length > 0.) else {
            return point;
        };
        [
            center[0] + delta[0] / length * radius,
            center[1] + delta[1] / length * radius,
        ]
    }
    /// A typed sweep turns the start about the center, toward the pointer's side.
    pub(super) fn arc_end_at_sweep(&self, point: [f64; 2]) -> [f64; 2] {
        let sweep = self.fields.typed_values()[2];
        let (crate::SketchPlacementTool::CenterPointArc, [center, start], Some(sweep)) =
            (self.tool, self.accepted.as_slice(), sweep)
        else {
            return point;
        };
        let radial = [start[0] - center[0], start[1] - center[1]];
        let toward = [point[0] - center[0], point[1] - center[1]];
        let side = if radial[0] * toward[1] - radial[1] * toward[0] < 0. {
            -1.
        } else {
            1.
        };
        let (sin, cos) = (side * sweep).sin_cos();
        [
            center[0] + radial[0] * cos - radial[1] * sin,
            center[1] + radial[0] * sin + radial[1] * cos,
        ]
    }
    /// The shortest radius that still reaches both ends of an arc; zero for anything else.
    pub(crate) fn least_typed_radius(&self) -> f64 {
        match (self.tool, self.accepted.as_slice()) {
            (crate::SketchPlacementTool::ThreePointArc { .. }, [start, end]) => {
                (end[0] - start[0]).hypot(end[1] - start[1]) / 2.
            }
            _ => 0.,
        }
    }
    /// A typed radius leaves two circles through the ends: take the one the pointer is nearer to.
    fn arc_point_at_radius(&self, pointer: [f64; 2]) -> [f64; 2] {
        let ([start, end], Some(radius)) =
            (self.accepted.as_slice(), self.fields.typed_values()[0])
        else {
            return pointer;
        };
        let middle = [(start[0] + end[0]) / 2., (start[1] + end[1]) / 2.];
        let chord = [end[0] - start[0], end[1] - start[1]];
        let length = chord[0].hypot(chord[1]);
        let normal = [-chord[1] / length, chord[0] / length];
        let rise = (radius * radius - length * length / 4.).max(0.).sqrt();
        let center_at = |side: f64| {
            [
                middle[0] + normal[0] * rise * side,
                middle[1] + normal[1] * rise * side,
            ]
        };
        let off_circle = |center: [f64; 2]| {
            ((pointer[0] - center[0]).hypot(pointer[1] - center[1]) - radius).abs()
        };
        let center = if off_circle(center_at(1.)) <= off_circle(center_at(-1.)) {
            center_at(1.)
        } else {
            center_at(-1.)
        };
        let bulge = (pointer[0] - middle[0]) * normal[0] + (pointer[1] - middle[1]) * normal[1];
        let side = if bulge < 0. { -1. } else { 1. };
        [
            center[0] + normal[0] * radius * side,
            center[1] + normal[1] * radius * side,
        ]
    }
    /// The point that finishes the shape once typed values have had their say over the pointer's.
    pub(super) fn typed_finishing_point(&self, pointer: [f64; 2]) -> [f64; 2] {
        if matches!(self.tool, crate::SketchPlacementTool::ThreePointArc { .. }) {
            return self.arc_point_at_radius(pointer);
        }
        self.typed_third_point()
            .unwrap_or(self.arc_end_at_sweep(pointer))
    }
    pub(super) fn typed_end(&self, start: [f64; 2]) -> [f64; 2] {
        if self.tool == crate::SketchPlacementTool::CenterPointArc {
            return self.arc_end_at_sweep(self.cursor_point.unwrap_or(start));
        }
        if matches!(self.tool, crate::SketchPlacementTool::ThreePointArc { .. }) {
            return self.arc_point_at_radius(self.cursor_point.unwrap_or(start));
        }
        if let Some(third) = self.typed_third_point() {
            return third;
        }
        let cursor = self.cursor_point.unwrap_or(start);
        let delta = [cursor[0] - start[0], cursor[1] - start[1]];
        let reach = |typed: Option<f64>, axis: usize, fraction: f64| {
            let side = if delta[axis] < 0. { -1. } else { 1. };
            match typed {
                Some(size) => side * size * fraction,
                None if delta[axis] != 0. => delta[axis],
                None => TYPED_SEED_REACH,
            }
        };
        let [first, second, _] = self.fields.typed_values();
        let offset = match self.tool {
            crate::SketchPlacementTool::Circle => [reach(first, 0, 0.5), 0.],
            crate::SketchPlacementTool::CenterRectangle => {
                [reach(first, 0, 0.5), reach(second, 1, 0.5)]
            }
            _ => [reach(first, 0, 1.), reach(second, 1, 1.)],
        };
        [start[0] + offset[0], start[1] + offset[1]]
    }
}
