// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> crate::HeadSketchWork for DesktopOwner<O, W, F, G, R, S>
{
    fn sketch_tool_candidate(
        &mut self,
        pixel: [i32; 2],
        window: &dyn crate::HeadWindow,
    ) -> Result<Option<oblikovati_viewport::ViewportSketchHover>, String> {
        self.resolve_tool_candidate(pixel, window)
    }
    fn set_part_options_from_gui(
        &mut self,
        options: oblikovati_model::application_options::PartApplicationOptions,
    ) -> Result<bool, String> {
        let store = self.part_store.as_mut().ok_or_else(|| {
            format!("part options {options:?}: expected an application preference owner")
        })?;
        if store.part_options() == options {
            return Ok(false);
        }
        store.set_part_options(options)?;
        Ok(true)
    }
    fn accept_sketch_alias(
        &mut self,
        event: oblikovati_viewport::ViewportEvent,
    ) -> (bool, Option<oblikovati_gui::GuiShellAction>) {
        self.route_sketch_alias(event)
    }
    fn set_sketch_options_from_gui(
        &mut self,
        options: oblikovati_model::application_options::SketchApplicationOptions,
    ) -> Result<bool, String> {
        self.change_sketch_options(options)?;
        Ok(true)
    }
    fn edit_dimension_value_from_gui(
        &mut self,
        value: &oblikovati_gui::GuiDimensionValue,
    ) -> Result<bool, String> {
        self.edit_dimension_value(value)
    }
    fn start_sketch_on_plane_from_gui(
        &mut self,
        plane: oblikovati_gui::GuiOriginPlaneKind,
    ) -> Result<bool, String> {
        use oblikovati_model::feature::work_plane::WorkPlaneOrigin;
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or_else(|| format!("{}: expected an active part document", plane.label()))?;
        let origin = match plane {
            oblikovati_gui::GuiOriginPlaneKind::Xy => WorkPlaneOrigin::XY,
            oblikovati_gui::GuiOriginPlaneKind::Xz => WorkPlaneOrigin::XZ,
            oblikovati_gui::GuiOriginPlaneKind::Yz => WorkPlaneOrigin::YZ,
        };
        self.create_automatic_sketch_for_edit(
            document,
            &oblikovati_model::history::DocumentSketchCreation {
                name: String::new(),
                placement: oblikovati_model::document_sketch::SketchPlacement::OriginPlane(origin),
                orientation: None,
                project_origin: true,
            },
        )?;
        Ok(true)
    }
    fn start_sketch_on_work_plane_from_gui(
        &mut self,
        plane: oblikovati_model::feature::FeatureId,
    ) -> Result<bool, String> {
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or_else(|| format!("work plane {plane:?}: expected an active part document"))?;
        self.create_automatic_sketch_for_edit(
            document,
            &oblikovati_model::history::DocumentSketchCreation {
                name: String::new(),
                placement: oblikovati_model::document_sketch::SketchPlacement::WorkPlane(plane),
                orientation: None,
                project_origin: true,
            },
        )?;
        Ok(true)
    }
    fn start_sketch_on_face_from_gui(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<bool, String> {
        self.start_sketch_on_face_line(document, line)
    }
    fn precise_point_from_gui(
        &mut self,
        point: &oblikovati_gui::GuiPrecisePoint,
    ) -> Result<bool, String> {
        DesktopOwner::precise_point_from_gui(self, point)
    }
    fn sketch_constraint_marks(&self) -> Vec<crate::HeadConstraintMark> {
        DesktopOwner::sketch_constraint_marks(self)
    }
    fn precise_triad_from_gui(
        &mut self,
        command: oblikovati_gui::GuiTriadCommand,
    ) -> Result<bool, String> {
        DesktopOwner::precise_triad_from_gui(self, command)
    }
    fn precise_triad_position(&self) -> Option<(oblikovati_model::sketch::SketchPlane, [f64; 2])> {
        DesktopOwner::precise_triad_position(self)
    }
    fn toggle_center_point_format_from_gui(&mut self) -> Result<bool, String> {
        DesktopOwner::toggle_center_point_format_from_gui(self)
    }
    fn delete_sketch_relation_from_gui(&mut self, id: u64) -> Result<bool, String> {
        self.delete_context_relation(id)
    }
    fn set_center_point_from_gui(&mut self, point: u64, center: bool) -> Result<bool, String> {
        DesktopOwner::set_center_point_from_gui(self, point, center)
    }
    fn line_command_from_gui(
        &mut self,
        command: oblikovati_gui::GuiLineCommand,
    ) -> Result<bool, String> {
        DesktopOwner::line_command_from_gui(self, command)
    }
    fn select_browser_sketch_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        toggle: bool,
    ) -> Result<bool, String> {
        self.select_browser_sketch(document, sketch, toggle)
    }
    fn set_sketch_line_style_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        style: &oblikovati_gui::GuiSketchLineStyle,
    ) -> Result<bool, String> {
        self.set_sketch_line_style(document, sketch, style)
    }
    fn set_sketch_visibility_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        visible: bool,
    ) -> Result<bool, String> {
        self.set_browser_sketch_visibility(document, sketch, visible)
    }
    fn edit_existing_sketch_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<bool, String> {
        self.begin_sketch_edit(document, sketch)?;
        Ok(true)
    }
    fn sketch_dimension_annotations(&self) -> Result<crate::HeadDimensionAnnotations, String> {
        self.dimension_annotations()
    }
    fn dimension_tool_from_gui(
        &mut self,
        action: &oblikovati_gui::GuiDimensionAction,
    ) -> Result<bool, String> {
        self.apply_dimension_intent(action)
    }
    fn toggle_sketch_format_from_gui(
        &mut self,
        format: oblikovati_model::sketch::SketchCurveFormat,
    ) -> Result<bool, String> {
        self.toggle_gui_sketch_format(format)
    }
    fn place_sketch_block_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.start_block_placement(name)
    }
    fn preview_sketch_pattern_from_gui(
        &mut self,
        request: Option<oblikovati_model::history::SketchRectangularPatternRequest>,
    ) -> Result<bool, String> {
        self.preview_gui_pattern(request)
    }
    fn create_sketch_pattern_from_gui(
        &mut self,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        self.create_gui_pattern(request)
    }
    fn preview_sketch_pattern_edit_from_gui(
        &mut self,
        pattern: oblikovati_model::sketch::SketchEntityId,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        self.preview_gui_pattern_edit(pattern, request)
    }
    fn edit_sketch_pattern_from_gui(
        &mut self,
        pattern: oblikovati_model::sketch::SketchEntityId,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        self.edit_gui_pattern(pattern, request)
    }
    fn sketch_move_from_gui(
        &mut self,
        action: oblikovati_gui::GuiSketchMoveAction,
    ) -> Result<bool, String> {
        self.sketch_move_command(action)
    }
    fn delete_sketch_selection_from_gui(&mut self) -> Result<bool, String> {
        self.delete_gui_sketch_selection()
    }
    fn create_sketch_block_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.create_selected_sketch_block(name)
    }
    fn constrain_sketch_from_gui(
        &mut self,
        choice: oblikovati_gui::GuiSketchConstraint,
    ) -> Result<bool, String> {
        let started = self.start_geometric_tool(choice)?;
        if started {
            self.sketch_memory.repeat_command =
                Some(oblikovati_gui::GuiSketchRepeat::Constraint(choice));
        }
        Ok(started)
    }
    fn place_sketch_tool_from_gui(
        &mut self,
        tool: oblikovati_gui::GuiSketchTool,
    ) -> Result<bool, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("{tool:?}: expected an active sketch edit"))?;
        let scale = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .parameters
            .units()
            .working_scale();
        self.start_sketch_placement(edit.document, edit.sketch, gui_placement_tool(tool, scale)?)?;
        self.sketch_memory.repeat_command = Some(oblikovati_gui::GuiSketchRepeat::Placement(tool));
        Ok(true)
    }
    fn edit_sketch_from_gui(&mut self, start: bool) -> Result<bool, String> {
        if !start {
            self.finish_sketch_edit()?;
            return Ok(true);
        }
        self.start_sketch_on_plane_from_gui(oblikovati_gui::GuiOriginPlaneKind::Xy)
    }
    fn decline_sketch_edit(&mut self, cause: String) -> Result<(), String> {
        self.sketch_edit = None;
        self.retire_sketch_tool()?;
        self.command_notice = Some(DesktopCommandNotice::SketchEditFailed { cause });
        Ok(())
    }
    fn sketch_edit(&self) -> Option<crate::HeadSketchEdit> {
        self.active_sketch_edit()
    }
    fn sketch_preview(&self) -> Option<oblikovati_viewport::ViewportSketchPreview> {
        if let Some((edit, generation, preview)) = &self.pattern_preview
            && self.sketch_tool.is_none()
            && self.sketch_context_valid(edit.document, edit.sketch, *generation)
        {
            return Some(preview.clone());
        }

        if let Some(crate::sketch_tool::SketchToolState::Move(tool)) = &self.sketch_tool {
            let available =
                self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation)
                    && self
                        .files
                        .workspace()
                        .session(tool.edit.document)
                        .is_ok_and(|session| !session.in_transaction());
            return available.then(|| tool.preview.clone()).flatten();
        }
        if let Some(crate::sketch_tool::SketchToolState::Block(tool)) = &self.sketch_tool {
            return self
                .sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation)
                .then(|| tool.preview.clone())
                .flatten();
        }
        let recipe = self.placement_preview()?.clone();
        let tool = self.placement_state()?;
        if matches!(
            tool.tool,
            crate::SketchPlacementTool::Line | crate::SketchPlacementTool::Point
        ) || tool.fields_engaged()
        {
            return tool.resolved_preview.clone();
        }
        let session = self.files.workspace().session(tool.document).ok()?;
        Some(oblikovati_viewport::ViewportSketchPreview::Recipe {
            solve_options: session.sketch_solve_options(),
            plane: tool.plane,
            recipe,
        })
    }
    fn take_sketch_scene(&mut self) -> Result<Option<crate::HeadSketchSceneUpdate>, String> {
        let opened_hidden = self.sketch_opened_while_hidden();
        self.sketch_scenes
            .refresh(&mut self.files, &mut self.part_scenes, opened_hidden)
    }
}
