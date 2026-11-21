// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiFrameRequest;
use imgui::Ui;

pub(super) const ZOOM_LABEL: &str = "Zoom Selected";
const PICK_GAP: f32 = 8.;

/// Place the mini toolbar near a projected face pick. Example: `near_pick(ui, request)`.
pub(super) fn near_pick(ui: &Ui, request: &GuiFrameRequest) -> Option<([f32; 2], [f32; 2])> {
    let point = request.commands.face_toolbar_anchor.as_ref()?.pixel;
    let scale = request.layout_scale();
    let anchor = std::array::from_fn(|axis| point[axis] as f32 / scale[axis]);
    let display = ui.io().display_size;
    if !(0..2).all(|axis| (0. ..display[axis]).contains(&anchor[axis])) {
        return None;
    }
    let style = ui.clone_style();
    let toolbar = [
        ui.calc_text_size(ZOOM_LABEL)[0] + 2. * (style.window_padding[0] + style.frame_padding[0]),
        ui.frame_height() + 2. * style.window_padding[1],
    ];
    Some(placement(anchor, display, toolbar))
}

fn placement(anchor: [f32; 2], display: [f32; 2], size: [f32; 2]) -> ([f32; 2], [f32; 2]) {
    let left = anchor[0] + PICK_GAP + size[0] > display[0];
    let above = anchor[1] - PICK_GAP >= size[1];
    let pivot = [if left { 1. } else { 0. }, if above { 1. } else { 0. }];
    let point = std::array::from_fn(|axis| anchor[axis] + PICK_GAP * (1. - 2. * pivot[axis]));
    (point, pivot)
}

#[cfg(test)]
mod tests {
    use super::placement;

    #[test]
    fn the_face_toolbar_flips_at_the_top_and_right_display_edges() {
        let extent = [600., 400.];
        let size = [150., 30.];
        assert_eq!(
            placement([250., 200.], extent, size),
            ([258., 192.], [0., 1.])
        );
        assert_eq!(placement([595., 4.], extent, size), ([587., 12.], [1., 0.]));
    }
}
