// SPDX-License-Identifier: GPL-2.0-only
//! Render 3D-projected User Coordinate System triads over the part viewport (#3918).
use super::{GuiUcsAction, GuiUcsStage, GuiUcsTriad};
use crate::{GuiFrameRequest, GuiShellAction};
use imgui::Ui;

const X_COLOR: [f32; 4] = [0.94, 0.27, 0.27, 1.0];
const Y_COLOR: [f32; 4] = [0.20, 0.84, 0.36, 1.0];
const Z_COLOR: [f32; 4] = [0.25, 0.56, 0.96, 1.0];
const ACTIVE_HANDLE_RING: [f32; 4] = [1.0, 0.88, 0.25, 1.0];
const ORIGIN_FILL: [f32; 4] = [0.98, 0.96, 0.85, 1.0];
const ORIGIN_STROKE: [f32; 4] = [0.12, 0.14, 0.18, 1.0];
const AXIS_THICKNESS: f32 = 2.5;
const ACTIVE_THICKNESS: f32 = 3.5;

/// Draw committed visible UCS triads and any active interactive placement triad.
pub(crate) fn draw_ucs_triads(
    ui: &Ui,
    request: &GuiFrameRequest,
    actions: &mut Vec<GuiShellAction>,
) {
    let ucs = &request.commands.ucs;
    if ucs.active.is_some()
        && !crate::widget_abi::any_popup_open(ui)
        && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
    {
        actions.push(GuiShellAction::Ucs(GuiUcsAction::Cancel));
    }
    if ucs.triads.is_empty()
        && ucs
            .active
            .as_ref()
            .and_then(|active| active.preview_triad.as_ref())
            .is_none()
    {
        return;
    }
    let scale = request.layout_scale();
    for triad in &ucs.triads {
        draw_single_triad(ui, scale, triad);
    }
    if let Some(preview) = ucs
        .active
        .as_ref()
        .and_then(|active| active.preview_triad.as_ref())
    {
        draw_single_triad(ui, scale, preview);
    }
}

fn draw_single_triad(ui: &Ui, scale: [f32; 2], triad: &GuiUcsTriad) {
    let Some(origin_px) = triad.origin_pixel else {
        return;
    };
    let logical = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let origin = logical(origin_px);
    let list = ui.get_background_draw_list();

    if let Some(x_px) = triad.x_pixel {
        let active = triad.stage == Some(GuiUcsStage::PickXDirection);
        draw_axis(&list, origin, logical(x_px), X_COLOR, "X", active);
    }
    if let Some(y_px) = triad.y_pixel {
        let active = triad.stage == Some(GuiUcsStage::PickYDirection);
        draw_axis(&list, origin, logical(y_px), Y_COLOR, "Y", active);
    }
    if let Some(z_px) = triad.z_pixel {
        draw_axis(&list, origin, logical(z_px), Z_COLOR, "Z", false);
    }

    list.add_circle(origin, 5.0, ORIGIN_FILL)
        .filled(true)
        .build();
    list.add_circle(origin, 5.0, ORIGIN_STROKE)
        .thickness(1.5)
        .build();
}

fn draw_axis(
    list: &imgui::DrawListMut<'_>,
    origin: [f32; 2],
    tip: [f32; 2],
    color: [f32; 4],
    label: &str,
    active_handle: bool,
) {
    let thickness = if active_handle {
        ACTIVE_THICKNESS
    } else {
        AXIS_THICKNESS
    };
    list.add_line(origin, tip, color)
        .thickness(thickness)
        .build();

    let dx = tip[0] - origin[0];
    let dy = tip[1] - origin[1];
    let len = dx.hypot(dy);
    if len > 6.0 {
        let ux = dx / len;
        let uy = dy / len;
        let px = -uy;
        let py = ux;
        let head = 9.0;
        let wing = 4.5;
        let base = [tip[0] - ux * head, tip[1] - uy * head];
        let left = [base[0] + px * wing, base[1] + py * wing];
        let right = [base[0] - px * wing, base[1] - py * wing];
        list.add_triangle(tip, left, right, color)
            .filled(true)
            .build();
        let label_pos = [tip[0] + ux * 8.0 - 4.0, tip[1] + uy * 8.0 - 6.0];
        list.add_text(label_pos, color, label);
    }
    if active_handle {
        list.add_circle(tip, 8.0, ACTIVE_HANDLE_RING)
            .thickness(2.0)
            .build();
    }
}
