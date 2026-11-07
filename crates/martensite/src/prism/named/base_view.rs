// SPDX-License-Identifier: GPL-2.0-only
//! Standard drawing views retain exact curves and map them to sheet coordinates.
use super::{DrawingProjection, DrawingProjectionFrame, NamedPrismSnapshot};
use oblikovati_api::types::*;
use oblikovati_model::{
    drawing::DrawingBaseViewDefinition, history::DocumentEditHistory, identity::DocumentGuid,
};

/// Exact projected curves and their authored sheet placement.
pub struct DrawingBaseProjection {
    pub curves: DrawingProjection,
    center: [f64; 2],
    scale: f64,
}
impl DrawingBaseProjection {
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
        let result = [
            self.center[0] + point[0] * self.scale * 10.,
            self.center[1] - point[1] * self.scale * 10.,
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
        let center = [view.center_x_mm, view.center_y_mm];
        if !view.scale.is_finite()
            || view.scale <= 0.
            || center.iter().any(|value| !value.is_finite())
        {
            return Err(format!(
                "drawing view scale {}, center {center:?}: expected positive finite scale and finite center",
                view.scale
            ));
        }
        let origin = super::super::provenance::representative(&self.snapshot.shape, "drawing view")
            .map_err(|cause| cause.to_string())?;
        let frame = standard_frame(view.orientation, origin)?;
        let curves = self.drawing_projection(history, document, key, frame)?;
        Ok(DrawingBaseProjection {
            curves,
            center,
            scale: view.scale,
        })
    }
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
