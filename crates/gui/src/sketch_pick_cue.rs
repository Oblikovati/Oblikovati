// SPDX-License-Identifier: GPL-2.0-only
//! Noninteractive identification for a checked sketch pick (#4042).
use crate::GuiFrameRequest;

/// A type cue anchored to the sketch plane at the picked position.
/// Example: use `pixel` after projecting `position` on `plane` for the current view.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSketchPickCue {
    pub pixel: [i32; 2],
    pub text: String,
    pub plane: oblikovati_model::sketch::SketchPlane,
    pub position: [f64; 2],
}

pub(crate) fn draw_sketch_pick_cue(ui: &imgui::Ui, request: &GuiFrameRequest) {
    let Some(cue) = request.commands.sketch_pick_cue.as_ref() else {
        return;
    };
    if !request.commands.sketch_at_ready {
        return;
    }
    let scale = request.layout_scale();
    let mouse = ui.io().mouse_pos;
    let anchor = [
        cue.pixel[0] as f32 / scale[0],
        cue.pixel[1] as f32 / scale[1],
    ];
    let size = ui.calc_text_size(&cue.text);
    let Some(position) = cue_position(anchor, mouse, size, ui.io().display_size, scale) else {
        return;
    };
    draw_cue(ui, position, size, &cue.text);
}

fn draw_cue(ui: &imgui::Ui, position: [f32; 2], size: [f32; 2], text: &str) {
    let end = [position[0] + size[0] + 12., position[1] + size[1] + 8.];
    let list = ui.get_background_draw_list();
    list.add_rect(position, end, ui.style_color(imgui::StyleColor::PopupBg))
        .filled(true)
        .build();
    list.add_text(
        [position[0] + 6., position[1] + 4.],
        ui.style_color(imgui::StyleColor::Text),
        text,
    );
}

fn cue_position(
    anchor: [f32; 2],
    mouse: [f32; 2],
    text: [f32; 2],
    display: [f32; 2],
    scale: [f32; 2],
) -> Option<[f32; 2]> {
    if !valid_cue_position(anchor, mouse, text, display, scale) {
        return None;
    }
    let size = [text[0] + 12., text[1] + 8.];
    Some([
        (anchor[0] + 12.).clamp(0., display[0] - size[0]),
        (anchor[1] + 12.).clamp(0., display[1] - size[1]),
    ])
}

fn valid_cue_position(
    anchor: [f32; 2],
    mouse: [f32; 2],
    text: [f32; 2],
    display: [f32; 2],
    scale: [f32; 2],
) -> bool {
    if !anchor
        .into_iter()
        .chain(mouse)
        .chain(text)
        .chain(display)
        .chain(scale)
        .all(f32::is_finite)
    {
        return false;
    }
    if scale.into_iter().any(|value| value <= 0.) {
        return false;
    }
    if (0..2).any(|axis| anchor[axis] < 0. || anchor[axis] >= display[axis]) {
        return false;
    }
    if ((anchor[0] - mouse[0]) * scale[0]).hypot((anchor[1] - mouse[1]) * scale[1]) > 12. {
        return false;
    }
    let size = [text[0] + 12., text[1] + 8.];
    display[0] >= size[0] && display[1] >= size[1]
}

#[cfg(test)]
mod tests {
    use super::cue_position;
    #[test]
    fn cue_stays_on_screen_and_hides_when_pointer_leaves() {
        assert_eq!(
            cue_position([98., 98.], [98., 98.], [20., 10.], [100., 100.], [1., 1.]),
            Some([68., 82.])
        );
        assert_eq!(
            cue_position([50., 50.], [70., 50.], [20., 10.], [100., 100.], [1., 1.]),
            None
        );
        assert_eq!(
            cue_position([50., 50.], [50., 50.], [100., 10.], [100., 100.], [1., 1.]),
            None
        );
        assert_eq!(
            cue_position(
                [f32::NAN, 50.],
                [50., 50.],
                [20., 10.],
                [100., 100.],
                [1., 1.]
            ),
            None
        );
        assert_eq!(
            cue_position([50., 50.], [57., 50.], [20., 10.], [100., 100.], [2., 2.]),
            None
        );
    }
}
