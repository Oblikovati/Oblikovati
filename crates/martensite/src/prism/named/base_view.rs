// SPDX-License-Identifier: GPL-2.0-only
//! Standard drawing views retain exact curves and map them to sheet coordinates.
use super::{DrawingProjection, DrawingProjectionFrame, NamedPrismSnapshot, NamedSolid};
use martensite_tkmath::bnd::BoundingBox3;
use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;
use oblikovati_api::types::*;
use oblikovati_model::{
    drawing::DrawingBaseViewDefinition, history::DocumentEditHistory, identity::DocumentGuid,
};

/// Exact projected curves and their authored sheet placement.
pub struct DrawingBaseProjection {
    pub curves: DrawingProjection,
    center: [f64; 2],
    scale: f64,
    rotation: [f64; 2],
}
impl DrawingBaseProjection {
    /// Sample a curve for sheet display; example: `view.sheet_polyline(true, 0, history, guid, 0.05)?`.
    pub fn sheet_polyline(
        &self,
        visible: bool,
        index: usize,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        deflection_mm: f64,
    ) -> Result<Vec<[f64; 2]>, String> {
        let curves = if visible {
            &self.curves.visible
        } else {
            &self.curves.hidden
        };
        let curve = curves.get(index).ok_or_else(|| {
            format!(
                "drawing curve index {index}: expected an index below {}",
                curves.len()
            )
        })?;
        let deflection = deflection_mm / self.scale / 10.;
        curve
            .display_parameters(history, document, deflection)?
            .into_iter()
            .map(|parameter| self.sheet_point(visible, index, history, document, parameter))
            .collect()
    }
    /// Evaluate a curve on the sheet in millimetres; example: `view.sheet_point(true, 0, history, guid, parameter)?`.
    pub fn sheet_point(
        &self,
        visible: bool,
        index: usize,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        parameter: f64,
    ) -> Result<[f64; 2], String> {
        let curves = if visible {
            &self.curves.visible
        } else {
            &self.curves.hidden
        };
        let curve = curves.get(index).ok_or_else(|| {
            format!(
                "drawing curve index {index}: expected an index below {}",
                curves.len()
            )
        })?;
        let point = curve.point_at(history, document, parameter)?;
        let [x, y] = [point[0] * self.scale * 10., -point[1] * self.scale * 10.];
        let [sin, cos] = self.rotation;
        let result = [
            self.center[0] + (x * cos - y * sin),
            self.center[1] + (x * sin + y * cos),
        ];
        if result.iter().any(|value| !value.is_finite()) {
            return Err(format!(
                "drawing sheet point {result:?}: expected finite millimetre coordinates"
            ));
        }
        Ok(result)
    }
}
impl NamedPrismSnapshot {
    /// Project a standard view of the current body; example: `body.drawing_base_view(history, guid, key, &view)?`.
    pub fn drawing_base_view(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
        view: &DrawingBaseViewDefinition,
    ) -> Result<DrawingBaseProjection, String> {
        self.body_report(history, document, key)?;
        drawing_base_view_bodies(history, document, &[self], view)
    }
}
/// Project all current bodies in one standard view; example: `drawing_base_view_bodies(history, guid, &bodies, &view)?`.
pub fn drawing_base_view_bodies(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    bodies: &[&NamedPrismSnapshot],
    view: &DrawingBaseViewDefinition,
) -> Result<DrawingBaseProjection, String> {
    let shapes = bodies
        .iter()
        .map(|body| {
            body.body_reference_key(history, document)?;
            Ok(&body.snapshot.shape)
        })
        .collect::<Result<Vec<_>, String>>()?;
    base_projection(view, &shapes, |frame| {
        super::projection::drawing_projection_bodies(history, document, bodies, frame)
    })
}
/// Project current named solids in a standard view; example: `drawing_base_view_solids(history, guid, &bodies, view)?`.
pub fn drawing_base_view_solids(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    bodies: &[&NamedSolid],
    view: &DrawingBaseViewDefinition,
) -> Result<DrawingBaseProjection, String> {
    let shapes = bodies
        .iter()
        .map(|body| body.native_shape(history, document))
        .collect::<Result<Vec<_>, String>>()?;
    base_projection(view, &shapes, |frame| {
        super::projection::drawing_projection_solids(history, document, bodies, frame)
    })
}
fn base_projection(
    view: &DrawingBaseViewDefinition,
    shapes: &[&super::Shape],
    project: impl FnOnce(DrawingProjectionFrame) -> Result<DrawingProjection, String>,
) -> Result<DrawingBaseProjection, String> {
    let center = [view.center_x_mm, view.center_y_mm];
    if !view.scale.is_finite() || view.scale <= 0. || center.iter().any(|value| !value.is_finite())
    {
        return Err(format!(
            "drawing view scale {}, center {center:?}: expected positive finite scale and finite center",
            view.scale
        ));
    }
    let frame = standard_frame(view.orientation, body_bounds_center(shapes)?)?;
    let rotation = sheet_rotation(view.rotation_deg)?;
    Ok(DrawingBaseProjection {
        curves: project(frame)?,
        center,
        scale: view.scale,
        rotation,
    })
}
fn sheet_rotation(angle: f64) -> Result<[f64; 2], String> {
    if !angle.is_finite() {
        return Err(format!("drawing rotation {angle}: expected finite degrees"));
    }
    // Preserve exact axes for quarter turns and avoid overflow for large authored angles.
    Ok(match angle.rem_euclid(360.) {
        0. => [0., 1.],
        90. => [1., 0.],
        180. => [0., -1.],
        270. => [-1., 0.],
        angle => {
            let (sin, cos) = angle.to_radians().sin_cos();
            [sin, cos]
        }
    })
}
fn body_bounds_center(shapes: &[&super::Shape]) -> Result<[f64; 3], String> {
    let mut bounds = BoundingBox3::default();
    for shape in shapes {
        BRepBndLib::add_optimal(shape, &mut bounds, false, false)
            .map_err(|cause| format!("drawing body bounds: {cause}"))?;
    }
    let point = bounds
        .center()
        .ok_or("drawing bodies: expected non-empty finite bounds")?;
    let origin = [point.x(), point.y(), point.z()];
    if origin.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "drawing body center {origin:?}: expected finite coordinates"
        ));
    }
    Ok(origin)
}
fn standard_frame(
    orientation: BaseViewOrientation,
    origin: [f64; 3],
) -> Result<DrawingProjectionFrame, String> {
    let (normal, x_direction) = match orientation {
        BASE_VIEW_FRONT => ([0., 1., 0.], [-1., 0., 0.]),
        BASE_VIEW_BACK => ([0., -1., 0.], [1., 0., 0.]),
        BASE_VIEW_TOP => ([0., 0., -1.], [-1., 0., 0.]),
        BASE_VIEW_BOTTOM => ([0., 0., 1.], [1., 0., 0.]),
        BASE_VIEW_RIGHT => ([-1., 0., 0.], [0., -1., 0.]),
        BASE_VIEW_LEFT => ([1., 0., 0.], [0., 1., 0.]),
        BASE_VIEW_ISO => ([-1., 1., -1.], [-1., -1., 0.]),
        _ => {
            return Err(format!(
                "drawing orientation {orientation:?}: expected a standard base orientation"
            ));
        }
    };
    // HLR's positive normal points toward the eye; Go's view direction points into the screen.
    // Reversing it also reverses the frame's Y axis, which sheet_point restores.
    let normal = normal.map(|value| -value);
    Ok(DrawingProjectionFrame {
        origin,
        normal,
        x_direction,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sheet_rotation_uses_ccw_degrees_and_reduces_large_angles() {
        assert_eq!(sheet_rotation(-90.).unwrap(), [-1., 0.]);
        assert_eq!(sheet_rotation(720.).unwrap(), [0., 1.]);
        let [sin, cos] = sheet_rotation(390.).unwrap();
        assert!((sin - 0.5).abs() < 1e-14);
        assert!((cos - 3_f64.sqrt() / 2.).abs() < 1e-14);
        assert!(
            sheet_rotation(f64::MAX)
                .unwrap()
                .iter()
                .all(|value| value.is_finite())
        );
        assert!(sheet_rotation(f64::NAN).is_err());
    }

    #[test]
    fn standard_frames_follow_go_screen_right_and_up_axes() {
        let cases = [
            (BASE_VIEW_FRONT, [0., 1., 0.], [-1., 0., 0.]),
            (BASE_VIEW_BACK, [0., -1., 0.], [1., 0., 0.]),
            (BASE_VIEW_TOP, [0., 0., -1.], [-1., 0., 0.]),
            (BASE_VIEW_BOTTOM, [0., 0., 1.], [1., 0., 0.]),
            (BASE_VIEW_RIGHT, [-1., 0., 0.], [0., -1., 0.]),
            (BASE_VIEW_LEFT, [1., 0., 0.], [0., 1., 0.]),
            (BASE_VIEW_ISO, [-1., 1., -1.], [-1., -1., 0.]),
        ];
        for (orientation, normal, right) in cases {
            let frame = standard_frame(orientation, [1., 2., 3.]).unwrap();
            assert_eq!(
                (frame.normal, frame.x_direction, frame.origin),
                (normal.map(|value| -value), right, [1., 2., 3.])
            );
        }
        assert!(standard_frame(BaseViewOrientation(255), [0.; 3]).is_err());
    }
}
