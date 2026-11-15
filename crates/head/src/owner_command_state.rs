// SPDX-License-Identifier: GPL-2.0-only
//! Everything the GUI draws from in one frame, read from the owner's documents and settings.
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn gui_command_snapshot(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        let theme = self.gui_theme_snapshot();
        let color_scheme = Some(oblikovati_gui::GuiColorSchemeState::from_registry(
            &self.color_schemes,
        )?);
        let (template_sizes, template_size_issue) = document_creation::template_size_snapshot();
        let template_materials = document_creation::gui_template_materials(&self.files);
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(oblikovati_gui::GuiCommandState {
                orientation_selection_locked: self.orientation_lock,
                marking_menus: Some(self.marking_menus.clone()),
                keymap: Some(self.keymap.clone()),
                keymap_bindings: self.keymap.catalog(),
                ui_scale: self.gui_ui_scale(),
                sketch_options: Some(self.sketch_options.clone()),
                recent_documents: self.gui_recent_rows(),
                theme,
                color_scheme,
                projects: self.gui_projects(),
                template_sizes,
                template_size_issue,
                template_materials,
                ..Default::default()
            });
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        Ok(oblikovati_gui::GuiCommandState {
            selection_mode: self.selection_mode,
            histories: self.gui_histories()?,
            selection_filter: Some(selection_filter::gui_selection_filter(
                &self.ambient_selection,
            )?),
            marking_menus: Some(self.marking_menus.clone()),
            sketch_marking_menu: Some(
                self.marking_menus
                    .get(oblikovati_api::types::SKETCH_ENVIRONMENT),
            ),
            sketch_classic_menu: self.classic_marking_menu,
            keymap: Some(self.keymap.clone()),
            keymap_bindings: self.keymap.catalog(),
            document_units: Some(
                oblikovati_api::client::Client::new(oblikovati_automation::FileQueryHost::new(
                    &self.files,
                ))
                .documents()
                .get_units()
                .map_err(|cause| cause.to_string())?,
            ),
            ui_scale: self.gui_ui_scale(),
            sketch_options: Some(self.sketch_options.clone()),
            // Projected by the frame, which owns the camera.
            constraint_glyphs: Vec::new(),
            recent_documents: self.gui_recent_rows(),
            theme,
            color_scheme,
            dimension_draft: self.gui_dimension_draft()?,
            sketch_formats: self.sketch_formats.clone(),
            sketch_format_available: self.gui_sketch_format_available(),
            dimension_labels: Vec::new(),
            sketch_delete_available: self.sketch_delete_available(document),
            sketch_selection: browser_selection::gui_sketch_selection(session),
            sketch_block_names: session
                .snapshot()
                .sketches
                .block_definitions()
                .iter()
                .map(|definition| definition.name().to_owned())
                .collect(),
            sketch_block_created: self.block_created.clone(),
            sketch_pattern_created: self.pattern_result,
            sketch_pattern_available: true,
            active_sketch: self
                .active_sketch_edit()
                .filter(|edit| edit.document == document)
                .map(|edit| edit.sketch),
            documents: self.gui_document_rows()?,
            document_model_states: self.gui_document_model_states()?,
            document_storage: self.gui_document_storage()?,
            drawing_documents: self.gui_drawing_documents()?,
            drawing_projections: self.gui_drawing_projections()?,
            drawing_updates_deferred: self
                .files
                .deferred_drawing_views(document)
                .map_err(|cause| cause.to_string())?
                .is_some(),
            projects: self.gui_projects(),
            template_sizes,
            template_size_issue,
            template_materials,
            prompts: self.prompt_frequencies,
            browser: Some(self.gui_browser_snapshot(document)?),
            selected_body_key: session
                .selected_browser_body()
                .map(|(key, _)| key.to_owned()),
            selected_body_keys: session
                .selected_browser_bodies()
                .iter()
                .map(|body| body.key.clone())
                .collect(),
            document: Some(document),
            document_type: Some(
                self.files
                    .document_type(document)
                    .map_err(|cause| cause.to_string())?,
            ),
            parameters: self.gui_parameter_rows()?,
            document_properties: session
                .snapshot()
                .properties
                .sets()
                .iter()
                .flat_map(|set| {
                    set.properties().iter().filter_map(|property| {
                        session
                            .snapshot()
                            .properties
                            .info(&set.name, &property.name)
                    })
                })
                .collect(),
            authoring: self.gui_authoring_context(document)?,
            undo_label: session.undo_label().map(str::to_owned),
            redo_label: session.redo_label().map(str::to_owned),
            undo_labels: session
                .undo_labels()
                .iter()
                .map(|&label| label.to_owned())
                .collect(),
            transaction_open: session.in_transaction(),
            extrude_available: self.extrusion_access.is_some(),
            extrude_distance_validation: self
                .extrusion_distance_validation
                .as_ref()
                .filter(|(generation, validation)| {
                    *generation == session.generation() && validation.document == document
                })
                .map(|(_, validation)| validation.clone()),
            extrude_preview_active: self.extrusion_preview.as_ref().is_some_and(
                |(id, generation, _, _)| *id == document && *generation == session.generation(),
            ),
            sketch_tool_keyboard: self
                .placement_state()
                .is_some_and(|tool| self.placement_context_valid(tool)),
            dimension_tool_active: self.dimension_tool_active(),
            takes_command_alias: self
                .placement_state()
                .is_some_and(|tool| tool.takes_command_alias()),
            sketch_alias: (!self.sketch_alias.is_empty()).then(|| self.sketch_alias.clone()),
            dimensions_needed: self.gui_dimensions_needed(document),
            sketch_move: self.gui_sketch_move_stage(),
            sketch_at_ready: self.sketch_tool.is_none() && self.active_sketch_edit().is_some(),
            // Projected by the frame, which owns the camera.
            origin_planes: Vec::new(),
            feature_edges: Vec::new(),
            origin_axes: Vec::new(),
            view_axes: None,
            inspected_file: self.inspected_file.clone(),
            model_face_chosen: self.chosen_model_face(document).is_some(),
            orientation_selection_available: self.chosen_model_face(document).is_some()
                || session.selected_browser_body().is_some(),
            orientation_selection_locked: self.orientation_lock,
            document_windows: self.document_windows.clone(),
            active_window: Some(self.active_view_slot()),
            measure: self.gui_measure_panel(),
            region_properties: self.gui_region_panel(),
            sketch_hud: self.gui_sketch_hud(),
            precise_target: self.gui_precise_target(),
            center_point_format: self.sketch_memory.center_point,
            sketch_context: self.gui_sketch_context()?,
            repeat_command: self.sketch_memory.repeat_command,
            line_command: self.gui_line_command_menu(),
        })
    }
    fn gui_drawing_documents(
        &self,
    ) -> Result<std::collections::BTreeMap<u64, oblikovati_model::drawing::DrawingDocument>, String>
    {
        let workspace = self.files.workspace();
        let mut drawings = std::collections::BTreeMap::new();
        for &id in workspace.documents() {
            let session = workspace.session(id).map_err(|cause| cause.to_string())?;
            if let Some(drawing) = &session.snapshot().drawing {
                drawings.insert(id, drawing.clone());
            }
        }
        Ok(drawings)
    }
}
