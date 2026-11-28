// SPDX-License-Identifier: GPL-2.0-only
//! Viewport manipulator gizmos for Direct Edit Move, Size, and Rotate modes (#3918).
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::{GuiCommandState, widget_abi};
use imgui::{Condition, Ui, WindowFlags};
use oblikovati_model::feature::direct_edit::DirectEditMode;

pub(crate) const MANIPULATOR_WINDOW: &str = "##direct-edit-manipulator";

const AXIS_X: [f32; 4] = [0.94, 0.27, 0.27, 0.95];
const AXIS_Y: [f32; 4] = [0.13, 0.77, 0.37, 0.95];
const AXIS_Z: [f32; 4] = [0.23, 0.51, 0.96, 0.95];
const SIZE_ACCENT: [f32; 4] = [0.18, 0.83, 0.75, 0.95];
const RING_BG: [f32; 4] = [0.08, 0.09, 0.12, 0.78];

pub(super) fn draw_manipulator(
    ui: &Ui,
    state: &GuiCommandState,
    area: WorkspaceArea,
    mode: DirectEditMode,
    has_selection: bool,
) {
    if !has_selection
        || !matches!(
            mode,
            DirectEditMode::Move | DirectEditMode::Size | DirectEditMode::Rotate
        )
    {
        return;
    }
    let scale = ui.io().display_framebuffer_scale;
    let anchor = state
        .direct_edit_preview
        .as_ref()
        .and_then(|preview| preview.anchor_pixels)
        .or_else(|| {
            state
                .direct_edit_pick
                .as_ref()
                .and_then(|pick| pick.anchor_pixels)
        })
        .map(|[x, y]| [x as f32 / scale[0], y as f32 / scale[1]])
        .unwrap_or([
            (area.left + area.right) * 0.5,
            (area.top + area.bottom) * 0.5,
        ]);
    let cx = anchor[0].clamp(area.left + 64.0, (area.right - 64.0).max(area.left + 64.0));
    let cy = anchor[1].clamp(area.top + 64.0, (area.bottom - 64.0).max(area.top + 64.0));
    let list = ui.get_foreground_draw_list();

    match mode {
        DirectEditMode::Move => draw_move_triad(&list, [cx, cy]),
        DirectEditMode::Size => draw_size_gizmo(&list, [cx, cy]),
        DirectEditMode::Rotate => draw_rotate_gizmo(&list, [cx, cy]),
        DirectEditMode::Scale | DirectEditMode::Delete => {}
    }

    let badge = match mode {
        DirectEditMode::Move => "Move Manipulator",
        DirectEditMode::Size => "Size Manipulator",
        DirectEditMode::Rotate => "Rotate Manipulator",
        _ => return,
    };
    let pos = [cx + 12.0, cy - 38.0];
    let _bg = widget_abi::push_style_color(ui, imgui::StyleColor::WindowBg, RING_BG);
    widget_abi::window_position_with_pivot(ui, pos, [0.0, 0.0], Condition::Always);
    ui.window(MANIPULATOR_WINDOW)
        .flags(
            WindowFlags::NO_DECORATION
                | WindowFlags::ALWAYS_AUTO_RESIZE
                | WindowFlags::NO_SAVED_SETTINGS
                | WindowFlags::NO_DOCKING
                | WindowFlags::NO_NAV
                | WindowFlags::NO_FOCUS_ON_APPEARING
                | WindowFlags::NO_INPUTS,
        )
        .build(|| {
            ui.text_colored(SIZE_ACCENT, badge);
        });
}

fn draw_move_triad(list: &imgui::DrawListMut<'_>, [cx, cy]: [f32; 2]) {
    let len = 34.0;
    list.add_circle([cx, cy], 5.0, [1.0, 1.0, 1.0, 0.9])
        .filled(true)
        .build();
    // +X arrow
    list.add_line([cx, cy], [cx + len, cy], AXIS_X)
        .thickness(2.4)
        .build();
    list.add_triangle(
        [cx + len + 7.0, cy],
        [cx + len - 2.0, cy - 4.0],
        [cx + len - 2.0, cy + 4.0],
        AXIS_X,
    )
    .filled(true)
    .build();
    // +Y arrow (up)
    list.add_line([cx, cy], [cx, cy - len], AXIS_Y)
        .thickness(2.4)
        .build();
    list.add_triangle(
        [cx, cy - len - 7.0],
        [cx - 4.0, cy - len + 2.0],
        [cx + 4.0, cy - len + 2.0],
        AXIS_Y,
    )
    .filled(true)
    .build();
    // +Z arrow (diagonal)
    let zx = cx - len * 0.65;
    let zy = cy + len * 0.65;
    list.add_line([cx, cy], [zx, zy], AXIS_Z)
        .thickness(2.4)
        .build();
    list.add_circle([zx, zy], 3.5, AXIS_Z).filled(true).build();
}

fn draw_size_gizmo(list: &imgui::DrawListMut<'_>, [cx, cy]: [f32; 2]) {
    list.add_circle([cx, cy], 22.0, SIZE_ACCENT)
        .thickness(2.2)
        .build();
    list.add_line([cx - 32.0, cy], [cx + 32.0, cy], SIZE_ACCENT)
        .thickness(2.2)
        .build();
    list.add_triangle(
        [cx + 38.0, cy],
        [cx + 29.0, cy - 5.0],
        [cx + 29.0, cy + 5.0],
        SIZE_ACCENT,
    )
    .filled(true)
    .build();
    list.add_triangle(
        [cx - 38.0, cy],
        [cx - 29.0, cy - 5.0],
        [cx - 29.0, cy + 5.0],
        SIZE_ACCENT,
    )
    .filled(true)
    .build();
}

fn draw_rotate_gizmo(list: &imgui::DrawListMut<'_>, [cx, cy]: [f32; 2]) {
    list.add_circle([cx, cy], 28.0, AXIS_Z)
        .thickness(2.2)
        .build();
    list.add_circle([cx, cy], 20.0, AXIS_Y)
        .thickness(1.8)
        .build();
    list.add_circle([cx, cy], 4.5, AXIS_X).filled(true).build();
}
