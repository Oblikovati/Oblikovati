// SPDX-License-Identifier: GPL-2.0-only
//! Ribbon panel identities and the panels available in each document context.
use crate::ribbon::RibbonTab;
use crate::{GuiCommandState, GuiShellAction};

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum RibbonPanel {
    Launch,
    Sketch,
    Create,
    Exit,
    Constrain,
    Layout,
    UserCommands,
    Modify,
    SketchPattern,
    Insert,
    ManageInsert,
    Format,
    Parameters,
    ToolOptions,
    ToolAssets,
    ToolMeasure,
    ToolFind,
    ToolClipboard,
    ToolTutorials,
    ToolTeamWeb,
    Navigate,
    Appearance,
    Visibility,
    Windows,
    Component,
    SheetMetalTools,
    DrawingViews,
    DrawingModify,
    DrawingSketch,
    DrawingSheets,
    AnnotateDimension,
    AnnotateText,
    AnnotateSymbols,
    PartCreate,
    PartModify,
    PartWorkFeatures,
    PartPattern,
    PartSimplification,
    PartNotes,
    PresentationModel,
}
impl RibbonPanel {
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::Launch => "Launch",
            Self::Sketch => "Sketch",
            Self::Create => "Create",
            Self::Exit => "Exit",
            Self::Constrain => "Constrain",
            Self::Layout => "Layout",
            Self::UserCommands => "User Commands",
            Self::Modify => "Modify",
            Self::SketchPattern => "Pattern",
            Self::Insert | Self::ManageInsert => "Insert",
            Self::Format => "Format",
            Self::Parameters => "Parameters",
            Self::ToolOptions => "Options",
            Self::ToolAssets => "Material and Appearance",
            Self::ToolMeasure => "Measure",
            Self::ToolFind => "Find",
            Self::ToolClipboard => "Clipboard",
            Self::ToolTutorials => "Tutorials",
            Self::ToolTeamWeb => "Team Web",
            Self::Navigate => "Navigate",
            Self::Appearance => "Appearance",
            Self::Visibility => "Visibility",
            Self::Windows => "Windows",
            Self::Component => "Component",
            Self::SheetMetalTools => "Sheet Metal",
            Self::DrawingViews | Self::PartCreate => "Create",
            Self::DrawingModify | Self::PartModify => "Modify",
            Self::PartWorkFeatures => "Work Features",
            Self::PartNotes => "Notes",
            Self::PartPattern => "Pattern",
            Self::PartSimplification => "Simplification",
            Self::DrawingSketch => "Sketch",
            Self::DrawingSheets => "Sheets",
            Self::AnnotateDimension => "Dimension",
            Self::AnnotateText => "Text",
            Self::AnnotateSymbols => "Symbols",
            Self::PresentationModel => "Workshop",
        }
    }
    pub(crate) fn command(self) -> Option<(&'static str, GuiShellAction)> {
        Some(match self {
            Self::Launch => ("New Part", GuiShellAction::NewPart),
            Self::Sketch => ("Start 2D Sketch", GuiShellAction::ChooseSketchPlane),
            Self::Exit => ("Finish Sketch", GuiShellAction::FinishSketch),
            Self::Modify => ("Delete", GuiShellAction::DeleteSketchSelection),
            Self::Navigate => ("Home", GuiShellAction::Home),
            Self::ToolMeasure => (
                "Measure",
                GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure),
            ),
            Self::SketchPattern
            | Self::Insert
            | Self::ManageInsert
            | Self::Create
            | Self::Constrain
            | Self::Layout
            | Self::UserCommands
            | Self::Format
            | Self::Parameters
            | Self::ToolOptions
            | Self::ToolAssets
            | Self::ToolFind
            | Self::ToolClipboard
            | Self::ToolTutorials
            | Self::ToolTeamWeb
            | Self::Appearance
            | Self::Visibility
            | Self::Windows
            | Self::Component
            | Self::SheetMetalTools
            | Self::DrawingViews
            | Self::DrawingModify
            | Self::DrawingSketch
            | Self::DrawingSheets
            | Self::AnnotateDimension
            | Self::AnnotateText
            | Self::AnnotateSymbols
            | Self::PartCreate
            | Self::PartModify
            | Self::PartWorkFeatures
            | Self::PartPattern
            | Self::PartSimplification
            | Self::PartNotes
            | Self::PresentationModel => {
                return None;
            }
        })
    }
}
/// Include the user panel in both the ribbon and its visibility controls.
pub(crate) fn available_panels(
    tab: RibbonTab,
    state: &GuiCommandState,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> Vec<RibbonPanel> {
    let mut panels = panels_for(tab, state);
    if state.document_type == Some(oblikovati_api::types::DOCUMENT_PART)
        && customization.has_panel(tab)
    {
        let before_exit = panels
            .iter()
            .position(|panel| *panel == RibbonPanel::Exit)
            .unwrap_or(panels.len());
        panels.insert(before_exit, RibbonPanel::UserCommands);
    }
    panels
}
/// Every panel a tab offers, hidden ones included.
pub(crate) fn panels_for(tab: RibbonTab, state: &GuiCommandState) -> Vec<RibbonPanel> {
    use RibbonPanel::*;
    let mut panels = match tab {
        RibbonTab::GetStarted => vec![Launch],
        RibbonTab::Model => vec![
            Sketch,
            PartCreate,
            PartModify,
            PartWorkFeatures,
            PartPattern,
            PartSimplification,
            Parameters,
        ],
        RibbonTab::SheetMetal => vec![SheetMetalTools, Sketch],
        RibbonTab::Assemble => vec![Component],
        RibbonTab::PlaceViews => vec![DrawingViews, DrawingModify, DrawingSketch, DrawingSheets],
        RibbonTab::Annotate
            if state.document_type == Some(oblikovati_api::types::DOCUMENT_PART) =>
        {
            vec![PartNotes]
        }
        RibbonTab::Annotate => vec![AnnotateDimension, AnnotateText, AnnotateSymbols],
        RibbonTab::Presentation => vec![PresentationModel],
        RibbonTab::Sketch => {
            let mut panels = vec![
                Create, Constrain, Layout, Modify, Insert, Format, Parameters, Exit,
            ];
            if state.sketch_pattern_available {
                panels.insert(4, SketchPattern);
            }
            panels
        }
        RibbonTab::Manage if state.document_type == Some(oblikovati_api::types::DOCUMENT_PART) => {
            vec![Parameters, ManageInsert]
        }
        RibbonTab::Manage => vec![Parameters],
        RibbonTab::Tools => vec![
            ToolMeasure,
            ToolAssets,
            ToolOptions,
            ToolClipboard,
            ToolFind,
        ],
        RibbonTab::View if state.theme.is_some() || state.color_scheme.is_some() => {
            vec![Visibility, Appearance, Windows, Navigate]
        }
        RibbonTab::View => vec![Visibility, Windows, Navigate],
    };
    if tab == RibbonTab::Tools && state.document_type == Some(oblikovati_api::types::DOCUMENT_PART)
    {
        panels.extend([ToolTeamWeb, ToolTutorials]);
    }
    if state.authoring == crate::GuiAuthoringContext::Sketch && !panels.contains(&Exit) {
        panels.push(Exit);
    }
    panels
}
