// SPDX-License-Identifier: GPL-2.0-only
//! Sketch editing services called by the desktop frame loop.
/// Route sketch editing to its owner; example: `work.edit_sketch_from_gui(true)?`.
pub trait HeadSketchWork {
    /// Save accepted Part options; example: `work.set_part_options_from_gui(options)`.
    fn set_part_options_from_gui(
        &mut self,
        options: oblikovati_model::application_options::PartApplicationOptions,
    ) -> Result<bool, String> {
        Err(format!(
            "part options {options:?}: expected an application preference owner"
        ))
    }
    /// Consume an alias input after the GUI releases it; example: `work.accept_sketch_alias(event)`.
    fn accept_sketch_alias(
        &mut self,
        _: oblikovati_viewport::ViewportEvent,
    ) -> (bool, Option<oblikovati_gui::GuiShellAction>) {
        (false, None)
    }
    /// Start repeated block placement; example: `work.place_sketch_block_from_gui("Rib")?`.
    fn place_sketch_block_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "block {name:?}: expected an active sketch placement owner"
        ))
    }
    /// Group the selected sketch geometry; example: `work.create_sketch_block_from_gui("Block1")?`.
    fn create_sketch_block_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "block name {name:?}: expected an active sketch owner"
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
    /// Start on an authored work plane; example: `work.start_sketch_on_work_plane_from_gui(id)`.
    fn start_sketch_on_work_plane_from_gui(
        &mut self,
        plane: oblikovati_model::feature::FeatureId,
    ) -> Result<bool, String> {
        Err(format!(
            "work plane {plane:?}: expected a document owner with sketches"
        ))
    }
    /// Start on the face under a view ray, or return false when it hits nothing.
    /// Example: `work.start_sketch_on_face_from_gui(document, window.view_line(pixel)?)?`.
    fn start_sketch_on_face_from_gui(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<bool, String> {
        Err(format!(
            "document {document}, face ray {line:?}: expected a sketch support owner"
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
    /// Delete one context relation; example: `work.delete_sketch_relation_from_gui(1)?`.
    fn delete_sketch_relation_from_gui(&mut self, id: u64) -> Result<bool, String> {
        Err(format!(
            "sketch relation {id}: expected an active sketch owner"
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
    /// Convert selected curves or arm a creation mode; example: `work.toggle_sketch_format_from_gui(SketchCurveFormat::Construction)?`.
    fn toggle_sketch_format_from_gui(
        &mut self,
        format: oblikovati_model::sketch::SketchCurveFormat,
    ) -> Result<bool, String> {
        Err(format!(
            "Format {format:?}: expected a document editor owner"
        ))
    }
    /// Preview a rectangular sketch pattern; example: `work.preview_sketch_pattern_from_gui(None)?`.
    fn preview_sketch_pattern_from_gui(
        &mut self,
        request: Option<oblikovati_model::history::SketchRectangularPatternRequest>,
    ) -> Result<bool, String> {
        Err(format!(
            "pattern preview {request:?}: expected an active sketch owner"
        ))
    }
    /// Create a rectangular sketch pattern; example: `work.create_sketch_pattern_from_gui(request)?`.
    fn create_sketch_pattern_from_gui(
        &mut self,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        Err(format!(
            "pattern {request:?}: expected an active sketch owner"
        ))
    }
    /// Preview a recorded pattern edit; example: `work.preview_sketch_pattern_edit_from_gui(pattern, request)?`.
    fn preview_sketch_pattern_edit_from_gui(
        &mut self,
        pattern: oblikovati_model::sketch::SketchEntityId,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        Err(format!(
            "pattern {pattern:?} preview {request:?}: expected an active sketch owner"
        ))
    }
    /// Edit a recorded pattern; example: `work.edit_sketch_pattern_from_gui(pattern, request)?`.
    fn edit_sketch_pattern_from_gui(
        &mut self,
        pattern: oblikovati_model::sketch::SketchEntityId,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        Err(format!(
            "pattern {pattern:?} edit {request:?}: expected an active sketch owner"
        ))
    }
    /// Start or finish the sketch Move tool; example: `work.sketch_move_from_gui(action)?`.
    fn sketch_move_from_gui(
        &mut self,
        action: oblikovati_gui::GuiSketchMoveAction,
    ) -> Result<bool, String> {
        Err(format!(
            "sketch move {action:?}: expected an active sketch owner"
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
}
