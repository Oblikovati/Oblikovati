// SPDX-License-Identifier: GPL-2.0-only
use super::*;

impl GuiFrameEngine {
    pub(super) fn draw_controls(
        &mut self,
        request: &GuiFrameRequest,
    ) -> Result<GuiFrameOutput, String> {
        self.parameter_editor
            .synchronize_document(request.commands.document)?;
        let keys = prepare_frame_inputs(
            &mut self.context,
            request,
            &mut self.modifiers_held,
            self.popup_was_open,
            self.ribbon.tips.active(),
            &mut self.widget_keys,
        );
        let ui = self.context.frame();
        crate::sketch_pick_cue::draw_sketch_pick_cue(ui, request);
        crate::sketch_pick_cue::draw_model_pick_cue(ui, request);
        self.workspace
            .sync_origin_document(request.commands.document);
        crate::dimension_labels::draw_dimension_labels(ui, request);
        let constraint_hover =
            crate::sketch::constraint_glyphs::draw_constraint_glyphs(ui, request);
        crate::application::workspace_layout::browser_feature::draw_feature_edges(ui, request);
        self.workspace.draw_selected_origin(ui, request);
        let mut selected = Vec::new();
        self.document_files.home_in_front = self.home.active;
        let band_height = crate::chrome::ribbon_band(
            ui,
            request,
            crate::chrome::RibbonDrawOwners {
                home_active: self.home.active,
                icons: &self.icons.presentation,
                ribbon: &mut self.ribbon,
                parameters: &mut self.parameter_editor,
                files: &mut self.document_files,
                theme_editor: &mut self.theme_editor,
                preferences: &mut self.preferences,
                precise_input: &mut self.precise_input,
                create_block: &mut self.create_block,
                sketch_pattern: &mut self.sketch_pattern,
                workspace: &mut self.workspace,
                workspaces: &mut self.workspaces,
                quick_access: &mut self.quick_access,
                plane_selection_active: self.plane_selection.active,
            },
            &mut selected,
        );
        let (window_action, window_controls) = crate::window_frame::draw(
            ui,
            request,
            crate::chrome::application_bar_height(&self.icons.presentation),
            &mut selected,
            &mut self.window_bar_drag_armed,
        );
        if std::mem::take(&mut self.pending_extrusion_shortcut) {
            selected.push(crate::extrude_dialog::start_action(&request.commands));
        }
        apply_quick_access(
            &mut self.quick_access,
            &mut self.document_files,
            (&mut self.plane_selection, &mut self.workspace),
            &mut self.extrude_dialog,
            (&mut self.material_browser, &mut self.appearance_browser),
            (&mut self.parameter_editor, &request.commands),
            &mut selected,
        );
        if !self.ribbon.tips.active() && !self.ribbon.tips.consumed {
            self.command_alias.panel(ui, &request.commands, &keys.typed);
        }
        crate::application::workspace_layout::status_bar::status_bar(
            ui,
            request,
            self.command_alias.text(),
        );
        let over_fields = crate::sketch::line::hud_panel(ui, request);
        self.document_tabs.previews.synchronize(&request.commands);
        self.document_tabs
            .panel(ui, request, &self.workspace, &mut self.home, &mut selected);
        selected.retain(|action| {
            if let GuiShellAction::OpenClearBodyAppearance { document } = action {
                self.clear_appearance.open(*document);
                return false;
            }
            if matches!(action, GuiShellAction::OpenIProperties) {
                self.metadata.open(&request.commands);
                return false;
            }
            if matches!(action, GuiShellAction::OpenHistoryBrowser) {
                self.history_browser.open();
                return false;
            }
            if matches!(action, GuiShellAction::OpenSelectionFilter) {
                self.selection_filter.open();
                return false;
            }
            true
        });
        self.metadata.panel(ui, &request.commands, &mut selected);
        self.clear_appearance
            .panel(ui, &request.commands, &mut selected);
        self.selection_filter
            .draw(ui, &request.commands, &mut selected);
        self.history_browser
            .draw(ui, &request.commands, &mut selected);
        let parameter_actions = self.parameter_editor.panel(ui);
        self.workspace
            .panels
            .record_status(request.status.as_deref());
        self.workspace.view_clicks_taken = self.orientation.owns_view_clicks();
        self.workspace.browser_panel(
            ui,
            &request.commands,
            request.extent[1] as f32 / request.layout_scale()[1]
                - crate::application::workspace_layout::status_bar::STATUS_BAR_HEIGHT
                - self.workspace.tab_height(),
            band_height,
            self.part_context_open,
            &mut selected,
        );
        let open_offset = selected.contains(&GuiShellAction::OpenOffsetPlane);
        selected.retain(|action| !matches!(action, GuiShellAction::OpenOffsetPlane));
        if open_offset || self.offset_plane.active() {
            let base = self
                .workspace
                .take_selected_origin_plane()
                .map(crate::GuiOffsetPlaneBase::Origin)
                .or_else(|| {
                    selected_work_plane(&request.commands).map(crate::GuiOffsetPlaneBase::WorkPlane)
                })
                .or_else(|| {
                    request
                        .commands
                        .model_face_chosen
                        .then_some(crate::GuiOffsetPlaneBase::SelectedFace)
                });
            if open_offset && let Some(document) = request.commands.document {
                self.offset_plane.open(document, base);
            } else if let Some(base) = base {
                self.offset_plane.choose(base);
            }
        }
        for action in &selected {
            if let GuiShellAction::OpenSketchPattern { sketch, pattern } = action {
                self.sketch_pattern
                    .open_edit(&request.commands, *sketch, *pattern);
            }
            if let GuiShellAction::OpenExtrudeEdit(feature) = action {
                self.extrude_dialog.open_edit(&request.commands, *feature);
            }
        }
        selected.retain(|action| !matches!(action, GuiShellAction::OpenSketchPattern { .. }));
        selected.retain(|action| !matches!(action, GuiShellAction::OpenExtrudeEdit(_)));
        let logical =
            std::array::from_fn(|axis| request.extent[axis] as f32 / request.layout_scale()[axis]);
        let area = self.workspace.document_area(logical, band_height);
        // Over Home there is no view for it to turn.
        if !self.home.active {
            self.orientation.panel(ui, request, area, &mut selected);
        }
        let detached = self.document_tabs.is_detached(request.commands.document);
        self.workspaces
            .panel(ui, &request.commands, area, detached, &mut selected);
        self.plane_selection.panel(
            ui,
            &request.commands,
            request.layout_scale(),
            area,
            &mut selected,
        );
        self.document_tabs.previews.showing = workspace_front(request, area, self.home.active);
        self.document_tabs.previews.showing_document =
            request.commands.document.filter(|_| !self.home.active);
        if !self.home.active {
            let mut menu_actions = Vec::new();
            let mut repeat_wheel = self.orientation.can_repeat_wheel().then_some(false);
            self.part_context_open = !self.orientation.navigation_owns_viewport()
                && crate::part_context_menu::canvas_menu(
                    ui,
                    &request.commands,
                    area,
                    &mut menu_actions,
                    &mut repeat_wheel,
                );
            if repeat_wheel == Some(true) {
                self.orientation.repeat_wheel();
            }
            for action in &mut menu_actions {
                if *action == GuiShellAction::OpenExtrude {
                    *action = crate::extrude_dialog::start_action(&request.commands);
                }
            }
            if !menu_actions.is_empty() {
                apply_quick_access(
                    &mut self.quick_access,
                    &mut self.document_files,
                    (&mut self.plane_selection, &mut self.workspace),
                    &mut self.extrude_dialog,
                    (&mut self.material_browser, &mut self.appearance_browser),
                    (&mut self.parameter_editor, &request.commands),
                    &mut menu_actions,
                );
                selected.extend(menu_actions);
            }
            if !self.blend_dialog.active() && !self.part_context_open {
                crate::application::workspace_layout::browser_feature::selection_toolbar(
                    ui,
                    request,
                    &mut selected,
                );
            }
        } else {
            self.part_context_open = false;
        }
        for action in &selected {
            if let GuiShellAction::OpenFilletEdit(feature) = action {
                self.blend_dialog.open_edit(&request.commands, *feature);
            }
        }
        selected.retain(|action| !matches!(action, GuiShellAction::OpenFilletEdit(_)));
        for action in &selected {
            if let GuiShellAction::OpenChamferEdit(feature) = action {
                self.blend_dialog
                    .open_chamfer_edit(&request.commands, *feature);
            }
        }
        selected.retain(|action| !matches!(action, GuiShellAction::OpenChamferEdit(_)));
        if selected.contains(&GuiShellAction::OpenFillet) {
            if self.blend_dialog.active() {
                self.blend_dialog.close(&mut selected);
            }
            self.blend_dialog.open(&request.commands);
            selected.retain(|action| !matches!(action, GuiShellAction::OpenFillet));
        }
        if selected.contains(&GuiShellAction::OpenChamfer) {
            if self.blend_dialog.active() {
                self.blend_dialog.close(&mut selected);
            }
            self.blend_dialog.open_chamfer(&request.commands);
            selected.retain(|action| !matches!(action, GuiShellAction::OpenChamfer));
        }
        self.home.panel(ui, &request.commands, area, &mut selected);
        let home_requests = std::mem::take(&mut self.home.requests);
        if home_requests.new_document {
            self.document_files.request_new();
        }
        if home_requests.open_document {
            self.document_files.request_open();
        }
        if let Some(path) = home_requests.file_properties {
            self.document_tabs.menu.properties.request_file(&path);
        }
        self.document_files.panel(
            ui,
            &request.commands,
            &mut selected,
            &self.icons.presentation,
        );
        self.preferences
            .panel(ui, &request.commands, &mut selected, self.home.active);
        self.material_browser
            .panel(ui, &request.commands, &mut selected);
        self.appearance_browser
            .panel(ui, &request.commands, &mut selected);
        self.extrude_dialog.panel(
            ui,
            &request.commands,
            self.plane_selection.active,
            &mut selected,
        );
        self.blend_dialog
            .panel(ui, &request.commands, &mut selected);
        self.offset_plane
            .panel(ui, &request.commands, &mut selected);
        self.ribbon
            .customization
            .panel(ui, &request.commands, &mut selected);
        self.create_block
            .panel(ui, &request.commands, &mut selected);
        self.sketch_pattern
            .panel(ui, &request.commands, &mut selected);
        crate::sketch::line::command_menu(ui, &request.commands, &mut selected);
        self.precise_input
            .panel(ui, &request.commands, &mut selected);
        crate::sketch::inspect::canvas_menu(
            ui,
            &request.commands,
            &mut selected,
            &mut self.sketch_popup_token,
        );
        self.dimension_value_editor
            .panel(ui, request, &mut selected);
        crate::sketch::move_tool::panel(
            ui,
            &request.commands,
            &mut self.precise_input,
            &mut selected,
        );
        crate::sketch::inspect::measure_panel(ui, &request.commands, &mut selected);
        crate::face_properties::draw(ui, &request.commands, &mut selected);
        crate::sketch::region_properties::panel(ui, &request.commands, &mut selected);
        crate::sketch_delete::append_keyboard_delete(ui, &request.commands, &mut selected);
        crate::application::document_tabs::document_keys(
            ui,
            keys.switch_document,
            self.home.active,
            &request.commands,
            &mut selected,
        );
        self.workspace
            .switcher
            .panel(ui, &request.commands, &mut selected);
        if self.document_files.take_options_request() {
            self.preferences.request();
        }
        if self.document_files.take_properties_request()
            && let Some(id) = request.commands.document
        {
            self.document_tabs
                .menu
                .properties
                .request(&request.commands, id);
        }
        self.document_tabs
            .menu
            .properties
            .panel(ui, &request.commands);
        self.document_files
            .closing_panel(ui, &request.commands, &mut selected);
        let close_prompt = self
            .document_files
            .close_prompt_semantics()
            .map(|mut prompt| {
                let scale = request.layout_scale();
                for bounds in std::iter::once(&mut prompt.bounds)
                    .chain(prompt.buttons.iter_mut().map(|button| &mut button.bounds))
                {
                    for axis in 0..2 {
                        bounds[axis] *= scale[axis];
                        bounds[axis + 2] *= scale[axis];
                    }
                }
                prompt
            });
        let home_changed = home_follows_actions(&mut self.home, &selected);
        self.known_documents.clone_from(&request.commands.documents);
        self.known_document_type = request.commands.document_type;
        self.workspaces.note_created(&selected);
        // A popup that closed this frame still owned the key that closed it.
        let popup_open = self.popup_was_open || crate::widget_abi::any_popup_open(ui);
        self.popup_was_open = crate::widget_abi::any_popup_open(ui);
        // A dialog over the workspace is not a picture of the document, nor of its tab.
        if popup_open {
            self.document_tabs.previews.showing = None;
        }
        self.document_tabs.previews.enqueue_picture(&mut selected);
        let constraint_hover =
            crate::sketch::constraint_glyphs::unobstructed_hover(ui, constraint_hover);
        let snapshot = capture_frame(self.context.render(), request.extent, &self.resources)?;
        let mut activity = frame_activity()?;
        let mut capture = input_capture(self.context.io());
        capture.keyboard &= !keys.tool_keyboard;
        // Dear ImGui claims the keyboard for whichever window last had focus. A click on the
        // ribbon must leave M and F8 to the sketch unless a widget is really using the keys.
        // A popup always owns them, even on the frame Escape closes it and focus moves on.
        capture.keyboard &= capture.text || self.context.io().nav_visible;
        capture.keyboard |= popup_open
            || self.ribbon.tips.consumed
            || self.workspace.browser_consumed_arrow()
            || self.ribbon.customization.navigation_consumed;
        let arrow_press = request.events.iter().any(|event| {
            matches!(
                event,
                GuiInputEvent::Key {
                    key: crate::GuiKey::LeftArrow
                        | crate::GuiKey::RightArrow
                        | crate::GuiKey::UpArrow
                        | crate::GuiKey::DownArrow,
                    pressed: true
                }
            )
        });
        // ImGui can queue the first navigation move before it exposes the focus highlight (#3874).
        if arrow_press && activity.pending_navigation && !keys.tool_keyboard {
            capture.keyboard = true;
        }
        if arrow_press
            && self.ribbon.tips.active()
            && !capture.text
            && !popup_open
            && !self.workspace.browser_consumed_arrow()
        {
            capture.keyboard = false;
        }
        if arrow_press && self.workspace.feature_keyboard_focus() {
            capture.keyboard = true;
        }
        capture.text |= self.ribbon.tips.consumed
            || (self.ribbon.tips.active()
                && request
                    .events
                    .iter()
                    .any(|event| matches!(event, GuiInputEvent::Text(_))));
        capture.popup = popup_open
            || self.dimension_value_editor.had_keyboard()
            || self.ribbon.tips.consumed
            || self.ribbon.customization.escape_consumed
            || self.ribbon.display.temporary_escape_consumed;
        capture.pointer |= self.dimension_value_editor.over_label || over_fields;
        activity.pending_layout |= self.orientation.needs_frame();
        activity.pending_layout |= self.ribbon.panels.pending_layout || self.ribbon.needs_frame();
        activity.pending_layout |=
            self.workspace.pending_layout || home_changed || self.home.hover_changed;
        activity.pending_layout |= self.document_files.needs_frame()
            || self.document_tabs.needs_frame()
            || self.workspaces.needs_frame()
            || self.workspace.switcher.needs_frame()
            || self.document_tabs.menu.properties.needs_frame();
        Ok(GuiFrameOutput {
            window_action,
            window_controls,
            close_prompt,
            accessibility_popup_active: self.popup_was_open,
            draws: snapshot,
            actions: selected,
            parameter_actions,
            constraint_hover,
            capture,
            activity,
            document_area: area.framebuffer_region(request.layout_scale(), request.extent),
        })
    }
}
