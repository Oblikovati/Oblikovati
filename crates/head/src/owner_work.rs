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
    fn set_selection_mode_from_gui(
        &mut self,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<bool, String> {
        use oblikovati_gui::GuiSelectionMode;
        use oblikovati_session::selection::SelectionPriority;
        let state = self.gui_command_snapshot()?;
        if !oblikovati_gui::GuiShellAction::SetSelectionMode(mode).enabled(&state) {
            return Err(format!(
                "selection mode {mode:?}: expected an idle part or assembly"
            ));
        }
        let priority = match mode {
            GuiSelectionMode::All => SelectionPriority::General,
            GuiSelectionMode::Faces => SelectionPriority::Face,
            GuiSelectionMode::Components => SelectionPriority::Part,
            GuiSelectionMode::Custom => {
                return Err("selection mode Custom: expected a concrete preset".into());
            }
        };
        self.ambient_selection.apply_priority(priority);
        let changed = self.selection_mode != mode;
        self.selection_mode = mode;
        Ok(changed)
    }
    fn set_selection_enabled_from_gui(
        &mut self,
        kind: oblikovati_gui::GuiSelectionKind,
        enabled: bool,
    ) -> Result<bool, String> {
        let state = self.gui_command_snapshot()?;
        if !(oblikovati_gui::GuiShellAction::SetSelectionEnabled { kind, enabled }).enabled(&state)
        {
            return Err(format!(
                "selection category {kind:?}: expected an idle part or assembly"
            ));
        }
        let kind = super::selection_filter::session_kind(kind);
        let changed = self.ambient_selection.enabled(kind) != enabled;
        self.ambient_selection.set_enabled(kind, enabled);
        if changed {
            self.selection_mode = oblikovati_gui::GuiSelectionMode::Custom;
        }
        Ok(changed)
    }
    fn move_selection_kind_from_gui(&mut self, from: usize, to: usize) -> Result<bool, String> {
        let state = self.gui_command_snapshot()?;
        if !(oblikovati_gui::GuiShellAction::MoveSelectionKind { from, to }).enabled(&state) {
            return Err(format!(
                "selection priority {from}->{to}: expected an idle part or assembly"
            ));
        }
        let count = self.ambient_selection.order().len();
        if from >= count || to >= count {
            return Err(format!(
                "selection priority {from}->{to}: expected indices below {count}"
            ));
        }
        self.ambient_selection.move_kind(from, to);
        Ok(from != to)
    }
    fn jump_history_from_gui(&mut self, document: u64, position: i64) -> Result<bool, String> {
        let client = oblikovati_api::client::Client::new(self.document_host());
        let current = client
            .transactions()
            .history(&oblikovati_api::wire::TransactionHistoryArgs { document })
            .map_err(|cause| cause.to_string())?;
        client
            .transactions()
            .jump_to(&oblikovati_api::wire::TransactionJumpToArgs { document, position })
            .map_err(|cause| cause.to_string())?;
        Ok(current.position != position)
    }
    fn update_part_from_gui(&mut self) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .update(&oblikovati_api::wire::UpdateDocumentArgs::default())
            .map_err(|error| error.to_string())?;
        Ok(true)
    }
    fn set_marking_menu_from_gui(
        &mut self,
        args: &oblikovati_api::wire::SetMarkingMenuArgs,
    ) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .ui()
            .set_marking_menu(args)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn take_shortcut_action(&mut self) -> Option<oblikovati_gui::GuiShellAction> {
        let action = self.shortcut_keys.take_action();
        if action.is_some() {
            self.sketch_alias.clear();
        }
        action
    }
    fn set_keymap_from_gui(
        &mut self,
        delta: &oblikovati_api::wire::KeymapExport,
    ) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .keymap()
            .import(delta)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn resolve_shortcut(
        &mut self,
        event: oblikovati_viewport::ViewportEvent,
    ) -> oblikovati_viewport::ViewportEvent {
        self.shortcut_keys.resolve(&self.keymap, event)
    }
    fn set_document_units_from_gui(
        &mut self,
        document: u64,
        request: &oblikovati_api::wire::SetDocumentUnitsArgs,
    ) -> Result<bool, String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "document {document}: expected the active document for unit settings"
            ));
        }
        oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .set_units(request)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn base_view_preview_from_gui(
        &mut self,
        request: Option<&oblikovati_gui::GuiBaseViewRequest>,
    ) -> Result<bool, String> {
        self.set_drawing_view_preview(request)
    }
    fn base_view_from_gui(
        &mut self,
        request: &oblikovati_gui::GuiBaseViewRequest,
    ) -> Result<bool, String> {
        self.create_drawing_base_view(request)
    }
    fn place_component_from_gui(
        &mut self,
        assembly: u64,
        request: &oblikovati_api::wire::PlaceOccurrenceArgs,
    ) -> Result<bool, String> {
        if self.files.workspace().active_document() != Some(assembly) {
            return Err(format!("assembly {assembly}: expected the active assembly"));
        }
        oblikovati_api::client::Client::new(self.document_host())
            .assembly()
            .place(request)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn place_component_file_from_gui(
        &mut self,
        assembly: u64,
        path: &str,
        name: &str,
    ) -> Result<bool, String> {
        if self.files.workspace().active_document() != Some(assembly) {
            return Err(format!("assembly {assembly}: expected the active assembly"));
        }
        let transform = oblikovati_api::types::Matrix {
            cells: [
                1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
            ],
        };
        self.document_host()
            .place_component_file(std::path::Path::new(path), name, transform)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn preview_component_from_gui(&mut self, request: Option<(u64, u64)>) -> Result<bool, String> {
        self.set_component_preview(request)
    }
    fn preview_component_file_from_gui(
        &mut self,
        assembly: u64,
        path: &str,
    ) -> Result<bool, String> {
        self.set_component_file_preview(assembly, path)
    }
    fn extrude_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: &str,
    ) -> Result<bool, String> {
        self.create_extrusion_from_gui(document, sketch, profile, distance)
    }
    fn extrusion_preview_from_gui(
        &mut self,
        request: Option<&oblikovati_gui::GuiExtrudePreview>,
    ) -> Result<bool, String> {
        let Some(preview) = request else {
            return Ok(self.clear_extrusion_preview());
        };
        self.set_extrusion_preview(
            preview.document,
            preview.sketch,
            preview.profile,
            &preview.distance,
        )
    }
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
    fn set_prompt_frequency_from_gui(
        &mut self,
        prompt: oblikovati_model::application_prompts::ApplicationPrompt,
        frequency: oblikovati_model::application_prompts::PromptFrequency,
    ) -> Result<bool, String> {
        self.change_prompt_frequency(prompt, frequency)
    }
    fn recent_from_gui(
        &mut self,
        action: &oblikovati_gui::GuiRecentAction,
    ) -> Result<bool, String> {
        DesktopOwner::recent_from_gui(self, action)
    }
    fn view_slot(&self) -> u64 {
        self.active_view_slot()
    }
    fn document_window_from_gui(&mut self, id: u64, close: bool) -> Result<bool, String> {
        if close {
            return self.close_document_window(id);
        }
        self.activate_document_window(id)
    }
    fn new_document_window_from_gui(&mut self) -> Result<bool, String> {
        self.new_document_window()
    }
    fn inspect_from_gui(
        &mut self,
        action: oblikovati_gui::GuiInspectAction,
    ) -> Result<bool, String> {
        DesktopOwner::inspect_from_gui(self, action)
    }
    fn set_ui_scale_from_gui(
        &mut self,
        options: oblikovati_model::ui_scale::UiScaleOptions,
    ) -> Result<bool, String> {
        self.persist_ui_scale(options)
    }
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        self.gui_command_snapshot()
    }
    fn gui_parameter_snapshot(
        &self,
    ) -> Result<Option<oblikovati_gui::GuiParameterSnapshot>, String> {
        super::parameters_gui::owner_parameter_snapshot(&self.files)
    }
    fn apply_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
    ) -> Result<bool, String> {
        if let oblikovati_gui::GuiParameterAction::GroupManagement(edit) = action {
            edit.validate()?;
            return self.apply_parameter_group_management(edit);
        }
        if let oblikovati_gui::GuiParameterAction::Link(edit) = action {
            edit.validate()?;
            let (source_document, linked) =
                super::parameters_gui::derived::linked_source_names(&self.files, edit)?;
            oblikovati_api::client::Client::new(self.document_host())
                .parameters()
                .add_derived_table(&oblikovati_api::wire::DerivedParameterTableAddArgs {
                    source_document,
                    linked: Some(linked),
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let oblikovati_gui::GuiParameterAction::DeleteLinkedTable { document, table } = action {
            super::parameters_gui::derived::require_linked_table_owner(
                &self.files,
                *document,
                *table,
            )?;
            oblikovati_api::client::Client::new(self.document_host())
                .parameters()
                .delete_derived_table(&oblikovati_api::wire::DerivedParameterTableDeleteArgs {
                    id: *table,
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        super::parameters_gui::apply::apply_owner_parameter(&mut self.files, action)
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
    fn create_document_template(
        &mut self,
        template: oblikovati_gui::GuiDocumentTemplate,
    ) -> Result<bool, String> {
        self.create_template_document(template)
    }
    fn create_part_document(&mut self) -> Result<bool, String> {
        let name = self.next_new_document_name("Part");
        oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .create(&oblikovati_api::wire::CreateDocumentArgs {
                type_: "part".into(),
                name,
                ..Default::default()
            })
            .map_err(|cause| format!("New Part: expected a created part document: {cause}"))?;
        Ok(true)
    }
    fn select_browser_block_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        instance: oblikovati_model::sketch::SketchEntityId,
        toggle: bool,
    ) -> Result<bool, String> {
        self.select_browser_block(document, sketch, instance, toggle)
    }
    fn select_browser_feature_from_gui(
        &mut self,
        document: u64,
        feature: oblikovati_model::feature::FeatureId,
    ) -> Result<bool, String> {
        self.select_browser_feature(document, feature)
    }
    fn update_body_properties_from_gui(
        &mut self,
        document: u64,
        body: i64,
    ) -> Result<bool, String> {
        self.update_body_physical(document, body)
    }
    fn assign_part_asset_from_gui(
        &mut self,
        document: u64,
        kind: oblikovati_gui::GuiPartAssetKind,
        id: &str,
    ) -> Result<bool, String> {
        self.assign_part_asset(document, kind, id)
    }
    fn set_body_appearance_from_gui(
        &mut self,
        document: u64,
        key: &str,
        appearance: &str,
    ) -> Result<bool, String> {
        self.set_body_appearance(document, key, appearance)
    }
    fn edit_feature_from_gui(
        &mut self,
        document: u64,
        edit: &oblikovati_gui::GuiFeatureEdit,
    ) -> Result<bool, String> {
        self.edit_feature(document, edit)
    }
    fn browser_choice_from_gui(
        &mut self,
        document: u64,
        choice: &oblikovati_gui::GuiBrowserChoice,
    ) -> Result<bool, String> {
        self.browser_choice(document, choice)
    }
    fn keep_document_picture_from_gui(
        &mut self,
        document: u64,
        picture: Option<&oblikovati_gui::GuiPreviewImage>,
    ) -> Result<bool, String> {
        self.keep_document_picture(document, picture);
        Ok(false)
    }
    fn inspect_file_from_gui(&mut self, path: &str) -> Result<bool, String> {
        self.inspect_file(path);
        Ok(true)
    }
    fn navigation_prompt_from_gui(&mut self, prompt: Option<&str>) -> Result<bool, String> {
        self.navigation_prompt(prompt)
    }
    fn face_edges_on_line(&mut self, line: [[f64; 3]; 2]) -> Result<Vec<[[f64; 3]; 2]>, String> {
        self.edges_of_face_on_line(line)
    }
    fn click_model_from_gui(&mut self, document: u64, line: [[f64; 3]; 2]) -> Result<bool, String> {
        self.click_model(document, line)
    }
    fn selected_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
        self.current_browser_outline()
    }
    fn open_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::OpenDocumentArgs,
    ) -> Result<bool, String> {
        let opened = oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .open(request)
            .map_err(|cause| cause.to_string())?;
        self.note_recent_file(&request.full_document_name);
        self.command_notice = Some(DesktopCommandNotice::DocumentOpened {
            document: opened.id,
            path: request.full_document_name.clone(),
        });
        Ok(true)
    }
    fn open_deferred_drawing_from_gui(&mut self, path: &str) -> Result<bool, String> {
        let document = self
            .document_host()
            .open_deferred_drawing_path(std::path::Path::new(path))
            .map_err(|cause| cause.to_string())?;
        self.note_recent_file(path);
        self.command_notice = Some(DesktopCommandNotice::DocumentOpened {
            document,
            path: path.to_owned(),
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
        self.forget_document_windows(request.id);
        self.command_notice = Some(DesktopCommandNotice::DocumentClosed {
            document: request.id,
        });
        Ok(true)
    }
    fn select_block_geometry_from_gui(&mut self, active: bool) -> Result<bool, String> {
        self.set_block_selection(active)
    }
    fn process_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
    ) -> Result<bool, String> {
        self.process_document_inputs(inputs)
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
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        self.take_scene_with_extrusion_preview()
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
