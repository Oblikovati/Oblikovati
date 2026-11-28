// SPDX-License-Identifier: GPL-2.0-only
//! Owned Dear ImGui frames for the application viewport draw boundary.
mod accessibility;
#[cfg(target_arch = "wasm32")]
#[allow(unsafe_code)]
mod wasm_c_abi;
pub use accessibility::{
    GuiAccessibleFocus, GuiClosePrompt, GuiClosePromptButton, GuiClosePromptChoice,
};
mod activity;
mod application;
#[cfg(test)]
mod test_context;
pub use application::document_creation::template_previews::GuiTemplateSheet;
pub use application::document_creation::templates::{
    GuiDocumentTemplate, GuiProjectTemplate, GuiTemplateFolder, GuiTemplateKind,
    GuiTemplateMaterial, GuiTemplateMaterialEntry, GuiTemplateUnits,
};
mod appearance_choice;
mod asset_choice;
pub use asset_choice::{GuiAppearancePreview, GuiAssetChoice, GuiAssetLibrary};
mod appearance_editor;
mod asset_browser_view;
pub use appearance_editor::{
    AppearanceCopy, AppearanceTemplate, GuiAppearanceChange, GuiAppearanceEdit,
};
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
mod command_ids;
mod commands;
#[allow(unsafe_code)]
mod context_owner;
mod dockable_lifecycle;
mod dockable_window;
mod document_tabs;
mod document_units;
mod document_units_draft;
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
    GuiParameterEngineering, GuiParameterGroup, GuiParameterLinkParameter, GuiParameterLinkSource,
    GuiParameterRow, GuiParameterSnapshot,
};
mod parameters;
mod ribbon;
mod ribbon_command_layout;
mod ribbon_fit;
mod ribbon_key_tips;
mod ribbon_panel_contents;
mod ribbon_panels;
mod ribbon_part_assets;
mod ribbon_split;
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
pub use sketch::constraint_glyphs::{
    GuiConstraintCandidate, GuiConstraintGlyph, GuiConstraintHover,
};
pub use sketch::curvature::{GuiCurvatureAction, GuiCurvatureSettings, GuiCurvatureSliders};
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
pub use sketch::text::{
    GuiSketchTextAction, GuiSketchTextDraft, GuiSketchTextEditorFields, GuiSketchTextFieldSources,
    GuiSketchTextFormatting, GuiSketchTextPresetId,
};
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
mod extrude_handle;
pub use commands::{
    GuiAuthoringContext, GuiBaseViewRequest, GuiBrowserChoice, GuiCommandState, GuiDocumentStorage,
    GuiDrawingLayoutAction, GuiDrawingLine, GuiDrawingProjection, GuiExtrudeDistanceValidation,
    GuiExtrudeEditRequest, GuiExtrudePreview, GuiExtrudeProfilePick, GuiExtrudeReferenceExtent,
    GuiFilletEditRequest, GuiFilletRequest, GuiLayoutComponent, GuiMoveBodyRequest,
    GuiOffsetPlaneBase, GuiOffsetPlaneValidation, GuiParameterMutation, GuiPartAssetKind,
    GuiPreviewImage, GuiProject, GuiProjectAction, GuiProjectGroup, GuiProjectSettings,
    GuiSelectionMode, GuiShellAction, GuiShellFeatureRequest, GuiSketchLineStyle,
};
pub use engine::{
    GuiFrameEngine, GuiFrameOutput, GuiFrameRequest, GuiWindowAction, GuiWindowControls,
};
pub use extrude_handle::GuiExtrudeHandle;
pub use oblikovati_model::feature::sketch_prism::SketchPrismDirection as GuiExtrudeDirection;
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
mod chamfer_panel;
pub use chamfer_panel::{
    GuiChamferPartialState, GuiChamferPickMode, GuiChamferTab, GuiChamferV2Action,
    GuiChamferV2EditSeed, GuiChamferV2Pick, GuiChamferV2Preview, GuiChamferV2PreviewState,
    GuiChamferV2Request,
};
mod fillet_panel;
pub use fillet_panel::{
    GuiFilletCornerSetback, GuiFilletEdgeSet, GuiFilletSelectionMode, GuiFilletSharedVertex,
    GuiFilletV2Action, GuiFilletV2EditSeed, GuiFilletV2Pick, GuiFilletV2Preview,
    GuiFilletV2PreviewState, GuiFilletV2Request, GuiFilletVariablePoint,
};
mod combine_dialog;
mod dimensions;
mod draft_dialog;
mod extrude_dialog;
#[cfg(test)]
mod face_draft;
mod face_properties;
mod hole_dialog;
mod slot_dialog;
mod split_dialog;
pub use application::model_browser::feature_properties::{GuiFeatureCondition, GuiFeatureEdit};
pub use combine_dialog::{
    GuiCombineAction, GuiCombineEditSeed, GuiCombinePick, GuiCombinePickMode, GuiCombinePreview,
    GuiCombinePreviewState, GuiCombineRequest,
};
pub use draft_dialog::{
    GuiDraftAction, GuiDraftEditSeed, GuiDraftPick, GuiDraftPickTarget, GuiDraftPreview,
    GuiDraftPreviewState, GuiDraftRequest,
};
pub use hole_dialog::{
    GuiHoleAction, GuiHoleEditSeed, GuiHolePick, GuiHolePreview, GuiHolePreviewState,
    GuiHoleRequest, GuiHoleSeatKind, GuiHoleTermination,
};
pub use slot_dialog::{
    GuiSlotAction, GuiSlotEditSeed, GuiSlotPick, GuiSlotPreview, GuiSlotPreviewState,
    GuiSlotRequest,
};
pub use split_dialog::{
    GuiSplitAction, GuiSplitEditSeed, GuiSplitPick, GuiSplitPickKind, GuiSplitPickMode,
    GuiSplitPreview, GuiSplitPreviewState, GuiSplitRequest,
};
mod loft_dialog;
mod move_body_dialog;
mod move_body_ribbon;
mod shell_feature_dialog;
mod shell_panel;
pub use loft_dialog::{
    GuiLoftAction, GuiLoftEditSeed, GuiLoftElement, GuiLoftPathMode, GuiLoftPlacedView,
    GuiLoftSection,
};
pub use shell_panel::{
    GuiShellOverrideSet, GuiShellPickMode, GuiShellV2Action, GuiShellV2EditSeed, GuiShellV2Pick,
    GuiShellV2Preview, GuiShellV2PreviewState, GuiShellV2Request,
};
mod feature_properties;
pub use feature_properties::FeatureOutputMode;
mod coil_dialog;
pub use coil_dialog::{
    GuiCoilAction, GuiCoilEditSeed, GuiCoilPreview, GuiCoilPreviewState, GuiCoilRequest,
};
mod emboss_dialog;
pub use emboss_dialog::{
    GuiEmbossAction, GuiEmbossEditSeed, GuiEmbossPreview, GuiEmbossPreviewState, GuiEmbossRequest,
};
mod derive_dialog;
pub use derive_dialog::{
    GuiDeriveAction, GuiDeriveEditSeed, GuiDerivePreview, GuiDerivePreviewState, GuiDeriveRequest,
    GuiDeriveSourceSummary,
};
mod rib_dialog;
pub use rib_dialog::{
    GuiRibAction, GuiRibEditSeed, GuiRibPreview, GuiRibPreviewState, GuiRibRequest,
    GuiRibSketchCandidate,
};
mod decal_dialog;
pub use decal_dialog::{
    GuiDecalAction, GuiDecalEditSeed, GuiDecalImageCandidate, GuiDecalPreview,
    GuiDecalPreviewState, GuiDecalRequest,
};
mod import_dialog;
pub use import_dialog::{
    GuiImportAction, GuiImportEditSeed, GuiImportPreview, GuiImportPreviewState, GuiImportRequest,
    GuiImportSourceSummary,
};
mod unwrap_dialog;
pub use unwrap_dialog::{
    GuiUnwrapAction, GuiUnwrapEdgeMode, GuiUnwrapEditSeed, GuiUnwrapFacePick, GuiUnwrapPreview,
    GuiUnwrapPreviewState, GuiUnwrapRequest,
};
mod revolve_dialog;
pub use revolve_dialog::{
    GuiRevolveAction, GuiRevolveEditSeed, GuiRevolveExtent, GuiRevolvePreview,
    GuiRevolvePreviewState, GuiRevolveRequest,
};
mod sweep_dialog;
pub use sweep_dialog::{
    GuiSweepAction, GuiSweepEditSeed, GuiSweepOrientation, GuiSweepOutput, GuiSweepPresetStore,
    GuiSweepPreview, GuiSweepPreviewState, GuiSweepProfile, GuiSweepRequest,
};
mod shell_ribbon;
pub use face_properties::GuiFaceProperties;
mod marking_menu;
mod marking_menu_layout;
mod offset_plane;
mod part_context_menu;
pub use dimensions::{GuiDimensionAction, GuiDimensionDraft};

mod dimension_labels;
pub use application::workspace_layout::object_visibility::GuiOverlayCategory;
pub use dimension_labels::{GuiDimensionLabel, GuiDimensionLabelRole};

mod sketch_delete;

mod constraint_settings_ui;
mod part_options_ui;
mod sketch_options_hud;
mod sketch_options_ui;
mod ui_scale;

mod icons;

mod clear_appearance;

#[cfg(test)]
#[path = "../tests/context/parallel.rs"]
mod context_parallel_tests;

#[allow(unsafe_code)]
mod detached_drawing;
mod popup_press;
mod window_frame;
pub use detached_drawing::{GuiDetachedDrawing, GuiDetachedDrawingFrame, drawing_preview_frame};

mod model_state_table;
mod model_state_table_acceptance;
mod model_state_table_cells;
mod model_state_table_extra;
mod model_state_table_rows;
