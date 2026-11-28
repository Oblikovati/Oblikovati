// SPDX-License-Identifier: GPL-2.0-only
//! Interactive live cross-section diagram for the Part Modify Slot Properties panel.
use crate::{GuiHoleSeatKind, GuiHoleTermination};
use imgui::Ui;
use oblikovati_model::feature::slot::SlotShape;

pub(super) struct SlotDiagramSpec<'a> {
    pub shape: SlotShape,
    pub seat: GuiHoleSeatKind,
    pub termination: GuiHoleTermination,
    pub end_mill_pointed: bool,
    pub length: &'a str,
    pub width: &'a str,
    pub depth: &'a str,
    pub seat_width: &'a str,
    pub seat_depth: &'a str,
    pub seat_angle: &'a str,
    pub end_mill_angle: &'a str,
    pub invalid: bool,
}

/// Draw the live Slot cross-section and plan preview diagram with dimension callouts.
pub(super) fn draw_slot_diagram(ui: &Ui, spec: &SlotDiagramSpec<'_>) {
    let width = ui.content_region_avail()[0].max(160.0);
    let height = 56.0;
    let origin = ui.cursor_screen_pos();
    ui.dummy([width, height]);
    let dl = ui.get_window_draw_list();

    let bg = [0.13, 0.14, 0.16, 1.0];
    let border = if spec.invalid {
        [0.86, 0.25, 0.25, 1.0]
    } else {
        [0.28, 0.30, 0.35, 1.0]
    };
    let outline = [0.82, 0.86, 0.92, 1.0];
    let dim = [0.46, 0.78, 0.96, 1.0];
    let p_max = [origin[0] + width, origin[1] + height];
    dl.add_rect(origin, p_max, bg)
        .filled(true)
        .rounding(3.0)
        .build();
    dl.add_rect(origin, p_max, border)
        .rounding(3.0)
        .thickness(1.0)
        .build();

    // Left: plan-view stadium/rectangle miniature
    let plan_x = origin[0] + 10.0;
    let plan_y = origin[1] + 16.0;
    let plan_w = 42.0;
    let plan_h = 18.0;
    let rounding = if spec.shape == SlotShape::Rounded {
        plan_h * 0.5
    } else {
        0.0
    };
    dl.add_rect(
        [plan_x, plan_y],
        [plan_x + plan_w, plan_y + plan_h],
        outline,
    )
    .rounding(rounding)
    .thickness(1.4)
    .build();
    dl.add_text([plan_x, origin[1] + 2.0], dim, spec.length);
    dl.add_text([plan_x, plan_y + plan_h + 2.0], dim, spec.width);

    // Right: cross-section profile with seat & end mill
    let cx = origin[0] + width * 0.64;
    let top_y = origin[1] + 16.0;
    let bot_y = origin[1] + height - 13.0;
    let bore_half = 12.0;
    let seat_half = 22.0;
    let seat_y = top_y + 11.0;

    dl.add_line(
        [cx - seat_half - 10.0, top_y],
        [cx - seat_half, top_y],
        outline,
    )
    .thickness(1.2)
    .build();
    dl.add_line([cx + seat_half, top_y], [p_max[0] - 8.0, top_y], outline)
        .thickness(1.2)
        .build();

    match spec.seat {
        GuiHoleSeatKind::None => {
            dl.add_line([cx - seat_half, top_y], [cx - bore_half, top_y], outline)
                .thickness(1.2)
                .build();
            dl.add_line([cx + bore_half, top_y], [cx + seat_half, top_y], outline)
                .thickness(1.2)
                .build();
            dl.add_line([cx - bore_half, top_y], [cx - bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, top_y], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
        }
        GuiHoleSeatKind::Counterbore | GuiHoleSeatKind::Spotface => {
            let sy = if spec.seat == GuiHoleSeatKind::Spotface {
                top_y + 9.0
            } else {
                seat_y
            };
            dl.add_line([cx - seat_half, top_y], [cx - seat_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + seat_half, top_y], [cx + seat_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx - seat_half, sy], [cx - bore_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, sy], [cx + seat_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx - bore_half, sy], [cx - bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, sy], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_text([cx - 18.0, origin[1] + 2.0], dim, spec.seat_width);
            dl.add_text([cx + seat_half + 4.0, top_y + 2.0], dim, spec.seat_depth);
        }
        GuiHoleSeatKind::Countersink => {
            dl.add_line([cx - seat_half, top_y], [cx - bore_half, seat_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + seat_half, top_y], [cx + bore_half, seat_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx - bore_half, seat_y], [cx - bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, seat_y], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_text([cx - 18.0, origin[1] + 2.0], dim, spec.seat_width);
            let angle_text = format_angle_callout(spec.seat_angle);
            dl.add_text([cx + seat_half + 4.0, top_y + 2.0], dim, &angle_text);
        }
    }

    if spec.termination == GuiHoleTermination::Distance {
        if spec.end_mill_pointed {
            let tip_y = bot_y + 8.0;
            dl.add_line([cx - bore_half, bot_y], [cx, tip_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, bot_y], [cx, tip_y], outline)
                .thickness(1.5)
                .build();
            let point_text = format_angle_callout(spec.end_mill_angle);
            dl.add_text([cx + bore_half + 5.0, bot_y - 4.0], dim, &point_text);
        } else {
            dl.add_line([cx - bore_half, bot_y], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
        }
        dl.add_text(
            [cx - seat_half - 24.0, (top_y + bot_y) * 0.5 - 5.0],
            dim,
            spec.depth,
        );
    } else {
        dl.add_line([cx - bore_half, bot_y], [cx - 3.0, bot_y], dim)
            .thickness(1.0)
            .build();
        dl.add_line([cx + 3.0, bot_y], [cx + bore_half, bot_y], dim)
            .thickness(1.0)
            .build();
    }
}

fn format_angle_callout(raw: &str) -> String {
    let trimmed = raw.trim();
    if trimmed.contains('\u{00b0}') {
        trimmed.to_owned()
    } else {
        let cleaned = trimmed.trim_end_matches("deg").trim();
        format!("{cleaned}\u{00b0}")
    }
}
