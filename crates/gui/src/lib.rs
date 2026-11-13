// SPDX-License-Identifier: GPL-2.0-only
//! Owned Dear ImGui frames for the application viewport draw boundary.
mod activity;
mod application;
pub use application::document_creation::templates::{
    GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits,
};
mod appearance_choice;
mod asset_choice;
pub use asset_choice::GuiAssetChoice;
mod browser;
pub use browser::{
    GuiBrowserBlockInstance, GuiBrowserBody, GuiBrowserFeature, GuiBrowserKind,
    GuiBrowserRectangularPattern, GuiBrowserSketch, GuiMaterialChoice, GuiModelBrowser,
};
mod clipboard;
pub use activity::GuiFrameActivity;
pub use clipboard::GuiClipboard;
mod chrome;
mod color_scheme_state;
mod commands;
#[allow(unsafe_code)]
mod context_owner;
mod document_tabs;
mod document_units;
mod engine;
mod files;
mod frame_keys;
mod input;
mod material_browser;
mod parameters;
mod ribbon;
mod ribbon_fit;
mod ribbon_key_tips;
mod ribbon_panel_contents;
mod ribbon_panels;
mod ribbon_part_assets;
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
pub use sketch::constraint_glyphs::GuiConstraintGlyph;
pub use sketch::dimension_edit::{GuiDimensionEdit, GuiDimensionValue};
pub use sketch::inspect::{
    GuiInspectAction, GuiMeasurePanel, GuiSketchContext, GuiSketchRelation, GuiSketchRepeat,
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
    GuiDrawingLine, GuiDrawingProjection, GuiExtrudePreview, GuiFeatureCondition, GuiFeatureEdit,
    GuiParameterMutation, GuiPartAssetKind, GuiPreviewImage, GuiProject, GuiProjectGroup,
    GuiShellAction, GuiSketchLineStyle,
};
pub use engine::{GuiFrameEngine, GuiFrameOutput, GuiFrameRequest};
pub use input::{GuiInputCapture, GuiInputEvent, GuiKey, GuiModifiers, GuiPointerButton};
pub use oblikovati_viewport::{
    GuiAtlas, GuiBatch, GuiFrame, GuiTextureKey, GuiTextureResources, GuiTextureSource, GuiVertex,
};
pub use sketch_tools::GuiSketchTool;

mod sketch_constraints;
pub use sketch_constraints::GuiSketchConstraint;

mod dimensions;
mod extrude_dialog;
pub use dimensions::{GuiDimensionAction, GuiDimensionDraft};

mod dimension_labels;
pub use dimension_labels::{GuiDimensionLabel, GuiDimensionLabelRole};

mod sketch_delete;

mod ui_scale;

mod icons;
