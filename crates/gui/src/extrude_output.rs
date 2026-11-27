// SPDX-License-Identifier: GPL-2.0-only
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
use crate::GuiExtrudeDirection;
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use imgui::Ui;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

const OPERATIONS: [(Option<PrismBoolean>, &str, IconGlyph); 4] = [
    (Some(PrismBoolean::Join), "Join", IconGlyph::ExtrudeJoin),
    (Some(PrismBoolean::Cut), "Cut", IconGlyph::ExtrudeCut),
    (
        Some(PrismBoolean::Intersect),
        "Intersect",
        IconGlyph::ExtrudeIntersect,
    ),
    (None, "New Solid", IconGlyph::ExtrudeNewSolid),
];

pub(crate) fn single_body(state: &GuiCommandState) -> bool {
    state
        .browser
        .as_ref()
        .is_some_and(|browser| browser.bodies.len() == 1)
}

impl ExtrudeDialog {
    pub(super) fn reference_direction(&self) -> GuiExtrudeDirection {
        if self.reference.through_all() {
            self.direction
        } else {
            GuiExtrudeDirection::Positive
        }
    }

    pub(super) fn reference_operation(&self) -> Option<PrismBoolean> {
        self.reference.through_all().then_some(PrismBoolean::Cut)
    }

    pub(super) fn update_distance_handle(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
    ) -> bool {
        if !self.reference.numeric() || self.edit_feature.is_some() || self.preview_hidden {
            self.handle = Default::default();
            return false;
        }
        let Some(distance) = self.handle.draw(ui, state, area) else {
            return false;
        };
        self.distance = distance;
        self.commit_error = None;
        true
    }

    pub(super) fn draw_advanced_settings(&mut self, ui: &Ui, icons: &GuiRibbonIcons) {
        const POPUP: &str = "##extrude-advanced-settings";
        ui.align_text_to_frame_padding();
        ui.text("Properties");
        let icon = icons.logical[IconSize::Small.index()];
        let label = "Advanced Settings";
        let width = icon[0] + ui.calc_text_size(label)[0] + 16.;
        ui.same_line_with_pos((ui.window_content_region_max()[0] - width).max(0.));
        let button = crate::widget_abi::IconButtonPresentation {
            texture: icons.key(IconGlyph::Preferences, IconSize::Small),
            size: [width, icon[1] + 6.],
            icon,
            large: false,
            show_caption: true,
        };
        if crate::widget_abi::icon_button(ui, label, button) {
            ui.open_popup(POPUP);
        }
        if let Some(_popup) = ui.begin_popup(POPUP)
            && ui
                .menu_item_config("Predict Boolean Operation")
                .selected(!self.manual_boolean)
                .build()
        {
            self.manual_boolean = !self.manual_boolean;
        }
    }

    pub(super) fn draw_output(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        icons: &GuiRibbonIcons,
    ) -> bool {
        if let Some(feature) = self.edit_feature {
            self.draw_edit_output(ui, state, feature);
            return false;
        }
        if !(self.reference.numeric() || self.reference.through_all()) {
            return self.operation.take().is_some();
        }
        let Some(browser) = &state.browser else {
            return false;
        };
        let previous = self.operation;
        let boolean_available = single_body(state);
        let through_all = self.reference.through_all();
        if through_all {
            self.operation = Some(PrismBoolean::Cut);
        } else if !boolean_available {
            self.operation = None;
        }
        if !ui.collapsing_header("Output", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return self.operation != previous;
        }
        self.draw_operation(ui, icons, boolean_available, through_all);
        if self.operation.is_none() {
            crate::theme::property_label(ui, "Body Name");
            ui.input_text("##extrude-body-name", &mut self.body_name)
                .hint(format!("Solid{}", browser.bodies.len() + 1))
                .build();
        }
        self.operation != previous
    }

    fn draw_operation(
        &mut self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        available: bool,
        through_all: bool,
    ) {
        let style = if available && !through_all {
            self.operation_style
                .draw_style_menu(ui, "Operation", "extrude-operation-style");
            self.operation_style
        } else {
            crate::theme::property_label(ui, "Operation");
            super::selector::SelectorStyle::Dropdown
        };
        let enabled = |operation| match operation {
            Some(operation) => available && (!through_all || operation == PrismBoolean::Cut),
            None => !through_all,
        };
        match style.choose(
            ui,
            icons,
            "##extrude-operation",
            self.operation,
            &OPERATIONS,
            enabled,
        ) {
            Some(Some(operation)) => self.choose_boolean_operation(operation),
            Some(None) => self.operation = None,
            None => {}
        }
    }

    fn draw_edit_output(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
    ) {
        let Some(body) = state
            .browser
            .as_ref()
            .and_then(|browser| browser.bodies.iter().find(|body| body.feature == feature))
        else {
            return;
        };
        if ui.collapsing_header("Output", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            crate::theme::property_label(ui, "Body Name");
            ui.text(&body.name);
            // A saved closed profile fixes the section for this distance edit.
            let _disabled = ui.begin_disabled(true);
            ui.checkbox("Match Shape", &mut true);
        }
    }

    fn choose_boolean_operation(&mut self, operation: PrismBoolean) {
        if !self.manual_boolean
            && matches!(operation, PrismBoolean::Cut | PrismBoolean::Intersect)
            && self.operation != Some(operation)
            && self.direction == GuiExtrudeDirection::Positive
        {
            self.direction = GuiExtrudeDirection::Negative;
        }
        self.operation = Some(operation);
    }

    pub(super) fn creation_action(&self, request: &crate::GuiExtrudePreview) -> GuiShellAction {
        let body_name = self.body_name.trim();
        let named_new_solid = request.operation.is_none()
            && request.reference_extent.is_none()
            && !body_name.is_empty();
        if request.reference_extent.is_some() || request.operation.is_some() || named_new_solid {
            return GuiShellAction::CommitExtrudeDraft(
                request.clone(),
                named_new_solid.then(|| body_name.to_owned()),
            );
        }
        GuiShellAction::Extrude {
            document: request.document,
            sketch: request.sketch,
            profile: request.profile,
            distance: request.distance.clone(),
            second_distance: request.second_distance.clone(),
            direction: request.direction,
        }
    }
}
