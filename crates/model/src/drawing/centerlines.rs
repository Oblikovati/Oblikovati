// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingAnnotationDefinition, DrawingDocument, DrawingError, annotation::unique_name};

/// The distance centre lines extend past each view edge in sheet millimetres.
pub const CENTERLINE_OVERSHOOT_MM: f64 = 4.;
const PERIOD_MM: f64 = 18.;
const STROKES_MM: [[f64; 2]; 2] = [[0., 12.], [14., 16.]];

/// Derive lower and upper sheet bounds from sampled projection points.
/// Example: `drawing_view_bounds([[0., 0.], [40., 20.]])`.
pub fn drawing_view_bounds(points: impl IntoIterator<Item = [f64; 2]>) -> Option<[[f64; 2]; 2]> {
    let mut points = points
        .into_iter()
        .filter(|point| point.iter().all(|v| v.is_finite()));
    let first = points.next()?;
    Some(points.fold([first, first], |[lower, upper], point| {
        [
            std::array::from_fn(|axis| lower[axis].min(point[axis])),
            std::array::from_fn(|axis| upper[axis].max(point[axis])),
        ]
    }))
}

impl DrawingDocument {
    /// Record the view association, e.g. `drawing.add_centerlines("Axes", "Top")?`.
    pub fn add_centerlines(
        &mut self,
        requested_name: &str,
        view_name: &str,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let target = &self.sheets()[sheet];
        if !target.views.iter().any(|view| view.name == view_name) {
            return Err(DrawingError(format!(
                "centerline view {view_name:?}: expected a view on the active sheet"
            )));
        }
        let annotation = DrawingAnnotationDefinition::Centerlines {
            name: unique_name(requested_name, &target.annotations),
            view_name: view_name.into(),
            characteristic: None,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, annotation.clone())?;
        Ok(annotation)
    }
}

/// Orthogonal dash-dot strokes derived from the current view bounds.
pub struct DrawingCenterlines {
    origin: [f64; 2],
    length: [f64; 2],
    center: [f64; 2],
}

impl DrawingCenterlines {
    /// Derive the stroke axes, e.g. `DrawingCenterlines::from_view_bounds([[0., 0.], [40., 20.]])`.
    pub fn from_view_bounds([lower, upper]: [[f64; 2]; 2]) -> Self {
        Self {
            origin: lower.map(|value| value - CENTERLINE_OVERSHOOT_MM),
            length: std::array::from_fn(|axis| {
                upper[axis] - lower[axis] + 2. * CENTERLINE_OVERSHOOT_MM
            }),
            center: std::array::from_fn(|axis| lower[axis].midpoint(upper[axis])),
        }
    }

    /// Count nominal dashes and dots, e.g. `centerlines.curve_count()`.
    pub fn curve_count(&self) -> usize {
        self.length
            .into_iter()
            .map(|length| {
                let cycles = (length / PERIOD_MM).floor();
                let tail = length % PERIOD_MM;
                (cycles as usize)
                    .saturating_mul(2)
                    .saturating_add(usize::from(tail > STROKES_MM[0][0]))
                    .saturating_add(usize::from(tail > STROKES_MM[1][0]))
            })
            .fold(0, usize::saturating_add)
    }

    /// Sample the visible dashes and dots, e.g. `centerlines.visible_segments([420., 297.], 0.5)`.
    pub fn visible_segments(&self, sheet: [f64; 2], mm_per_pixel: f64) -> Vec<[[f64; 2]; 2]> {
        let mut segments = Vec::new();
        for axis in [0, 1] {
            self.append_axis(axis, sheet, mm_per_pixel, &mut segments);
        }
        segments
    }

    fn append_axis(
        &self,
        axis: usize,
        sheet: [f64; 2],
        mm_per_pixel: f64,
        segments: &mut Vec<[[f64; 2]; 2]>,
    ) {
        let cross_axis = 1 - axis;
        if !(0. ..=sheet[cross_axis]).contains(&self.center[cross_axis]) {
            return;
        }
        let origin = self.origin[axis];
        let start = (-origin).max(0.);
        let end = self.length[axis].min(sheet[axis] - origin);
        if start >= end {
            return;
        }
        let point = |distance| {
            let mut point = self.center;
            point[axis] = origin + distance;
            point
        };
        if PERIOD_MM < mm_per_pixel {
            segments.push([point(start), point(end)]);
            return;
        }
        for cycle in (start / PERIOD_MM).floor() as usize..(end / PERIOD_MM).ceil() as usize {
            for [a, b] in STROKES_MM {
                let offset = cycle as f64 * PERIOD_MM;
                let from = (offset + a).max(start);
                let to = (offset + b).min(end);
                if from < to {
                    segments.push([point(from), point(to)]);
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dash_dot_strokes_exceed_the_bounds_and_keep_their_phase_when_clipped() {
        let bounds = drawing_view_bounds([[50., 20.], [10., 40.], [f64::NAN, 90.]]).unwrap();
        let axes = DrawingCenterlines::from_view_bounds(bounds);
        let segments = axes.visible_segments([100., 100.], 0.25);
        assert_eq!(axes.curve_count(), 8);
        assert_eq!(segments.len(), 8);
        assert_eq!(segments[0], [[6., 30.], [18., 30.]]);
        assert_eq!(segments[1], [[20., 30.], [22., 30.]]);
        assert_eq!(segments[4], [[42., 30.], [54., 30.]]);
        assert_eq!(segments[5], [[30., 16.], [30., 28.]]);
        let cropped = axes.visible_segments([21., 60.], 0.25);
        assert_eq!(cropped, [segments[0], [[20., 30.], [21., 30.]]]);
    }
}
