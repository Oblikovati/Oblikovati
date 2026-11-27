// SPDX-License-Identifier: GPL-2.0-only
use super::selector::SelectorStyle;
use crate::GuiExtrudeDirection;
use crate::icons::{GuiRibbonIcons, IconGlyph};
use imgui::Ui;

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

impl SelectorStyle {
    pub(super) fn draw_direction(
        &mut self,
        ui: &Ui,
        direction: &mut GuiExtrudeDirection,
        read_only: bool,
        icons: &GuiRibbonIcons,
        asymmetric: bool,
    ) -> bool {
        self.draw_style_menu(ui, "Direction", "extrude-direction-style");
        let _disabled = ui.begin_disabled(read_only);
        let choices = if asymmetric {
            &DIRECTIONS[..]
        } else {
            &DIRECTIONS[..3]
        };
        let Some(selected) = self.choose(
            ui,
            icons,
            "##extrude-direction",
            *direction,
            choices,
            |_| true,
        ) else {
            return false;
        };
        let changed = selected != *direction;
        *direction = selected;
        changed
    }
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
    let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
    let edited = {
        let _numeric = crate::ui_scale::numeric_font(ui);
        ui.input_text(label, second).build()
    };
    ui.same_line();
    let flipped = crate::widget_abi::button(ui, flip, [0.; 2]);
    if flipped {
        std::mem::swap(first, second);
    }
    let _disabled = ui.begin_disabled(true);
    let _numeric = crate::ui_scale::numeric_font(ui);
    ui.input_text("Taper B", &mut String::from("0 deg")).build();
    edited || flipped
}
