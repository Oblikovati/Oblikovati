// SPDX-License-Identifier: GPL-2.0-only
//! Shared Icons/Dropdown selector control for Feature Properties groups.
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::widget_abi::{self, IconButtonPresentation};
use imgui::{StyleColor, Ui};

#[derive(Clone, Copy, Default, PartialEq, Eq)]
pub(crate) enum SelectorStyle {
    #[default]
    Icons,
    Dropdown,
}

impl SelectorStyle {
    pub(crate) fn draw_style_menu(&mut self, ui: &Ui, title: &str, popup: &str) {
        let _bg = ui.push_style_color(StyleColor::Button, [0.; 4]);
        let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(0.));
        let _pad = ui.push_style_var(imgui::StyleVar::FramePadding([2., 2.]));
        let _ink = ui.push_style_color(StyleColor::Text, crate::theme::text_secondary(ui));
        let label = format!("{title}##{popup}");
        if widget_abi::button(ui, &label, [0.; 2]) {
            ui.open_popup(popup);
        }
        let [right, top] = [ui.item_rect_max()[0], ui.item_rect_min()[1]];
        let bottom = ui.item_rect_max()[1];
        let mid_y = ((top + bottom) * 0.5).floor();
        let x = right + 4.;
        ui.get_window_draw_list()
            .add_triangle(
                [x, mid_y - 1.],
                [x + 6., mid_y - 1.],
                [x + 3., mid_y + 3.],
                crate::theme::text_muted(ui),
            )
            .filled(true)
            .build();
        let Some(_popup) = ui.begin_popup(popup) else {
            return;
        };
        ui.text("Selector Style");
        for (style, label) in [(Self::Icons, "Icons"), (Self::Dropdown, "Dropdown")] {
            if ui.menu_item_config(label).selected(*self == style).build() {
                *self = style;
            }
        }
    }

    pub(crate) fn choose<T: Copy + Eq>(
        self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        dropdown: &str,
        current: T,
        choices: &[(T, &str, IconGlyph)],
        enabled: impl Fn(T) -> bool,
    ) -> Option<T> {
        match self {
            Self::Icons => draw_icons(ui, icons, current, choices, enabled),
            Self::Dropdown => draw_dropdown(ui, dropdown, current, choices, enabled),
        }
    }
}

fn draw_icons<T: Copy + Eq>(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    current: T,
    choices: &[(T, &str, IconGlyph)],
    enabled: impl Fn(T) -> bool,
) -> Option<T> {
    let spacing = 4.;
    let count = choices.len().max(1) as f32;
    let cell = ((ui.content_region_avail()[0] - spacing * (count - 1.)) / count).max(1.);
    let show_caption = choices
        .iter()
        .all(|(_, label, _)| ui.calc_text_size(label)[0] + 12. <= cell);
    let icon = icons.logical[IconSize::Small.index()].map(|axis| axis.min(cell));
    let mut selected = None;
    for (index, (choice, label, glyph)) in choices.iter().copied().enumerate() {
        if index > 0 {
            ui.same_line_with_spacing(0., spacing);
        }
        let _disabled = ui.begin_disabled(!enabled(choice));
        let is_active = choice == current;
        let fill = if is_active {
            crate::theme::brand_subtle(ui)
        } else {
            crate::theme::surface_subtle(ui)
        };
        let border = if is_active {
            crate::theme::brand_accent(ui)
        } else {
            ui.style_color(StyleColor::Border)
        };
        let _fill = widget_abi::push_style_color(ui, StyleColor::Button, fill);
        let _edge = widget_abi::push_style_color(ui, StyleColor::Border, border);
        let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
        let presentation = IconButtonPresentation {
            texture: icons.key(glyph, IconSize::Small),
            size: [cell, icon[1] + ui.text_line_height() + 10.],
            icon,
            large: true,
            show_caption,
        };
        if widget_abi::icon_button(ui, label, presentation) {
            selected = Some(choice);
        }
        if !show_caption && ui.is_item_hovered() {
            ui.tooltip_text(label);
        }
    }
    selected
}

fn draw_dropdown<T: Copy + Eq>(
    ui: &Ui,
    id: &str,
    current: T,
    choices: &[(T, &str, IconGlyph)],
    enabled: impl Fn(T) -> bool,
) -> Option<T> {
    let (_, label, _) = choices.iter().find(|(choice, _, _)| *choice == current)?;
    let _combo = ui.begin_combo(id, label)?;
    choices.iter().find_map(|&(choice, label, _)| {
        ui.selectable_config(label)
            .selected(choice == current)
            .disabled(!enabled(choice))
            .build()
            .then_some(choice)
    })
}
