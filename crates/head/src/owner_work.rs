// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> HeadWork for DesktopOwner<O, W, F, G, R, S>
{
    fn observe_window_capture_extent(&mut self, extent: Option<[u32; 2]>) {
        if let Some(captures) = &mut self.window_captures {
            captures.observe_extent(extent);
        }
    }
    fn take_window_capture_requests(&mut self) -> Vec<crate::WindowCaptureJob> {
        self.window_captures
            .as_mut()
            .map_or_else(Vec::new, |captures| captures.take_pending())
    }
    fn complete_save_path(
        &mut self,
        document: u64,
        selected: Option<&std::path::Path>,
    ) -> Result<bool, String> {
        Ok(self.complete_document_save_path(document, selected))
    }
    fn select_color_scheme_from_gui(
        &mut self,
        request: &oblikovati_api::wire::SetColorSchemeArgs,
    ) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .color_schemes()
            .set_active(request)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn edit_theme_color_from_gui(
        &mut self,
        name: &str,
        token: &oblikovati_api::types::ThemeToken,
        value: oblikovati_api::types::Rgba,
    ) -> Result<bool, String> {
        self.preview_custom_theme_color(name, token, value)
    }
    fn duplicate_theme_from_gui(&mut self, base: &str, name: &str) -> Result<bool, String> {
        self.duplicate_application_theme(base, name)
    }
    fn delete_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.delete_application_theme(name)
    }
    fn save_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.persist_active_custom_theme(name)
    }
    fn select_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.select_application_theme(name)
    }
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        let theme = self.gui_theme_snapshot();
        let color_scheme = Some(oblikovati_gui::GuiColorSchemeState::from_registry(
            &self.color_schemes,
        )?);
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(oblikovati_gui::GuiCommandState {
                theme,
                color_scheme,
                ..Default::default()
            });
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        Ok(oblikovati_gui::GuiCommandState {
            theme,
            color_scheme,
            dimension_draft: self.gui_dimension_draft()?,
            dimension_labels: Vec::new(),
            sketch_selection: oblikovati_viewport::ViewportSketchSelection {
                sketches: session.selected_browser_sketches(),
                entities: session
                    .selected_editor_picks()
                    .iter()
                    .filter_map(|pick| match pick.target() {
                        oblikovati_session::selection::SketchPickTarget::Entity(entity) => {
                            Some((pick.sketch(), entity))
                        }
                        _ => None,
                    })
                    .collect(),
            },
            documents: self.gui_document_rows()?,
            browser: Some(self.gui_browser_snapshot(document)?),
            document: Some(document),
            document_type: Some(
                self.files
                    .document_type(document)
                    .map_err(|cause| cause.to_string())?,
            ),
            parameters: self.gui_parameter_rows()?,
            authoring: self.gui_authoring_context(document)?,
            undo_label: session.undo_label().map(str::to_owned),
            redo_label: session.redo_label().map(str::to_owned),
            transaction_open: session.in_transaction(),
            sketch_tool_transaction: self.placement_state().is_some_and(|tool| {
                tool.tool == crate::SketchPlacementTool::Line && self.line_scope_context_valid(tool)
            }),
        })
    }
    fn edit_parameter_from_gui(
        &mut self,
        mutation: oblikovati_gui::GuiParameterMutation,
        request: &oblikovati_api::wire::ParameterSetArgs,
    ) -> Result<bool, String> {
        let client = oblikovati_api::client::Client::new(self.document_host());
        match mutation {
            oblikovati_gui::GuiParameterMutation::Add => client.parameters().add(request),
            oblikovati_gui::GuiParameterMutation::Set => client.parameters().set(request),
        }
        .map_err(|cause| format!("Parameter {:?}: {cause}", request.name))?;
        Ok(true)
    }
    fn activate_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::ActivateDocumentArgs,
    ) -> Result<bool, String> {
        self.activate_gui_document(request)
    }
    fn create_part_document(&mut self) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .create(&oblikovati_api::wire::CreateDocumentArgs {
                type_: "part".into(),
                name: "Part".into(),
                ..Default::default()
            })
            .map_err(|cause| format!("New Part: expected a created part document: {cause}"))?;
        Ok(true)
    }
    fn select_browser_sketch_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        toggle: bool,
    ) -> Result<bool, String> {
        self.select_browser_sketch(document, sketch, toggle)
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
    fn open_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::OpenDocumentArgs,
    ) -> Result<bool, String> {
        let opened = oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .open(request)
            .map_err(|cause| cause.to_string())?;
        self.command_notice = Some(DesktopCommandNotice::DocumentOpened {
            document: opened.id,
            path: request.full_document_name.clone(),
        });
        Ok(true)
    }
    fn close_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::CloseDocumentArgs,
    ) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .close(request)
            .map_err(|cause| cause.to_string())?;
        self.command_notice = Some(DesktopCommandNotice::DocumentClosed {
            document: request.id,
        });
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
    fn constrain_sketch_from_gui(
        &mut self,
        choice: oblikovati_gui::GuiSketchConstraint,
    ) -> Result<bool, String> {
        self.start_geometric_tool(choice)
    }
    fn place_sketch_tool_from_gui(
        &mut self,
        tool: oblikovati_gui::GuiSketchTool,
    ) -> Result<bool, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("{tool:?}: expected an active sketch edit"))?;
        self.start_sketch_placement(edit.document, edit.sketch, gui_placement_tool(tool))?;
        Ok(true)
    }
    fn edit_sketch_from_gui(&mut self, start: bool) -> Result<bool, String> {
        if !start {
            self.finish_sketch_edit()?;
            return Ok(true);
        }
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or("Sketch XY: expected an active part document")?;
        self.create_sketch_for_edit(
            document,
            &oblikovati_model::history::DocumentSketchCreation {
                name: "Sketch".into(),
                placement: oblikovati_model::document_sketch::SketchPlacement::Fixed(
                    oblikovati_model::sketch::SketchPlane::xy(),
                ),
                orientation: None,
                project_origin: true,
            },
        )?;
        Ok(true)
    }
    fn process_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
    ) -> Result<bool, String> {
        self.process_document_inputs(inputs)
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
        let recipe = self.placement_preview()?.clone();
        let tool = self.placement_state()?;
        if tool.tool == crate::SketchPlacementTool::Line || tool.fields_engaged() {
            return tool.resolved_preview.clone();
        }
        let session = self.files.workspace().session(tool.document).ok()?;
        Some(oblikovati_viewport::ViewportSketchPreview::Recipe {
            solve_options: session.sketch_solve_options(),
            plane: tool.plane,
            recipe,
        })
    }

    fn process_window_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
        window: &dyn crate::HeadWindow,
    ) -> Result<bool, String> {
        self.process_placement_inputs(inputs, window)
    }
    fn take_command_notice(&mut self) -> Option<DesktopCommandNotice> {
        self.command_notice.take()
    }
    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String> {
        if budget == 0 {
            return Err("desktop drain budget 0: expected a positive bounded batch".into());
        }
        let serviced = self.drain_calls(budget)?;
        let changed = self.factory.take_changed() || serviced != 0;
        let changed = self.synchronize_sketch_edit() || changed;
        let pending = self.dispatch.pending().map_err(|error| error.to_string())? != 0;
        Ok(HeadWorkStatus {
            changed,
            pending,
            wake_after: None,
        })
    }
    fn take_sketch_scene(&mut self) -> Result<Option<crate::HeadSketchSceneUpdate>, String> {
        self.sketch_scenes.refresh(&self.files)
    }
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        self.part_scenes.refresh(&mut self.files)
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        self.refused_close = self.close_blockers()?;
        let pending = self.dispatch.pending().map_err(|error| error.to_string())?;
        Ok(self.refused_close.is_empty() && pending == 0)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.retire_sketch_tool()?;
        self.sketch_edit = None;
        self.part_scenes.clear();
        self.sketch_scenes.clear();
        self.dispatch.close();
        Ok(())
    }
}
