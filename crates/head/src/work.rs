// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWorkStatus;
/// Owner-thread services borrow model state only while the window loop polls them.
pub trait HeadWork {
    /// Create a drawing view through the public drawing commands; example: `work.base_view_from_gui(&request)?`.
    fn base_view_from_gui(
        &mut self,
        request: &oblikovati_gui::GuiBaseViewRequest,
    ) -> Result<bool, String> {
        Err(format!(
            "drawing view {request:?}: expected a drawing owner"
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
    /// Save the sketch options; example: `work.set_sketch_options_from_gui(options)`.
    fn set_sketch_options_from_gui(
        &mut self,
        options: oblikovati_model::application_options::SketchApplicationOptions,
    ) -> Result<bool, String> {
        Err(format!(
            "sketch options {options:?}: expected an application preference owner"
        ))
    }
    /// Accept a dimension's new value; example: `work.edit_dimension_value_from_gui(&value)`.
    fn edit_dimension_value_from_gui(
        &mut self,
        value: &oblikovati_gui::GuiDimensionValue,
    ) -> Result<bool, String> {
        Err(format!(
            "dimension value {value:?}: expected a document with an edited sketch"
        ))
    }
    /// Start a sketch on an origin plane; example: `work.start_sketch_on_plane_from_gui(plane)`.
    fn start_sketch_on_plane_from_gui(
        &mut self,
        plane: oblikovati_gui::GuiOriginPlaneKind,
    ) -> Result<bool, String> {
        Err(format!(
            "{}: expected a document owner with sketches",
            plane.label()
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
    /// Place the running command's next point exactly; example: `work.precise_point_from_gui(&point)`.
    fn precise_point_from_gui(
        &mut self,
        point: &oblikovati_gui::GuiPrecisePoint,
    ) -> Result<bool, String> {
        Err(format!(
            "precise point {point:?}: expected a document owner"
        ))
    }
    /// Move or reset the toolbar's triad; example: `work.precise_triad_from_gui(command)`.
    fn precise_triad_from_gui(
        &mut self,
        command: oblikovati_gui::GuiTriadCommand,
    ) -> Result<bool, String> {
        Err(format!("triad {command:?}: expected a document owner"))
    }
    /// Where the toolbar's triad sits; example: `work.precise_triad_position()`.
    fn precise_triad_position(&self) -> Option<(oblikovati_model::sketch::SketchPlane, [f64; 2])> {
        None
    }
    /// Change what new points are; example: `work.toggle_center_point_format_from_gui()`.
    fn toggle_center_point_format_from_gui(&mut self) -> Result<bool, String> {
        Err("center point format: expected a document owner".into())
    }
    /// Change one point's role; example: `work.set_center_point_from_gui(point, true)`.
    fn set_center_point_from_gui(&mut self, point: u64, center: bool) -> Result<bool, String> {
        Err(format!(
            "point {point} center {center}: expected a document owner"
        ))
    }
    /// The edited sketch's shown relation marks; example: `work.sketch_constraint_marks()`.
    fn sketch_constraint_marks(&self) -> Vec<crate::HeadConstraintMark> {
        Vec::new()
    }
    /// Apply one active Line menu choice; example: `work.line_command_from_gui(command)`.
    fn line_command_from_gui(
        &mut self,
        command: oblikovati_gui::GuiLineCommand,
    ) -> Result<bool, String> {
        Err(format!(
            "Line command {command:?}: expected a running Line command"
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
    /// Apply a live custom colour preview; example: `work.edit_theme_color_from_gui(name, token, value)?`.
    fn edit_theme_color_from_gui(
        &mut self,
        name: &str,
        token: &oblikovati_api::types::ThemeToken,
        _value: oblikovati_api::types::Rgba,
    ) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}, token {token:?}: expected an editable application theme owner"
        ))
    }
    /// Copy the active theme; example: `work.duplicate_theme_from_gui("Dark", "Workshop")?`.
    fn duplicate_theme_from_gui(&mut self, base: &str, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {base:?} -> {name:?}: expected an application theme owner"
        ))
    }
    /// Delete an active custom; example: `work.delete_theme_from_gui("Workshop")?`.
    fn delete_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}: expected an application theme owner"
        ))
    }
    /// Save the active custom snapshot; example: `work.save_theme_from_gui(name)?`.
    fn save_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}: expected an application theme storage owner"
        ))
    }
    /// Apply the canonical palette selection; example: `work.select_color_scheme_from_gui(&request)?`.
    fn select_color_scheme_from_gui(
        &mut self,
        request: &oblikovati_api::wire::SetColorSchemeArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "colour scheme {:?}: expected an application palette owner",
            request.name
        ))
    }
    /// Select the application theme; example: `work.select_theme_from_gui("Light")?`.
    fn select_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}: expected an application theme owner"
        ))
    }
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
    /// Select a checked browser row without starting an edit.
    /// Example: `work.select_browser_sketch_from_gui(document, sketch, false)?`.
    fn select_browser_sketch_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        _toggle: bool,
    ) -> Result<bool, String> {
        Err(format!(
            "browser sketch {sketch:?}, document {document}: expected a selection owner"
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
    /// Change how a sketch's lines are drawn; example: from Geometry Properties.
    fn set_sketch_line_style_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        _: &oblikovati_gui::GuiSketchLineStyle,
    ) -> Result<bool, String> {
        Err(format!(
            "sketch {sketch:?}, document {document}: expected a sketch style owner"
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
    /// The chosen row's outline, in model space; example: to draw it over the view.
    fn selected_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
        Vec::new()
    }
    /// Change stored sketch visibility; example: `work.set_sketch_visibility_from_gui(document, sketch, false)?`.
    fn set_sketch_visibility_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        _visible: bool,
    ) -> Result<bool, String> {
        Err(format!(
            "browser sketch {sketch:?}, document {document}: expected a visibility owner"
        ))
    }
    /// Activate a stable sketch selected in the model browser.
    /// Example: `work.edit_existing_sketch_from_gui(document, sketch)?` after a double-click.
    fn edit_existing_sketch_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<bool, String> {
        Err(format!(
            "browser sketch {sketch:?}, document {document}: expected an owner with sketch edit support"
        ))
    }
    /// Observe the current drawable before admitting API capture requests.
    /// Example: `work.observe_window_capture_extent(Some([1440, 900]))` before polling.
    fn observe_window_capture_extent(&mut self, _: Option<[u32; 2]>) {}
    /// Transfer bounded capture work to the native window; example: `work.take_window_capture_requests()`.
    fn take_window_capture_requests(&mut self) -> Vec<crate::WindowCaptureJob> {
        Vec::new()
    }
    /// Convert selected curves or arm a creation mode; example: `work.toggle_sketch_format_from_gui(SketchCurveFormat::Construction)?`.
    fn toggle_sketch_format_from_gui(
        &mut self,
        format: oblikovati_model::sketch::SketchCurveFormat,
    ) -> Result<bool, String> {
        Err(format!(
            "Format {format:?}: expected a document editor owner"
        ))
    }
    /// Delete checked editor picks through their owner; example: `work.delete_sketch_selection_from_gui()?`.
    fn delete_sketch_selection_from_gui(&mut self) -> Result<bool, String> {
        Err("Delete Sketch Selection: expected a document editor owner".into())
    }
    /// Activate a supported placement engine; example: `work.place_sketch_tool_from_gui(GuiSketchTool::Circle)?`.
    fn place_sketch_tool_from_gui(
        &mut self,
        tool: oblikovati_gui::GuiSketchTool,
    ) -> Result<bool, String> {
        Err(format!("{tool:?}: expected an owner with an active sketch"))
    }
    /// Read authored and preview labels without changing history; example: `work.sketch_dimension_annotations()?` before projection.
    fn sketch_dimension_annotations(&self) -> Result<crate::HeadDimensionAnnotations, String> {
        Ok(Default::default())
    }
    /// Route a dimension intent to its checked owner draft; example: `work.dimension_tool_from_gui(&GuiDimensionAction::Start)?`.
    fn dimension_tool_from_gui(
        &mut self,
        action: &oblikovati_gui::GuiDimensionAction,
    ) -> Result<bool, String> {
        Err(format!(
            "dimension action {action:?}: expected an owner with an active sketch"
        ))
    }
    /// Start a checked geometric constraint tool on the active sketch.
    /// Example: `work.constrain_sketch_from_gui(GuiSketchConstraint::Horizontal)?;`.
    fn constrain_sketch_from_gui(
        &mut self,
        choice: oblikovati_gui::GuiSketchConstraint,
    ) -> Result<bool, String> {
        Err(format!(
            "constraint {choice:?}: expected an owner with an active sketch"
        ))
    }
    /// Enter or finish the owner sketch edit; example: `work.edit_sketch_from_gui(true)?`.
    fn edit_sketch_from_gui(&mut self, start: bool) -> Result<bool, String> {
        Err(format!(
            "GUI sketch start={start}: expected a sketch document owner"
        ))
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
    /// Consume a committed authoring update; example: `work.take_sketch_scene()?` after polling history.
    fn take_sketch_scene(&mut self) -> Result<Option<crate::HeadSketchSceneUpdate>, String> {
        Ok(None)
    }
    /// End an edit whose view could not be prepared; example: `work.decline_sketch_edit(cause)?` after a camera refusal.
    fn decline_sketch_edit(&mut self, cause: String) -> Result<(), String> {
        Err(format!(
            "sketch edit framing {cause}: expected an owner that can cancel the edit"
        ))
    }
    /// Copy the current placed sketch edit; example: `work.sketch_edit()` before camera synchronization.
    fn sketch_edit(&self) -> Option<crate::HeadSketchEdit> {
        None
    }
    /// Copy the current transient authoring graph; example: `work.sketch_preview()` before drawing.
    fn sketch_preview(&self) -> Option<oblikovati_viewport::ViewportSketchPreview> {
        None
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
    /// Apply a parameter editor request on the document owner; example: `work.edit_parameter_from_gui(GuiParameterMutation::Add, &request)?`.
    fn edit_parameter_from_gui(
        &mut self,
        mutation: oblikovati_gui::GuiParameterMutation,
        request: &oblikovati_api::wire::ParameterSetArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "GUI parameter {mutation:?} {request:?}: expected a document parameter owner"
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
    /// Check close without discarding documents; example: `work.close_allowed()?` after a native close request.
    fn close_allowed(&mut self) -> Result<bool, String>;
    /// Stop admission before window retirement; example: `work.finish()?` after the owner loop ends.
    fn finish(&mut self) -> Result<(), String>;
}
