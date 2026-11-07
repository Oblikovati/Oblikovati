// SPDX-License-Identifier: GPL-2.0-only
//! Owned Dear ImGui frames for the application viewport draw boundary.
mod activity;
mod application;
pub use application::document_creation::templates::{
    GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits,
};
mod appearance_choice;
mod browser;
pub use browser::{
    GuiBrowserBody, GuiBrowserFeature, GuiBrowserKind, GuiBrowserSketch, GuiModelBrowser,
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
mod engine;
mod files;
mod frame_keys;
mod input;
mod parameters;
mod ribbon;
mod ribbon_fit;
mod ribbon_panel_contents;
mod ribbon_panels;
mod sketch;
pub use application::document_tabs::windows::{GuiDocumentWindow, active_window_name};
pub use application::home_recent_documents::{GuiRecentAction, GuiRecentDocument};
pub use application::quick_access_toolbar::plane_selection::{GuiOriginPlane, GuiOriginPlaneKind};
pub use application::viewport_orientation::{
    GuiCubeFace, GuiOrientCommand, GuiOrientationLayout, GuiViewAxes, GuiViewProjection,
    GuiViewStep,
};
pub use application::workspace_layout::body_properties::{GuiAppearanceChoice, GuiBodyPhysical};
pub use application::workspace_layout::browser_origin::GuiOriginAxis;
pub use sketch::constraint_glyphs::GuiConstraintGlyph;
pub use sketch::dimension_edit::{GuiDimensionEdit, GuiDimensionValue};
pub use sketch::inspect::{GuiInspectAction, GuiMeasurePanel, GuiSketchContext};
pub use sketch::line::{GuiLineCommand, GuiLineCommandMenu, GuiSketchHud, GuiSketchHudField};
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
pub use commands::{
    GuiAuthoringContext, GuiCommandState, GuiDocumentStorage, GuiParameterMutation,
    GuiPreviewImage, GuiProject, GuiProjectGroup, GuiShellAction,
};
pub use engine::{GuiFrameEngine, GuiFrameOutput, GuiFrameRequest};
pub use input::{GuiInputCapture, GuiInputEvent, GuiKey, GuiModifiers, GuiPointerButton};
pub use oblikovati_viewport::{GuiAtlas, GuiBatch, GuiFrame, GuiVertex};
pub use sketch_tools::GuiSketchTool;

mod sketch_constraints;
pub use sketch_constraints::GuiSketchConstraint;

mod dimensions;
pub use dimensions::{GuiDimensionAction, GuiDimensionDraft};

mod dimension_labels;
pub use dimension_labels::{GuiDimensionLabel, GuiDimensionLabelRole};

mod sketch_delete;

mod ui_scale;
