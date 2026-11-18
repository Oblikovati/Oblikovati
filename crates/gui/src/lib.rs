// SPDX-License-Identifier: GPL-2.0-only
//! Owned Dear ImGui frames for the application viewport draw boundary.
mod accessibility;
pub use accessibility::{
    GuiAccessibleFocus, GuiClosePrompt, GuiClosePromptButton, GuiClosePromptChoice,
};
mod activity;
mod application;
#[cfg(test)]
mod test_context;
pub use application::document_creation::templates::{
    GuiDocumentTemplate, GuiTemplateKind, GuiTemplateMaterial, GuiTemplateMaterialEntry,
    GuiTemplateUnits,
};
mod appearance_choice;
mod asset_choice;
pub use asset_choice::GuiAssetChoice;
mod browser;
pub use browser::{
    GuiBrowserBlockInstance, GuiBrowserBody, GuiBrowserFeature, GuiBrowserKind,
    GuiBrowserOccurrence, GuiBrowserRectangularPattern, GuiBrowserSketch, GuiExtrudeEditSeed,
    GuiFilletEditSeed, GuiMaterialChoice, GuiModelBrowser, GuiOccurrencePath,
};
mod clipboard;
pub use activity::GuiFrameActivity;
pub use clipboard::GuiClipboard;
mod chrome;
mod color_scheme_state;
mod commands;
#[allow(unsafe_code)]
mod context_owner;
mod dockable_lifecycle;
mod dockable_window;
mod document_tabs;
mod document_units;
mod engine;
mod external_dock_window;
mod files;
mod frame_keys;
mod history_browser;
mod input;
mod material_browser;
mod parameter_drafts;
mod parameter_panel;
mod parameter_snapshot;
pub use parameter_drafts::{
    GuiParameterAck, GuiParameterDrafts, GuiParameterTextEdit, GuiParameterTextField,
    GuiParameterTextTarget,
};
pub use parameter_panel::{
    GuiParameterAction, GuiParameterChoiceEdit, GuiParameterCreation, GuiParameterGroupAction,
    GuiParameterGroupEdit, GuiParameterInitialValue, GuiParameterLinkEdit,
    GuiParameterRowOperation, GuiParameterToggleField, GuiParameterToleranceEdit,
};
pub use parameter_snapshot::{
    GuiParameterGroup, GuiParameterLinkParameter, GuiParameterLinkSource, GuiParameterRow,
    GuiParameterSnapshot,
};
mod parameters;
mod ribbon;
mod ribbon_command_layout;
mod ribbon_fit;
mod ribbon_key_tips;
mod ribbon_panel_contents;
mod ribbon_panels;
mod ribbon_part_assets;
mod selection_filter;
pub use selection_filter::{GuiSelectionFilter, GuiSelectionFilterEntry, GuiSelectionKind};
mod sketch;
pub use application::document_opening::GuiInspectedFile;
pub use application::document_tabs::windows::{GuiDocumentWindow, active_window_name};
pub use application::home_recent_documents::{GuiRecentAction, GuiRecentDocument};
pub use application::quick_access_toolbar::plane_selection::{GuiOriginPlane, GuiOriginPlaneKind};
pub use application::viewport_orientation::{
    GuiCubeFace, GuiNavigationMode, GuiOrientCommand, GuiOrientationLayout, GuiViewAxes,
    GuiViewCommand, GuiViewProjection, GuiViewStep,
};
pub use application::workspace_layout::body_properties::{GuiAppearanceChoice, GuiBodyPhysical};
pub use application::workspace_layout::browser_origin::GuiOriginAxis;
pub use sketch::constraint_glyphs::{GuiConstraintGlyph, GuiConstraintHover};
pub use sketch::dimension_edit::{GuiDimensionEdit, GuiDimensionValue};
pub use sketch::inspect::{
    GuiInspectAction, GuiMeasurePanel, GuiSketchCandidate, GuiSketchContext, GuiSketchRelation,
    GuiSketchRepeat,
};
pub use sketch::line::{GuiLineCommand, GuiLineCommandMenu, GuiSketchHud, GuiSketchHudField};
pub use sketch::move_tool::{GuiSketchMoveAction, GuiSketchMoveStage, GuiSketchMoveStatus};
pub use sketch::precise_input::{
    GuiPreciseMode, GuiPrecisePoint, GuiPreciseTarget, GuiTriadCommand,
};
pub use sketch::region_properties::GuiRegionPanel;
mod sketch_flyouts;
mod sketch_formats;
mod sketch_tools;
mod snapshot;
mod theme;
pub use color_scheme_state::GuiColorSchemeState;
mod theme_editor;
mod theme_editor_rows;
mod theme_state;
pub use theme_state::GuiThemeState;
#[allow(unsafe_code)]
mod widget_abi;
pub use application::ribbon_customization::GuiRibbonSettingsStore;
pub use commands::{
    GuiAuthoringContext, GuiBaseViewRequest, GuiBrowserChoice, GuiCommandState, GuiDocumentStorage,
    GuiDrawingLine, GuiDrawingProjection, GuiExtrudeDistanceValidation, GuiExtrudeEditRequest,
    GuiExtrudePreview, GuiFeatureCondition, GuiFeatureEdit, GuiFilletEditRequest, GuiFilletRequest,
    GuiOffsetPlaneBase, GuiOffsetPlaneValidation, GuiParameterMutation, GuiPartAssetKind,
    GuiPreviewImage, GuiProject, GuiProjectGroup, GuiSelectionMode, GuiShellAction,
    GuiSketchLineStyle,
};
pub use engine::{
    GuiFrameEngine, GuiFrameOutput, GuiFrameRequest, GuiWindowAction, GuiWindowControls,
};
pub use sketch_pick_cue::{GuiModelPickCue, GuiSketchPickCue};
mod sketch_pick_cue;
pub use input::{GuiInputCapture, GuiInputEvent, GuiKey, GuiModifiers, GuiPointerButton};
pub use oblikovati_viewport::{
    GuiAtlas, GuiBatch, GuiFrame, GuiTextureKey, GuiTextureResources, GuiTextureSource, GuiVertex,
};
pub use sketch_tools::GuiSketchTool;

mod sketch_constraints;
pub use sketch_constraints::GuiSketchConstraint;

mod blend_dialog;
mod dimensions;
mod extrude_dialog;
mod face_properties;
pub use face_properties::GuiFaceProperties;
mod marking_menu;
mod marking_menu_layout;
mod offset_plane;
mod part_context_menu;
pub use dimensions::{GuiDimensionAction, GuiDimensionDraft};

mod dimension_labels;
pub use dimension_labels::{GuiDimensionLabel, GuiDimensionLabelRole};

mod sketch_delete;

mod part_options_ui;
mod sketch_options_ui;
mod ui_scale;

mod icons;

mod clear_appearance;

#[cfg(test)]
#[path = "../tests/context/parallel.rs"]
mod context_parallel_tests;

mod popup_press;
mod window_frame;
