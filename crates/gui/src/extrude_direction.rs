// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiExtrudeDirection;
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::widget_abi::{self, IconButtonPresentation};
use imgui::{StyleColor, Ui};

const DIRECTIONS: [(GuiExtrudeDirection, &str, IconGlyph); 4] = [
    (
        GuiExtrudeDirection::Positive,
        "Default",
        IconGlyph::DirectionDefault,
    ),
    (
        GuiExtrudeDirection::Negative,
        "Flipped",
        IconGlyph::DirectionFlipped,
    ),
    (
        GuiExtrudeDirection::Symmetric,
        "Symmetric",
        IconGlyph::DirectionSymmetric,
    ),
    (
        GuiExtrudeDirection::Asymmetric,
        "Asymmetric",
        IconGlyph::DirectionAsymmetric,
    ),
];

#[derive(Clone, Copy, Default, PartialEq, Eq)]
pub(super) enum SelectorStyle {
    #[default]
    Icons,
    Dropdown,
}

impl SelectorStyle {
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        direction: &mut GuiExtrudeDirection,
        read_only: bool,
        icons: &GuiRibbonIcons,
    ) -> bool {
        self.draw_style_menu(ui);
        let _disabled = ui.begin_disabled(read_only);
        let previous = *direction;
        let selected = match self {
            Self::Icons => draw_icons(ui, previous, icons),
            Self::Dropdown => draw_dropdown(ui, previous),
        };
        *direction = selected.unwrap_or(previous);
        *direction != previous
    }

    fn draw_style_menu(&mut self, ui: &Ui) {
        const POPUP: &str = "extrude-direction-style";
        if ui.button("Direction") {
            ui.open_popup(POPUP);
        }
        let Some(_popup) = ui.begin_popup(POPUP) else {
            return;
        };
        ui.text("Selector Style");
        for (style, label) in [(Self::Icons, "Icons"), (Self::Dropdown, "Dropdown")] {
            if ui.menu_item_config(label).selected(*self == style).build() {
                *self = style;
            }
        }
    }
}

fn draw_icons(
    ui: &Ui,
    direction: GuiExtrudeDirection,
    icons: &GuiRibbonIcons,
) -> Option<GuiExtrudeDirection> {
    let mut selected = None;
    let width = ui.content_region_avail()[0];
    let spacing = ui.clone_style().item_spacing[0];
    let icon = icons.logical[IconSize::Small.index()];
    let mut used = 0.;
    for (choice, label, glyph) in DIRECTIONS {
        let size = [ui.calc_text_size(label)[0] + icon[0] + 12., icon[1] + 6.];
        if used > 0. && used + spacing + size[0] <= width {
            ui.same_line();
            used += spacing;
        } else {
            used = 0.;
        }
        used += size[0];
        if draw_direction_icon(
            ui,
            label,
            choice == direction,
            IconButtonPresentation {
                texture: icons.key(glyph, IconSize::Small),
                size,
                icon,
                large: false,
                show_caption: true,
            },
        ) {
            selected = Some(choice);
        }
    }
    selected
}

fn draw_direction_icon(ui: &Ui, label: &str, selected: bool, icon: IconButtonPresentation) -> bool {
    let active = ui.style_color(StyleColor::ButtonActive);
    let _highlight = selected.then(|| ui.push_style_color(StyleColor::Button, active));
    widget_abi::icon_button(ui, label, icon)
}

fn draw_dropdown(ui: &Ui, direction: GuiExtrudeDirection) -> Option<GuiExtrudeDirection> {
    let (_, label, _) = DIRECTIONS
        .iter()
        .find(|(choice, _, _)| *choice == direction)?;
    let _combo = ui.begin_combo("##extrude-direction", label)?;
    DIRECTIONS.iter().find_map(|&(choice, label, _)| {
        ui.selectable_config(label)
            .selected(choice == direction)
            .build()
            .then_some(choice)
    })
}

pub(super) fn draw_second_distance(ui: &Ui, first: &mut String, second: &mut String) -> bool {
    let style = ui.clone_style();
    let label = "Distance B";
    let flip = "Flip A/B";
    let reserved = ui.calc_text_size(label)[0]
        + ui.calc_text_size(flip)[0]
        + style.item_inner_spacing[0]
        + style.item_spacing[0]
        + 2. * style.frame_padding[0];
    ui.set_next_item_width((ui.content_region_avail()[0] - reserved).max(1.));
    let edited = ui.input_text(label, second).build();
    ui.same_line();
    let flipped = crate::widget_abi::button(ui, flip, [0.; 2]);
    if flipped {
        std::mem::swap(first, second);
    }
    let _disabled = ui.begin_disabled(true);
    ui.input_text("Taper B", &mut String::from("0 deg")).build();
    edited || flipped
}
