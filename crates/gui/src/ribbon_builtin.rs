// SPDX-License-Identifier: GPL-2.0-only
//! Built-in commands use one action in both ribbon locations.
use super::{RibbonPanel, RibbonPanelOwners};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
#[path = "ribbon_plane_group.rs"]
mod plane_group;

#[derive(Clone, Copy)]
enum Builtin {
    Copy,
    Measure,
    Plane,
}
fn builtin_for(panel: RibbonPanel) -> Option<Builtin> {
    match panel {
        RibbonPanel::ToolClipboard => Some(Builtin::Copy),
        RibbonPanel::ToolMeasure => Some(Builtin::Measure),
        RibbonPanel::PartWorkFeatures => Some(Builtin::Plane),
        _ => None,
    }
}
pub(super) fn supports(panel: RibbonPanel) -> bool {
    builtin_for(panel).is_some()
}
pub(super) fn is_expanded(
    panel: RibbonPanel,
    state: &GuiCommandState,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> bool {
    builtin_for(panel)
        .is_some_and(|command| customization.builtin_expanded(&command_id(state, command)))
}
fn command_id(state: &GuiCommandState, command: Builtin) -> String {
    use oblikovati_api::types::{
        DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DOCUMENT_PRESENTATION,
    };
    let document = match state.document_type {
        Some(DOCUMENT_PART) => "part",
        Some(DOCUMENT_ASSEMBLY) => "assembly",
        Some(DOCUMENT_DRAWING) => "drawing",
        Some(DOCUMENT_PRESENTATION) => "presentation",
        _ => "unknown",
    };
    let suffix = match command {
        Builtin::Copy => "clipboard.copy",
        Builtin::Measure => "measure.measure",
        Builtin::Plane => return format!("{document}.model.work-features.plane"),
    };
    format!("{document}.tools.{suffix}")
}
/// Render a command at its applied location; example: `draw(ui, panel, state, owners, actions, false)`.
pub(crate) fn draw(
    ui: &Ui,
    panel: RibbonPanel,
    state: &GuiCommandState,
    owners: &mut RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
    expanded: bool,
) {
    let Some(command) = builtin_for(panel) else {
        return;
    };
    let id = command_id(state, command);
    let icons = owners.icons;
    if matches!(command, Builtin::Plane) {
        let right = owners
            .customization
            .builtin_group_draw(ui, &id, expanded, |ungrouped| {
                plane_group::draw(ui, state, icons, actions, ungrouped, expanded);
            });
        owners.quick_access.command_took_right_click |= right;
        return;
    }
    let right = owners
        .customization
        .builtin_draw(ui, &id, expanded, || match command {
            Builtin::Copy => unavailable_copy(ui, icons),
            Builtin::Measure => {
                let action = GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure);
                let size = if expanded {
                    icons.appearance.cell(0).1
                } else {
                    icons.appearance.single_cell()
                };
                icons.command_button(ui, "Measure", action, state, size, actions);
            }
            Builtin::Plane => {}
        });
    owners.quick_access.command_took_right_click |= right;
}
pub(crate) fn plane_main_cells(
    state: &GuiCommandState,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> usize {
    let id = command_id(state, Builtin::Plane);
    if customization.builtin_expanded(&id) {
        return 0;
    }
    if customization.builtin_ungrouped(&id) {
        return plane_group::COMMAND_COUNT;
    }
    1
}
fn unavailable_copy(ui: &Ui, icons: &crate::icons::GuiRibbonIcons) {
    let disabled = ui.begin_disabled(true);
    if icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
        crate::widget_abi::button(ui, "Copy", [108., 26.]);
    } else {
        icons.button(
            ui,
            "Copy",
            crate::icons::IconGlyph::Copy,
            icons.appearance.single_cell(),
        );
    }
    drop(disabled);
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text("This command is not available in this build.");
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn command_locations_are_distinct_across_panels_and_documents() {
        let mut state = GuiCommandState {
            document_type: Some(oblikovati_api::types::DOCUMENT_PART),
            ..Default::default()
        };
        assert_eq!(
            command_id(&state, Builtin::Copy),
            "part.tools.clipboard.copy"
        );
        assert_eq!(
            command_id(&state, Builtin::Measure),
            "part.tools.measure.measure"
        );
        state.document_type = Some(oblikovati_api::types::DOCUMENT_ASSEMBLY);
        assert_eq!(
            command_id(&state, Builtin::Copy),
            "assembly.tools.clipboard.copy"
        );
    }
}
