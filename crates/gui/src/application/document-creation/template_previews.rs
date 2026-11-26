// SPDX-License-Identifier: GPL-2.0-only
//! Draw chooser sheet thumbnails from authored page settings and layout positions.
use super::workspaces::{LINE_COLOR as INK, SHEET_COLOR as PAPER};
use super::{DrawingLayout, TemplateChoice};
use imgui::{DrawListMut, SelectableFlags, StyleColor, Ui};
use oblikovati_api::types::{
    BOTTOM_LEFT_TITLE_BLOCK, TOP_LEFT_TITLE_BLOCK, TOP_RIGHT_TITLE_BLOCK, TitleBlockLocation,
};
use oblikovati_model::drawing::{DrawingAnnotationDefinition, DrawingSheetRecipe};
use std::sync::LazyLock;

/// Settings and authored positions needed for a drawing template thumbnail.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiTemplateSheet {
    dimensions: [u32; 2],
    border: bool,
    title_block: Option<TitleBlockLocation>,
    views: Vec<[u16; 2]>,
    tables: Vec<[u16; 2]>,
}

impl GuiTemplateSheet {
    /// Capture thumbnail data from a validated drawing sheet.
    /// Example: `GuiTemplateSheet::from_recipe(&drawing.sheets()[drawing.active_sheet()])`.
    pub fn from_recipe(sheet: &DrawingSheetRecipe) -> Option<Self> {
        let dimensions = sheet.dimensions_mm().ok()?;
        let anchor = |x, y| {
            [x / dimensions[0], y / dimensions[1]]
                .map(|part: f64| (part.clamp(0., 1.) * 1000.).round() as u16)
        };
        Some(Self {
            dimensions: dimensions.map(|side| ((side * 1000.).round() as u32).max(1)),
            border: sheet.border,
            title_block: (!sheet.title_block.is_empty()).then_some(sheet.title_block_at),
            views: sheet
                .views
                .iter()
                .map(|view| anchor(view.center_x_mm, view.center_y_mm))
                .collect(),
            tables: sheet
                .annotations
                .iter()
                .filter_map(|annotation| match annotation {
                    DrawingAnnotationDefinition::PartsList { x_mm, y_mm, .. }
                    | DrawingAnnotationDefinition::CustomTable { x_mm, y_mm, .. }
                    | DrawingAnnotationDefinition::RevisionTable { x_mm, y_mm, .. } => {
                        Some(anchor(*x_mm, *y_mm))
                    }
                    DrawingAnnotationDefinition::Note { .. }
                    | DrawingAnnotationDefinition::RevisionTag { .. }
                    | DrawingAnnotationDefinition::RevisionCloud { .. } => None,
                })
                .collect(),
        })
    }
}

static DEFAULT_SHEET: LazyLock<GuiTemplateSheet> = LazyLock::new(|| {
    GuiTemplateSheet::from_recipe(&DrawingSheetRecipe::a3_landscape("Sheet:1"))
        .expect("default template has valid sheet dimensions")
});

pub(super) fn show(ui: &Ui, template: &TemplateChoice, selection: &mut Option<DrawingLayout>) {
    let sheet = match template {
        TemplateChoice::Project(template) => {
            template.sheet_preview.as_deref().unwrap_or(&DEFAULT_SHEET)
        }
        TemplateChoice::BuiltIn(_) => &DEFAULT_SHEET,
    };
    for (label, layout) in [
        ("Use Template", DrawingLayout::Template),
        ("Assembly Layout", DrawingLayout::Assembly),
    ] {
        let selected = *selection == Some(layout);
        let _highlight = selected.then(|| {
            ui.push_style_color(StyleColor::Header, ui.style_color(StyleColor::HeaderActive))
        });
        if ui
            .selectable_config(label)
            .flags(SelectableFlags::DONT_CLOSE_POPUPS)
            .size([0., ui.text_line_height() * 4.5])
            .selected(selected)
            .build()
        {
            *selection = Some(layout);
        }
        draw_thumbnail(ui, sheet, label, layout == DrawingLayout::Assembly);
    }
}

fn draw_thumbnail(ui: &Ui, sheet: &GuiTemplateSheet, label: &str, populated: bool) {
    let low = ui.item_rect_min();
    let high = ui.item_rect_max();
    let text_width = ui.calc_text_size(label)[0];
    let height = (high[1] - low[1] - 10.).max(1.);
    let width = (high[0] - low[0] - text_width - 22.).min(height * 1.6);
    if width < 16. {
        return;
    }
    let ratio = sheet.dimensions[0] as f32 / sheet.dimensions[1] as f32;
    let width = width.min(height * ratio);
    let height = width / ratio;
    let minimum = [high[0] - 8. - width, (low[1] + high[1] - height) * 0.5];
    let maximum = [minimum[0] + width, minimum[1] + height];
    let list = ui.get_window_draw_list();
    list.add_rect(minimum, maximum, PAPER).filled(true).build();
    let inset = (width.min(height) * 0.14).min(4.);
    let inside = [minimum[0] + inset, minimum[1] + inset];
    let end = [maximum[0] - inset, maximum[1] - inset];
    if sheet.border {
        list.add_rect(inside, end, INK).build();
    }
    if let Some(location) = &sheet.title_block {
        title_block(&list, inside, end, location);
    }
    if populated {
        layout_marks(&list, sheet, inside, end);
    }
}

fn title_block(
    list: &DrawListMut<'_>,
    low: [f32; 2],
    high: [f32; 2],
    location: &TitleBlockLocation,
) {
    let size = [(high[0] - low[0]) * 0.32, (high[1] - low[1]) * 0.16];
    let x = if matches!(*location, BOTTOM_LEFT_TITLE_BLOCK | TOP_LEFT_TITLE_BLOCK) {
        low[0]
    } else {
        high[0] - size[0]
    };
    let y = if matches!(*location, TOP_LEFT_TITLE_BLOCK | TOP_RIGHT_TITLE_BLOCK) {
        low[1]
    } else {
        high[1] - size[1]
    };
    list.add_rect([x, y], [x + size[0], y + size[1]], INK)
        .build();
}

fn layout_marks(list: &DrawListMut<'_>, sheet: &GuiTemplateSheet, low: [f32; 2], high: [f32; 2]) {
    let dx = ((high[0] - low[0]) * 0.19).min(6.);
    let dy = ((high[1] - low[1]) * 0.19).min(4.);
    let locate = |point: [u16; 2]| {
        let fraction = |value| (f32::from(value) / 1000.).clamp(0.19, 0.81);
        [
            low[0] + (high[0] - low[0]) * fraction(point[0]),
            high[1] - (high[1] - low[1]) * fraction(point[1]),
        ]
    };
    let views = if sheet.views.is_empty() {
        &[[280, 420]][..]
    } else {
        &sheet.views
    };
    let tables = if sheet.tables.is_empty() {
        &[[680, 650]][..]
    } else {
        &sheet.tables
    };
    for &point in views {
        let [x, y] = locate(point);
        let half_width = dx.min(5.);
        list.add_rect([x - half_width, y - dy], [x + half_width, y + dy], INK)
            .build();
    }
    for &point in tables {
        let [x, y] = locate(point);
        list.add_rect([x - dx, y - dy], [x + dx, y + dy], INK)
            .build();
        list.add_line([x - dx, y], [x + dx, y], INK).build();
    }
}
