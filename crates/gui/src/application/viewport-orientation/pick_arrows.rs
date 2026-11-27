// SPDX-License-Identifier: GPL-2.0-only
//! Hit testing for face-on orientation arrows.
use super::*;

pub(super) fn near(point: [f32; 2], center: [f32; 2]) -> bool {
    (point[0] - center[0]).hypot(point[1] - center[1]) <= ARROW_REACH
}
pub(super) fn is_face_on(axes: &GuiViewAxes) -> bool {
    GuiCubeFace::ALL
        .iter()
        .any(|face| dot(face.normal(), axes.toward) > FACE_ON)
}
/// The arrows show only face-on: an oblique view has no neighbour to step to.
pub(super) fn arrow_at(
    point: [f32; 2],
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
) -> Option<GuiOrientCommand> {
    use GuiOrientCommand::{Roll, Step};
    if !is_face_on(axes) {
        return None;
    }
    [
        (layout.step_right, Step(GuiViewStep::Right)),
        (layout.step_left, Step(GuiViewStep::Left)),
        (layout.step_up, Step(GuiViewStep::Up)),
        (layout.step_down, Step(GuiViewStep::Down)),
        (layout.roll_upper, Roll { clockwise: true }),
        (layout.roll_lower, Roll { clockwise: false }),
    ]
    .into_iter()
    .find(|(center, _)| near(point, *center))
    .map(|(_, command)| command)
}
