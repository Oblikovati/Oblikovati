// SPDX-License-Identifier: GPL-2.0-only
//! Ribbon panel identities and the panels available in each document context.
use crate::application::ribbon_display::RibbonPlacement;
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
    ManageUpdate,
    ManageDocument,
    ManageStyles,
    ManageInsert,
    ManagePointCloud,
    ManageLayout,
    ManageAuthor,
    ManageRules,
    ManageContentCenter,
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
    PartPrimitives,
    PartCreate,
    PartModify,
    PartExplore,
    PartWorkFeatures,
    PartPattern,
    PartSimplification,
    PartSimulation,
    PartPrint,
    PartNotes,
    PartShapeGenerator,
    PartFreeform,
    PartSurface,
    PartPlastic,
    PartHarness,
    PartFactory,
    PartConvert,
    PartGeometricAnnotation,
    PartGeneralAnnotation,
    PartAnnotationManage,
    PartToleranceAnalysis,
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
            Self::ManageUpdate => "Update",
            Self::ManageDocument => "Manage",
            Self::ManageStyles => "Styles and Standards",
            Self::ManagePointCloud => "Point Cloud",
            Self::ManageLayout => "Layout",
            Self::ManageAuthor => "Author",
            Self::ManageRules => "iLogic",
            Self::ManageContentCenter => "Content Center",
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
            Self::PartPrimitives => "Primitives",
            Self::DrawingViews | Self::PartCreate => "Create",
            Self::DrawingModify | Self::PartModify => "Modify",
            Self::PartExplore => "Explore",
            Self::PartWorkFeatures => "Work Features",
            Self::PartNotes => "Notes",
            Self::PartShapeGenerator => "Shape Generator",
            Self::PartFreeform => "Create Freeform",
            Self::PartSurface => "Surface",
            Self::PartPlastic => "Plastic Part",
            Self::PartHarness => "Harness",
            Self::PartFactory => "iPart/Assembly",
            Self::PartConvert => "Convert",
            Self::PartGeometricAnnotation => "Geometric Annotation",
            Self::PartGeneralAnnotation => "General Annotation",
            Self::PartAnnotationManage => "Manage",
            Self::PartToleranceAnalysis => "Tolerance Analysis",
            Self::PartPattern => "Pattern",
            Self::PartSimplification => "Simplification",
            Self::PartSimulation => "Simulation",
            Self::PartPrint => "3D Print",
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
            Self::ManageUpdate => ("Update", GuiShellAction::UpdatePart),
            Self::ManageDocument => ("iProperties", GuiShellAction::OpenIProperties),
            Self::ToolMeasure | Self::PartExplore => (
                "Measure",
                GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure),
            ),
            Self::SketchPattern
            | Self::Insert
            | Self::ManageInsert
            | Self::ManageStyles
            | Self::ManagePointCloud
            | Self::ManageLayout
            | Self::ManageAuthor
            | Self::ManageRules
            | Self::ManageContentCenter
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
            | Self::PartPrimitives
            | Self::PartCreate
            | Self::PartModify
            | Self::PartWorkFeatures
            | Self::PartPattern
            | Self::PartSimplification
            | Self::PartSimulation
            | Self::PartPrint
            | Self::PartNotes
            | Self::PartShapeGenerator
            | Self::PartFreeform
            | Self::PartSurface
            | Self::PartPlastic
            | Self::PartHarness
            | Self::PartFactory
            | Self::PartConvert
            | Self::PartGeometricAnnotation
            | Self::PartGeneralAnnotation
            | Self::PartAnnotationManage
            | Self::PartToleranceAnalysis
            | Self::PresentationModel => {
                return None;
            }
        })
    }
}
/// Match a custom panel to its document type, as in `supports_custom_panel(RibbonTab::Assemble, state)`.
pub(crate) fn supports_custom_panel(tab: RibbonTab, state: &GuiCommandState) -> bool {
    use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART};
    if tab == RibbonTab::Assemble {
        state.document_type == Some(DOCUMENT_ASSEMBLY)
    } else {
        state.document_type == Some(DOCUMENT_PART)
    }
}

/// Include the user panel in both the ribbon and its visibility controls.
pub(crate) fn available_panels(
    tab: RibbonTab,
    state: &GuiCommandState,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> Vec<RibbonPanel> {
    let mut panels = panels_for(tab, state);
    if supports_custom_panel(tab, state) && customization.has_panel(tab) {
        let before_exit = panels
            .iter()
            .position(|panel| *panel == RibbonPanel::Exit)
            .unwrap_or(panels.len());
        panels.insert(before_exit, RibbonPanel::UserCommands);
    }
    panels
}
/// Match the panel catalog to the ribbon placement, e.g. the side-docked 3D Model groups.
pub(crate) fn panels_for_placement(
    tab: RibbonTab,
    state: &GuiCommandState,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
    placement: RibbonPlacement,
) -> Vec<RibbonPanel> {
    let panels = available_panels(tab, state, customization);
    match placement {
        RibbonPlacement::Left | RibbonPlacement::Right if tab == RibbonTab::Model => {
            panels.into_iter().flat_map(side_model_groups).collect()
        }
        RibbonPlacement::Floating => panels.into_iter().flat_map(floating_groups).collect(),
        _ => panels,
    }
}
fn side_model_groups(panel: RibbonPanel) -> impl Iterator<Item = RibbonPanel> {
    use RibbonPanel::*;
    let extra: &[RibbonPanel] = match panel {
        PartPattern => &[PartFreeform],
        PartSurface => &[Parameters],
        PartSimplification => &[PartPlastic, ToolMeasure, Insert, PartHarness, PartFactory],
        PartSimulation => &[PartConvert],
        _ => &[],
    };
    // Parameters is placed above Simplification on the side ribbon.
    (panel != Parameters)
        .then_some(panel)
        .into_iter()
        .chain(extra.iter().copied())
}
fn floating_groups(panel: RibbonPanel) -> impl Iterator<Item = RibbonPanel> {
    use RibbonPanel::*;
    let entries = match panel {
        PartModify => [Some(panel), Some(PartShapeGenerator), None, None, None],
        PartPattern => [Some(panel), Some(PartFreeform), None, None, None],
        PartNotes => [
            Some(PartGeometricAnnotation),
            Some(PartGeneralAnnotation),
            Some(panel),
            Some(PartAnnotationManage),
            Some(PartToleranceAnalysis),
        ],
        _ => [Some(panel), None, None, None, None],
    };
    entries.into_iter().flatten()
}

/// Every panel a tab offers, hidden ones included.
pub(crate) fn panels_for(tab: RibbonTab, state: &GuiCommandState) -> Vec<RibbonPanel> {
    use RibbonPanel::*;
    let mut panels = match tab {
        RibbonTab::GetStarted => vec![Launch],
        RibbonTab::Model => vec![
            Sketch,
            PartPrimitives,
            PartCreate,
            PartModify,
            PartExplore,
            PartWorkFeatures,
            PartPattern,
            PartSurface,
            PartSimplification,
            Parameters,
            PartSimulation,
            PartPrint,
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
            vec![
                ManageUpdate,
                Parameters,
                ManageDocument,
                ManageStyles,
                ManageInsert,
                ManagePointCloud,
                ManageLayout,
                ManageAuthor,
                ManageRules,
                ManageContentCenter,
            ]
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
