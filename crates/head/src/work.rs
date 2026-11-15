// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadSketchWork, HeadWorkStatus};
/// Owner-thread services borrow model state only while the window loop polls them.
pub trait HeadWork: HeadSketchWork {
    orientation_work_methods!();
    /// Set the viewport selection mode; example: `work.set_selection_mode_from_gui(mode)?`.
    fn set_selection_mode_from_gui(
        &mut self,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<bool, String> {
        Err(format!(
            "selection mode {mode:?}: expected a desktop selection owner"
        ))
    }
    /// Set one ambient selection category.
    /// Example: `work.set_selection_enabled_from_gui(kind, true)?`.
    fn set_selection_enabled_from_gui(
        &mut self,
        kind: oblikovati_gui::GuiSelectionKind,
        enabled: bool,
    ) -> Result<bool, String> {
        Err(format!(
            "selection category {kind:?}, enabled {enabled}: expected a desktop selection owner"
        ))
    }
    /// Change an ambient category's pick priority.
    /// Example: `work.move_selection_kind_from_gui(2, 1)?`.
    fn move_selection_kind_from_gui(&mut self, from: usize, to: usize) -> Result<bool, String> {
        Err(format!(
            "selection priority {from}->{to}: expected a desktop selection owner"
        ))
    }
    /// Restore the default selection filter; example: `work.reset_selection_filter_from_gui()?`.
    fn reset_selection_filter_from_gui(&mut self) -> Result<bool, String> {
        Err("selection filter reset: expected a desktop selection owner".into())
    }
    /// Move a document's transaction cursor.
    /// Example: `work.jump_history_from_gui(document, 0)?`.
    fn jump_history_from_gui(&mut self, document: u64, position: i64) -> Result<bool, String> {
        Err(format!(
            "history {document}/{position}: expected a document history owner"
        ))
    }
    /// Recompute the active part; example: `work.update_part_from_gui()?`.
    fn update_part_from_gui(&mut self) -> Result<bool, String> {
        Err("part update: expected a part evaluation owner".into())
    }
    /// Apply a marking menu; example: `work.set_marking_menu_from_gui(&args)?`.
    fn set_marking_menu_from_gui(
        &mut self,
        args: &oblikovati_api::wire::SetMarkingMenuArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "marking menu {args:?}: expected a desktop menu owner"
        ))
    }
    /// Take a resolved command shortcut after GUI input ownership is known.
    /// Example: `work.take_shortcut_action()` after resolving one input.
    fn take_shortcut_action(&mut self) -> Option<oblikovati_gui::GuiShellAction> {
        None
    }
    /// Apply a keymap draft; example: `work.set_keymap_from_gui(&delta)?`.
    fn set_keymap_from_gui(
        &mut self,
        delta: &oblikovati_api::wire::KeymapExport,
    ) -> Result<bool, String> {
        Err(format!("keymap {delta:?}: expected a desktop keymap owner"))
    }
    /// Resolve application shortcuts before GUI ownership is decided.
    /// Example: `work.resolve_shortcut(event)` before dispatch.
    fn resolve_shortcut(
        &mut self,
        event: oblikovati_viewport::ViewportEvent,
    ) -> oblikovati_viewport::ViewportEvent {
        event
    }
    /// Select a block row in the edited sketch; example: `work.select_browser_block_from_gui(document, sketch, block, false)?`.
    fn select_browser_block_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        instance: oblikovati_model::sketch::SketchEntityId,
        toggle: bool,
    ) -> Result<bool, String> {
        Err(format!(
            "browser block {instance:?} in sketch {sketch:?}, document {document}, toggle {toggle}: expected a sketch selection owner"
        ))
    }
    /// Change the block selection mode; example: `work.select_block_geometry_from_gui(true)?`.
    fn select_block_geometry_from_gui(&mut self, active: bool) -> Result<bool, String> {
        Err(format!(
            "block geometry selection {active}: expected an active sketch owner"
        ))
    }
    /// Set document unit preferences through the typed API; example: `work.set_document_units_from_gui(id, &request)?`.
    fn set_document_units_from_gui(
        &mut self,
        document: u64,
        request: &oblikovati_api::wire::SetDocumentUnitsArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "document {document} units {request:?}: expected a document owner"
        ))
    }
    /// Show or clear a drawing preview; example: `work.base_view_preview_from_gui(None)?`.
    fn base_view_preview_from_gui(
        &mut self,
        request: Option<&oblikovati_gui::GuiBaseViewRequest>,
    ) -> Result<bool, String> {
        Err(format!(
            "drawing view preview {request:?}: expected a drawing owner"
        ))
    }
    /// Create a drawing view through the public drawing commands; example: `work.base_view_from_gui(&request)?`.
    fn base_view_from_gui(
        &mut self,
        request: &oblikovati_gui::GuiBaseViewRequest,
    ) -> Result<bool, String> {
        Err(format!(
            "drawing view {request:?}: expected a drawing owner"
        ))
    }
    /// Place an open component through the document owner; example: `work.place_component_from_gui(assembly, &request)?`.
    fn place_component_from_gui(
        &mut self,
        assembly: u64,
        request: &oblikovati_api::wire::PlaceOccurrenceArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "assembly {assembly} placement {request:?}: expected an assembly owner"
        ))
    }
    /// Place a component from disk; example: `work.place_component_file_from_gui(assembly, path, name)?`.
    fn place_component_file_from_gui(
        &mut self,
        assembly: u64,
        path: &str,
        name: &str,
    ) -> Result<bool, String> {
        Err(format!(
            "assembly {assembly} file {path:?} occurrence {name:?}: expected an assembly owner"
        ))
    }
    /// Show or clear an assembly placement preview; example: `work.preview_component_from_gui(None)?`.
    fn preview_component_from_gui(&mut self, request: Option<(u64, u64)>) -> Result<bool, String> {
        Err(format!(
            "component preview {request:?}: expected an assembly owner"
        ))
    }
    /// Preview a file before placement; example: `work.preview_component_file_from_gui(assembly, path)?`.
    fn preview_component_file_from_gui(
        &mut self,
        assembly: u64,
        path: &str,
    ) -> Result<bool, String> {
        Err(format!(
            "assembly {assembly} preview file {path:?}: expected an assembly owner"
        ))
    }
    /// Apply an Extrude dialog request through the document command owner.
    /// Example: `work.extrude_from_gui(document, sketch, 0, "10 mm")?`.
    fn extrude_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: &str,
    ) -> Result<bool, String> {
        Err(format!(
            "extrude document {document}, sketch {sketch:?}, profile {profile}, distance {distance:?}: expected a certified extrusion owner"
        ))
    }
    /// Validate a draft through the document owner; example: `work.validate_extrusion_distance(1, "10 mm")?`.
    fn validate_extrusion_distance(
        &mut self,
        document: u64,
        distance: &str,
    ) -> Result<bool, String> {
        Err(format!(
            "extrusion document {document}, distance {distance:?}: expected a document owner"
        ))
    }
    /// Show or clear a transient extrusion; example: `work.extrusion_preview_from_gui(None)?`.
    fn extrusion_preview_from_gui(
        &mut self,
        request: Option<&oblikovati_gui::GuiExtrudePreview>,
    ) -> Result<bool, String> {
        Err(format!(
            "extrusion preview {request:?}: expected a certified extrusion owner"
        ))
    }
    /// Change how often a prompt appears; example: `work.set_prompt_frequency_from_gui(prompt, frequency)?`.
    fn set_prompt_frequency_from_gui(
        &mut self,
        prompt: oblikovati_model::application_prompts::ApplicationPrompt,
        frequency: oblikovati_model::application_prompts::PromptFrequency,
    ) -> Result<bool, String> {
        Err(format!(
            "prompt {prompt:?}, frequency {frequency:?}: expected an application preference owner"
        ))
    }
    /// Change the stored recent list; example: `work.recent_from_gui(&action)`.
    fn recent_from_gui(
        &mut self,
        action: &oblikovati_gui::GuiRecentAction,
    ) -> Result<bool, String> {
        Err(format!(
            "recent list {action:?}: expected an owner with a recent list"
        ))
    }
    /// The window in front, for the view it keeps; example: `work.view_slot()`.
    fn view_slot(&self) -> u64 {
        0
    }
    /// Bring forward or close one document window; example: `work.document_window_from_gui(7, false)`.
    fn document_window_from_gui(&mut self, id: u64, close: bool) -> Result<bool, String> {
        Err(format!(
            "document window {id}, close={close}: expected an owner with document windows"
        ))
    }
    /// Open a second window on the active document; example: the View tab's New.
    fn new_document_window_from_gui(&mut self) -> Result<bool, String> {
        Err("New window: expected an owner with document windows".into())
    }
    /// Apply one inspection choice; example: `work.inspect_from_gui(action)`.
    fn inspect_from_gui(
        &mut self,
        action: oblikovati_gui::GuiInspectAction,
    ) -> Result<bool, String> {
        Err(format!(
            "inspection {action:?}: expected a document with an edited sketch"
        ))
    }
    /// Save application text/icon scale; example: `work.set_ui_scale_from_gui(options)?`.
    fn set_ui_scale_from_gui(
        &mut self,
        options: oblikovati_model::ui_scale::UiScaleOptions,
    ) -> Result<bool, String> {
        Err(format!(
            "UI scale {options:?}: expected an application preference owner"
        ))
    }
    crate::work_theme::theme_work_methods!();
    /// Activate a visible document through its owner; example: `work.activate_document_from_gui(&request)?`.
    fn activate_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::ActivateDocumentArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "activate document {}: expected a document owner",
            request.id
        ))
    }
    /// Close a document through its owner's save guard; example: `work.close_document_from_gui(&request)?`.
    fn close_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::CloseDocumentArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "close document {}: expected an owner with document file support",
            request.id
        ))
    }
    /// Open a chosen document through the document owner; example: `work.open_document_from_gui(&request)?`.
    fn open_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::OpenDocumentArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "open document {:?}: expected an owner with document file support",
            request.full_document_name
        ))
    }
    /// Open a drawing with deferred updates; example: `work.open_deferred_drawing_from_gui(path)?`.
    fn open_deferred_drawing_from_gui(&mut self, path: &str) -> Result<bool, String> {
        Err(format!(
            "deferred drawing {path:?}: expected a drawing owner"
        ))
    }
    /// Choose one feature row; example: `work.select_browser_feature_from_gui(document, feature)?`.
    fn select_browser_feature_from_gui(
        &mut self,
        document: u64,
        feature: oblikovati_model::feature::FeatureId,
    ) -> Result<bool, String> {
        Err(format!(
            "browser feature {feature:?}, document {document}: expected a selection owner"
        ))
    }
    /// Calculate one solid's physical values; example: `work.update_body_properties_from_gui(document, 0)?`.
    fn update_body_properties_from_gui(
        &mut self,
        document: u64,
        body: i64,
    ) -> Result<bool, String> {
        Err(format!(
            "body {body}, document {document}: expected a properties owner"
        ))
    }
    /// Assign a part asset; example: `work.assign_part_asset_from_gui(document, kind, id)?`.
    fn assign_part_asset_from_gui(
        &mut self,
        document: u64,
        kind: oblikovati_gui::GuiPartAssetKind,
        id: &str,
    ) -> Result<bool, String> {
        Err(format!(
            "part {document}, asset {kind:?} {id:?}: expected an asset owner"
        ))
    }
    /// Give a solid its own appearance; example: `work.set_body_appearance_from_gui(document, key, id)?`.
    fn set_body_appearance_from_gui(
        &mut self,
        document: u64,
        key: &str,
        appearance: &str,
    ) -> Result<bool, String> {
        Err(format!(
            "body {key:?}, appearance {appearance:?}, document {document}: expected a properties owner"
        ))
    }
    /// Change a feature's name and suppression; example: from Feature Properties.
    fn edit_feature_from_gui(
        &mut self,
        document: u64,
        edit: &oblikovati_gui::GuiFeatureEdit,
    ) -> Result<bool, String> {
        Err(format!(
            "feature edit {edit:?}, document {document}: expected a feature owner"
        ))
    }
    /// Choose a whole browser row or change what it shows; example: a body's Visibility.
    fn browser_choice_from_gui(
        &mut self,
        document: u64,
        choice: &oblikovati_gui::GuiBrowserChoice,
    ) -> Result<bool, String> {
        Err(format!(
            "browser choice {choice:?}, document {document}: expected a selection owner"
        ))
    }
    /// Say what a document's next save stores as its picture, without a redraw of its own.
    fn keep_document_picture_from_gui(
        &mut self,
        document: u64,
        _: Option<&oblikovati_gui::GuiPreviewImage>,
    ) -> Result<bool, String> {
        Err(format!("document {document}: expected a picture owner"))
    }
    /// Read a closed file's stored properties; example: on a selection in the Open dialog.
    fn inspect_file_from_gui(&mut self, path: &str) -> Result<bool, String> {
        Err(format!("file {path:?}: expected a file inspection owner"))
    }
    /// Show a running navigation command's prompt, or Ready; example: from the navigation bar.
    fn navigation_prompt_from_gui(&mut self, prompt: Option<&str>) -> Result<bool, String> {
        Err(format!(
            "navigation prompt {prompt:?}: expected a status owner"
        ))
    }
    /// The edges of the face on a line of sight; example: for Zoom Selected.
    fn face_edges_on_line(&mut self, line: [[f64; 3]; 2]) -> Result<Vec<[[f64; 3]; 2]>, String> {
        Err(format!("view line {line:?}: expected a selection owner"))
    }
    /// A click on the view chooses the face on its line of sight, or nothing.
    /// Example: `work.click_model_from_gui(document, window.view_line(pixel)?)?`.
    fn click_model_from_gui(&mut self, document: u64, line: [[f64; 3]; 2]) -> Result<bool, String> {
        Err(format!(
            "view line {line:?}, document {document}: expected a selection owner"
        ))
    }
    /// Choose a viewport candidate with a screen-derived depth window.
    /// Example: `work.click_model_with_window(document, ray, 0.1)?`.
    fn click_model_with_window(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        _window: f64,
    ) -> Result<bool, String> {
        self.click_model_from_gui(document, line)
    }
    /// The chosen row's outline, in model space; example: to draw it over the view.
    fn selected_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
        Vec::new()
    }
    /// Observe the current drawable before admitting API capture requests.
    /// Example: `work.observe_window_capture_extent(Some([1440, 900]))` before polling.
    fn observe_window_capture_extent(&mut self, _: Option<[u32; 2]>) {}
    /// Transfer bounded capture work to the native window; example: `work.take_window_capture_requests()`.
    fn take_window_capture_requests(&mut self) -> Vec<crate::WindowCaptureJob> {
        Vec::new()
    }
    /// Create a part through the document owner; example: `work.create_part_document()?`.
    fn create_part_document(&mut self) -> Result<bool, String> {
        Err("New Part: expected a document owner with part creation support".into())
    }
    /// Create a selected document template; example: `work.create_document_template(GuiDocumentTemplate::Metric)?`.
    fn create_document_template(
        &mut self,
        template: oblikovati_gui::GuiDocumentTemplate,
    ) -> Result<bool, String> {
        Err(format!(
            "document template {template:?}: expected a document owner with template creation support"
        ))
    }
    /// Count bounded frames only after startup is complete; example: `work.startup_ready()` before smoke accounting.
    fn startup_ready(&self) -> bool {
        true
    }
    /// Complete the requesting document's path choice; example: `work.complete_save_path(id, Some(path))?`.
    fn complete_save_path(
        &mut self,
        document: u64,
        selected: Option<&std::path::Path>,
    ) -> Result<bool, String> {
        if selected.is_none() {
            return Ok(false);
        }
        Err(format!(
            "save path for document {document}, selection {selected:?}: expected a document-save owner"
        ))
    }
    /// Copy bounded command metadata on the owner; example: `work.gui_command_state()?` after polling.
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        Ok(Default::default())
    }
    metadata_work_methods!();
    /// Copy the current parameter editor view.
    /// Example: `work.gui_parameter_snapshot()?` before drawing the panel.
    fn gui_parameter_snapshot(
        &self,
    ) -> Result<Option<oblikovati_gui::GuiParameterSnapshot>, String> {
        Ok(None)
    }
    /// Apply an editor action through the document owner.
    /// Example: `work.apply_gui_parameter(&action)?`.
    fn apply_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
    ) -> Result<bool, String> {
        Err(format!(
            "GUI parameter action {action:?}: expected a parameter-edit owner"
        ))
    }
    /// Apply a shell action through an owner that supports it.
    /// Example: `work.apply_gui_shell_action(action)?`.
    fn apply_gui_shell_action(
        &mut self,
        action: oblikovati_gui::GuiShellAction,
    ) -> Result<bool, String> {
        Err(format!(
            "GUI shell action {action:?}: expected an owner that supports this legacy route"
        ))
    }
    /// Apply native commands on the document owner; example: `work.process_inputs(&events)?` before polling.
    fn process_inputs(&mut self, _: &[oblikovati_viewport::ViewportEvent]) -> Result<bool, String> {
        Ok(false)
    }
    /// Route owner commands with the current window projection; example: `work.process_window_inputs(&events, window)?`.
    fn process_window_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
        _: &dyn crate::HeadWindow,
    ) -> Result<bool, String> {
        self.process_inputs(inputs)
    }
    /// Consume the latest command result for display; example: `work.take_command_notice()` after Save.
    fn take_command_notice(&mut self) -> Option<crate::DesktopCommandNotice> {
        None
    }
    /// Consume a complete scene update after all owner services; example: `work.take_scene()?` before presentation.
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        Ok(None)
    }
    /// Drain one bounded request batch; example: `work.poll(32)?` before presentation.
    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String>;
    /// Poll owner work that requires the window.
    /// Example: `work.poll_window(32, window)?`.
    fn poll_window(
        &mut self,
        budget: usize,
        _: &mut dyn crate::HeadWindow,
    ) -> Result<HeadWorkStatus, String> {
        self.poll(budget)
    }
    /// Check close without discarding documents; example: `work.close_allowed()?` after a native close request.
    fn close_allowed(&mut self) -> Result<bool, String>;
    /// Stop admission before window retirement; example: `work.finish()?` after the owner loop ends.
    fn finish(&mut self) -> Result<(), String>;
}
