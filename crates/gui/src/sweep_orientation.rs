// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiSweepOrientation,
    icons::{GuiRibbonIcons, IconGlyph, IconSize},
    widget_abi::{self, IconButtonPresentation},
};
use imgui::{StyleColor, Ui};

const OPTIONS: [(GuiSweepOrientation, &str, IconGlyph); 3] = [
    (
        GuiSweepOrientation::FollowPath,
        "Follow Path",
        IconGlyph::Sweep,
    ),
    (
        GuiSweepOrientation::Fixed,
        "Fixed",
        IconGlyph::DirectionDefault,
    ),
    (GuiSweepOrientation::Guide, "Guide", IconGlyph::WorkAxis),
];

#[derive(Default)]
pub(super) struct OrientationSelector {
    dropdown: bool,
}

impl OrientationSelector {
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        orientation: &mut GuiSweepOrientation,
        icons: &GuiRibbonIcons,
    ) {
        if ui.button("Orientation") {
            ui.open_popup("sweep-orientation-style");
        }
        if let Some(_menu) = ui.begin_popup("sweep-orientation-style") {
            ui.text("Selector Style");
            for (dropdown, title) in [(false, "Icons"), (true, "Dropdown")] {
                if ui
                    .menu_item_config(title)
                    .selected(self.dropdown == dropdown)
                    .build()
                {
                    self.dropdown = dropdown;
                }
            }
        }
        if self.dropdown {
            let label = OPTIONS
                .iter()
                .find(|(option, _, _)| option == orientation)
                .unwrap()
                .1;
            if let Some(_menu) = ui.begin_combo("##sweep-orientation", label) {
                for (option, label, _) in OPTIONS {
                    if ui
                        .selectable_config(label)
                        .selected(option == *orientation)
                        .build()
                    {
                        *orientation = option;
                    }
                }
            }
        } else {
            let cell = ((ui.content_region_avail()[0] - 8.) / 3.).max(1.);
            let icon = icons.logical[IconSize::Small.index()].map(|extent| extent.min(cell));
            for (index, (option, label, glyph)) in OPTIONS.into_iter().enumerate() {
                if index != 0 {
                    ui.same_line_with_spacing(0., 4.);
                }
                let active = ui.style_color(StyleColor::ButtonActive);
                let _selected = (option == *orientation)
                    .then(|| ui.push_style_color(StyleColor::Button, active));
                if widget_abi::icon_button(
                    ui,
                    label,
                    IconButtonPresentation {
                        texture: icons.key(glyph, IconSize::Small),
                        size: [cell, (icon[1] + 8.).max(29.)],
                        icon,
                        large: false,
                        show_caption: true,
                    },
                ) {
                    *orientation = option;
                }
            }
        }
    }
}
