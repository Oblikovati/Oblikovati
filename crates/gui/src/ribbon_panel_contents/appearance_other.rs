// SPDX-License-Identifier: GPL-2.0-only
//! Document-specific ribbon controls share icon layout without changing their actions.
use super::*;
use crate::icons::IconGlyph as Glyph;

pub(super) fn draw(
    ui: &Ui,
    panel: RibbonPanel,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    match panel {
        RibbonPanel::Component => component(ui, origin, state, owners, actions),
        RibbonPanel::DrawingViews => drawing_views(ui, origin, state, owners),
        RibbonPanel::PresentationModel => {
            let size = owners.icons.appearance.single_cell();
            if owners
                .icons
                .button(ui, "Insert Model", Glyph::InsertModel, size)
            {
                owners.workspaces.request_insert();
            }
        }
        _ => return unavailable(ui, panel, origin, owners.icons),
    }
    true
}
fn component(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let icons = owners.icons;
    if state.active_express() {
        if let Some(id) = state.document {
            crate::chrome::command_control(
                ui,
                "Load Full",
                GuiShellAction::LoadFullDocument(id),
                state,
                actions,
                || {
                    icons.button(
                        ui,
                        "Load Full",
                        Glyph::InsertModel,
                        icons.appearance.single_cell(),
                    )
                },
            );
        }
        return;
    }
    let (position, size) = icons.appearance.cell(0);
    icons.cursor(ui, origin, position);
    let disabled = ui.begin_disabled(owners.home_active || state.transaction_open);
    if icons.button(ui, "Place", Glyph::PlaceComponent, size) {
        owners.workspaces.request_assembly_place();
    }
    drop(disabled);
    let (position, size) = icons.appearance.cell(1);
    icons.cursor(ui, origin, position);
    let disabled = ui.begin_disabled(true);
    icons.button(ui, "Create", Glyph::NewAssembly, size);
    unavailable_tooltip(ui);
    drop(disabled);
}
fn drawing_views(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
) {
    let icons = owners.icons;
    let (position, size) = icons.appearance.cell(0);
    icons.cursor(ui, origin, position);
    let disabled = ui.begin_disabled(state.transaction_open || state.drawing_updates_deferred);
    if icons.button(ui, "Base View", Glyph::DrawingBase, size) {
        owners.workspaces.request_base_view();
    }
    drop(disabled);
    let (position, size) = icons.appearance.cell(1);
    icons.cursor(ui, origin, position);
    let disabled = ui.begin_disabled(true);
    icons.button(ui, "Projected View", Glyph::DrawingProjected, size);
    unavailable_tooltip(ui);
    drop(disabled);
}
fn unavailable(
    ui: &Ui,
    panel: RibbonPanel,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
) -> bool {
    let entries: &[(&str, Option<Glyph>)] = match panel {
        RibbonPanel::SheetMetalTools => &[
            ("Flange", Some(Glyph::SheetFlange)),
            ("Flat Pattern", Some(Glyph::SheetUnfold)),
        ],
        RibbonPanel::DrawingModify => &[
            ("Break", Some(Glyph::DrawingBreak)),
            ("Crop", Some(Glyph::DrawingCrop)),
        ],
        RibbonPanel::DrawingSketch => &[("Start Sketch", Some(Glyph::CreateSketch))],
        RibbonPanel::DrawingSheets => &[("New Sheet", Some(Glyph::DrawingSheet))],
        RibbonPanel::AnnotateDimension => &[
            ("General", Some(Glyph::Dimension)),
            ("Baseline", Some(Glyph::DrawingBaseline)),
        ],
        RibbonPanel::AnnotateText => &[("Leader", Some(Glyph::Annotation))],
        RibbonPanel::AnnotateSymbols => &[
            ("Surface", Some(Glyph::DrawingSurface)),
            ("Weld", Some(Glyph::DrawingWeld)),
        ],
        RibbonPanel::PartNotes => &[
            ("General Note", Some(Glyph::Annotation)),
            ("Leader Note", Some(Glyph::Annotation)),
        ],
        _ => return false,
    };
    muted_icon_commands(ui, origin, icons, entries, false);
    true
}

fn unavailable_tooltip(ui: &Ui) {
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(NOT_PORTED);
    }
}
