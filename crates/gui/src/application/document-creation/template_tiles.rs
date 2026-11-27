// SPDX-License-Identifier: GPL-2.0-only
//! Measured icon tiles share the chooser's owned texture resources.
use super::{
    DetailsLayout, GuiCommandState, GuiTemplateKind, TEMPLATE_GROUPS, TemplateChoice,
    TemplateChooser, TemplateView,
};
use crate::icons::{GuiRibbonIcons, IconSize, document_glyph};
use imgui::{StyleColor, Ui};

impl TemplateView {
    pub(super) fn is_icon_view(self) -> bool {
        matches!(
            self,
            Self::SmallIcons | Self::MediumIcons | Self::ExtraLargeIcons
        )
    }
    pub(super) fn tile_size(self, factor: f32) -> [f32; 2] {
        let size = match self {
            Self::SmallIcons => [160., 30.],
            Self::MediumIcons => [160., 68.],
            Self::ExtraLargeIcons => [160., 94.],
            Self::List | Self::Details => [0., 0.],
        };
        size.map(|edge| edge * factor)
    }
}
impl TemplateChooser {
    pub(super) fn tile_width(
        &self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        entries: &[TemplateChoice],
    ) -> f32 {
        let base = self.view.tile_size(icons.layout_factor())[0];
        let padding = if self.view == TemplateView::SmallIcons {
            icons.logical[IconSize::Small.index()][0] + 7.
        } else {
            12.
        };
        entries
            .iter()
            .map(|entry| ui.calc_text_size(entry.title())[0] + padding)
            .fold(base, f32::max)
    }
    pub(super) fn gallery_width(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        icons: &GuiRibbonIcons,
        details: DetailsLayout,
    ) -> f32 {
        if self.view == TemplateView::Details {
            return details.width;
        }
        if !self.view.is_icon_view() {
            return 0.;
        }
        let gap = ui.clone_style().item_spacing[0];
        let groups = self.tile_groups();
        let widest = groups
            .iter()
            .map(|kinds| self.group_width(ui, icons, &self.catalog(state, kinds), gap))
            .fold(0., f32::max);
        widest + ui.clone_style().indent_spacing + gap
    }
    fn tile_groups(&self) -> Vec<Vec<GuiTemplateKind>> {
        if self.ungrouped {
            return vec![
                TEMPLATE_GROUPS
                    .iter()
                    .flat_map(|(_, kinds)| kinds.iter().copied())
                    .collect(),
            ];
        }
        TEMPLATE_GROUPS
            .iter()
            .map(|(_, kinds)| kinds.to_vec())
            .collect()
    }
    fn group_width(
        &self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        entries: &[TemplateChoice],
        gap: f32,
    ) -> f32 {
        self.tile_width(ui, icons, entries) * entries.len() as f32
            + gap * entries.len().saturating_sub(1) as f32
    }
    pub(super) fn icon_tile(
        &self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        template: &TemplateChoice,
        width: f32,
        selected: bool,
    ) -> bool {
        let color = ui.style_color(StyleColor::ButtonActive);
        let _selected = selected.then(|| ui.push_style_color(StyleColor::Button, color));
        let _selected_hover =
            selected.then(|| ui.push_style_color(StyleColor::ButtonHovered, color));
        let presentation = self.icon_presentation(icons, template.kind(), width);
        crate::widget_abi::icon_button(ui, &template.title(), presentation)
    }
    fn icon_presentation(
        &self,
        icons: &GuiRibbonIcons,
        kind: GuiTemplateKind,
        width: f32,
    ) -> crate::widget_abi::IconButtonPresentation {
        let large = self.view != TemplateView::SmallIcons;
        let style = if large {
            IconSize::Large
        } else {
            IconSize::Small
        };
        let mut icon = icons.logical[style.index()];
        if self.view == TemplateView::ExtraLargeIcons {
            icon = icon.map(|edge| edge * 1.4);
        }
        crate::widget_abi::IconButtonPresentation {
            texture: icons.key(document_glyph(kind), style),
            size: [width, self.view.tile_size(icons.layout_factor())[1]],
            icon,
            large,
            show_caption: true,
        }
    }
}
