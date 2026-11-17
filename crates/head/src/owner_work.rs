// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[macro_use]
#[path = "owner_work_extrusion.rs"]
mod extrusion_work;
#[macro_use]
#[path = "owner_work_document.rs"]
mod document_work;
#[macro_use]
#[path = "owner_work_selection.rs"]
mod selection_work;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> HeadWork for DesktopOwner<O, W, F, G, R, S>
{
    fn set_orientation_lock_from_gui(&mut self, locked: bool) -> Result<bool, String> {
        self.change_orientation_lock(locked)
    }
    fn orientation_selection_pivot(&mut self) -> Result<Option<[f64; 3]>, String> {
        self.orientation_pivot()
    }
    fn look_at_selection_plane(
        &self,
    ) -> Result<Option<oblikovati_model::sketch::SketchPlane>, String> {
        self.look_at_plane()
    }

    fn set_selection_mode_from_gui(
        &mut self,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<bool, String> {
        self.set_selection_mode(mode)
    }
    fn set_selection_enabled_from_gui(
        &mut self,
        kind: oblikovati_gui::GuiSelectionKind,
        enabled: bool,
    ) -> Result<bool, String> {
        self.set_selection_enabled(kind, enabled)
    }
    fn move_selection_kind_from_gui(&mut self, from: usize, to: usize) -> Result<bool, String> {
        self.move_selection_kind(from, to)
    }
    fn reset_selection_filter_from_gui(&mut self) -> Result<bool, String> {
        self.reset_selection_filter()
    }
    fn jump_history_from_gui(&mut self, document: u64, position: i64) -> Result<bool, String> {
        self.jump_history_action(document, position)
    }
    fn update_part_from_gui(&mut self) -> Result<bool, String> {
        self.update_part_action()
    }
    fn set_marking_menu_from_gui(
        &mut self,
        args: &oblikovati_api::wire::SetMarkingMenuArgs,
    ) -> Result<bool, String> {
        self.set_marking_menu_action(args)
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
        self.set_keymap_action(delta)
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
    desktop_extrusion_work!();
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
    fn clear_selected_face_appearances_from_gui(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<bool, String> {
        self.clear_selected_face_appearances(document, keys)
    }
    fn clear_selected_mixed_appearance_from_gui(
        &mut self,
        document: u64,
        bodies: &[String],
        faces: &[String],
    ) -> Result<bool, String> {
        self.clear_selected_appearances(document, bodies, faces)
    }
    fn clear_selected_body_appearances_from_gui(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<bool, String> {
        self.clear_selected_body_appearances(document, keys)
    }
    fn set_document_property_from_gui(
        &mut self,
        request: &oblikovati_api::wire::SetPropertyArgs,
    ) -> Result<bool, String> {
        self.set_typed_document_property(request)
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
        let previous = self.files.workspace().active_document();
        let name = self.next_new_document_name("Part");
        let created = oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .create(&oblikovati_api::wire::CreateDocumentArgs {
                type_: "part".into(),
                name,
                ..Default::default()
            })
            .map_err(|cause| format!("New Part: expected a created part document: {cause}"))?;
        self.enter_new_part_startup(created.id, previous)?;
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
    desktop_selection_work!();
    desktop_document_work!();
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
        let cue_changed = self.update_model_pick_cue(inputs, window);
        Ok(self.process_placement_inputs(inputs, window)? || cue_changed)
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
        let changed = self.synchronize_model_pick_cue() || changed;
        let pending = self.dispatch.pending().map_err(|error| error.to_string())? != 0;
        Ok(HeadWorkStatus {
            changed,
            pending,
            wake_after: None,
        })
    }
    fn datum_planes(&mut self) -> Result<crate::HeadDatumPlaneUpdate, String> {
        self.datum_planes_for_view()
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
