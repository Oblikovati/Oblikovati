// SPDX-License-Identifier: GPL-2.0-only
//! The cube's faces as they fall on the screen, from the view's axes.
use super::*;
/// A cube face as it shows on screen.
pub(super) struct ShownFace {
    pub(super) face: GuiCubeFace,
    pub(super) facing: f32,
    pub(super) corners: [[f32; 2]; 4],
}
pub(super) fn shown_faces(
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
    origin: [f32; 2],
) -> Vec<ShownFace> {
    let mut faces: Vec<ShownFace> = GuiCubeFace::ALL
        .into_iter()
        .map(|face| ShownFace {
            face,
            facing: dot(face.normal(), axes.toward),
            corners: face.corners().map(|corner| {
                [
                    origin[0] + layout.center[0] + dot(corner, axes.right) * layout.half,
                    origin[1] + layout.center[1] - dot(corner, axes.up) * layout.half,
                ]
            }),
        })
        .filter(|shown| shown.facing > FACING_VIEWER)
        .collect();
    faces.sort_by(|first, second| second.facing.total_cmp(&first.facing));
    faces
}
pub(super) fn inside([x, y]: [f32; 2], corners: &[[f32; 2]; 4]) -> bool {
    let side = |index: usize| {
        let [ax, ay] = corners[index];
        let [bx, by] = corners[(index + 1) % 4];
        (bx - ax) * (y - ay) - (by - ay) * (x - ax)
    };
    let signs = [side(0), side(1), side(2), side(3)];
    signs.iter().all(|value| *value >= 0.) || signs.iter().all(|value| *value <= 0.)
}
