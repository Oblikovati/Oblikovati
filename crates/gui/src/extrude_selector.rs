// SPDX-License-Identifier: GPL-2.0-only
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::widget_abi::{self, IconButtonPresentation};
use imgui::{StyleColor, Ui};

#[derive(Clone, Copy, Default, PartialEq, Eq)]
pub(super) enum SelectorStyle {
    #[default]
    Icons,
    Dropdown,
}

impl SelectorStyle {
    pub(super) fn draw_style_menu(&mut self, ui: &Ui, title: &str, popup: &str) {
        if widget_abi::button(ui, title, [0.; 2]) {
            ui.open_popup(popup);
        }
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

    pub(super) fn choose<T: Copy + Eq>(
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
    const CELLS: f32 = 4.;
    let spacing = 4.;
    let cell =
        ((ui.content_region_avail()[0] - spacing * (choices.len() - 1) as f32) / CELLS).max(1.);
    let show_caption = choices
        .iter()
        .all(|(_, label, _)| ui.calc_text_size(label)[0] + 2. <= cell);
    let icon = icons.logical[IconSize::Small.index()].map(|axis| axis.min(cell));
    let mut selected = None;
    for (index, (choice, label, glyph)) in choices.iter().copied().enumerate() {
        if index > 0 {
            ui.same_line_with_spacing(0., spacing);
        }
        let _disabled = ui.begin_disabled(!enabled(choice));
        let active = ui.style_color(StyleColor::ButtonActive);
        let _highlight = (choice == current)
            .then(|| widget_abi::push_style_color(ui, StyleColor::Button, active));
        let presentation = IconButtonPresentation {
            texture: icons.key(glyph, IconSize::Small),
            size: [cell, icon[1] + ui.text_line_height() + 6.],
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
