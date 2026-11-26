// SPDX-License-Identifier: GPL-2.0-only
use super::{frame_inputs::sketch_activation, *};
#[path = "feature_actions.rs"]
mod feature_actions;
impl GuiFrameEngine {
    pub(super) fn draw_controls(
        &mut self,
        request: &GuiFrameRequest,
    ) -> Result<GuiFrameOutput, String> {
        self.parameter_editor
            .synchronize_document(request.commands.document)?;
        let seed_asset_dock = self.prepare_workspace_frame(&request.commands);
        let keys = self.prepare_control_inputs(request);
        self.sync_tool_visibility();
        let mut selected = Vec::new();
        self.prepare_dock_owners(&request.commands, &mut selected);
        crate::widget_abi::prepare_dock_hit_test(workspace_layout::workspace_pointer(
            &self.context,
            request,
        ));
        let ui = self.context.frame();
        self.dock_lifecycle.reconcile_native_tabs(ui)?;
        self.workspace
            .sync_origin_document(request.commands.document);
        crate::application::workspace_layout::browser_feature::draw_feature_edges(ui, request);
        self.workspace.draw_selected_origin(ui, request);
        self.document_files.home_in_front = self.home.active;
        let home_was_active = self.home.active;
        // The selector can close before the same gesture releases over the viewport.
        let right = imgui::MouseButton::Right;
        if ui.is_mouse_clicked(right) {
            self.selector_context_gesture = self.ribbon.display.selector_open;
        }
        let selector_context_gesture = self.selector_context_gesture;
        self.selector_context_gesture &= ui.is_mouse_down(right);
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
        let visibility = self.workspace.tool_visibility;
        self.ribbon.team_web.draw(ui, &mut selected);
        self.material_browser.set_open(visibility.material);
        self.appearance_browser.set_open(visibility.appearance);
        self.parameter_editor.set_open(visibility.parameters);
        self.selection_filter.set_open(visibility.selection_filter);
        self.history_browser.set_open(visibility.history);
        let focus = self.accessible_focus.take();
        let frame_focus = super::frame_inputs::frame_focus(focus);
        if let Some(crate::GuiAccessibleFocus::ClosePrompt(choice)) = focus {
            self.document_files.focus_close_prompt(choice);
        }
        let (window_action, window_controls) = crate::window_frame::draw(
            ui,
            request,
            crate::chrome::application_bar_height(&self.icons.presentation),
            &mut selected,
            &mut self.window_frame_interaction,
            frame_focus,
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
        self.workspace.draw_status(
            ui,
            request,
            home_was_active,
            (
                self.command_alias.text(),
                self.extrude_dialog.distance_prompt(),
            ),
        );
        self.workspace
            .sync_before_docking(&request.commands, home_was_active);
        self.dock_lifecycle.reconcile_native_tabs(ui)?;
        let logical =
            std::array::from_fn(|axis| request.extent[axis] as f32 / request.layout_scale()[axis]);
        let outer = self.workspace.outer_area(logical, band_height);
        let area = crate::application::workspace_docking::draw(
            ui,
            outer,
            &mut self.workspace,
            logical,
            band_height,
            seed_asset_dock,
        );
        let dimension_positions =
            crate::dimension_labels::dimension_label_layout(ui, request, area);
        let constraint_hover =
            crate::sketch::draw_viewport_overlays(ui, request, &dimension_positions);
        let over_fields = crate::sketch::line::hud_panel(ui, request);
        self.document_tabs.previews.synchronize(&request.commands);
        self.document_tabs.panel(
            ui,
            request,
            &self.workspace,
            (area, &self.icons.presentation),
            &mut self.home,
            &mut selected,
        );
        let mut open_inference_scope = false;
        selected.retain(|action| {
            if matches!(action, GuiShellAction::OpenInferenceScope) {
                open_inference_scope = true;
                return false;
            }
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
        super::inference_dialog::draw(
            &mut self.inference_scope,
            ui,
            &request.commands,
            &mut selected,
            open_inference_scope,
        );
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
            (area, &self.icons.presentation),
            self.part_context_open,
            selector_context_gesture,
            &mut selected,
        );
        if selected.contains(&GuiShellAction::OpenModelStateTable) {
            self.model_state_table.open(&request.commands);
        }
        selected.retain(|action| !matches!(action, GuiShellAction::OpenModelStateTable));
        self.model_state_table
            .panel(ui, &request.commands, &mut selected);
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
        }
        selected.retain(|action| !matches!(action, GuiShellAction::OpenSketchPattern { .. }));
        if !self.home.active {
            if let Some(zoom) = self.ribbon.customization.pending_zoom.take() {
                self.orientation.run_zoom(zoom, request, &mut selected);
            }
            self.orientation.panel(
                ui,
                &self.icons.presentation,
                request,
                area,
                &mut selected,
                crate::application::viewport_orientation::ViewControlVisibility {
                    cube: self.workspace.view_cube_visible,
                    navigation: self.workspace.navigation_bar_visible,
                },
            );
        } else {
            self.orientation.set_suspended(true);
        }
        let detached = self.document_tabs.is_detached(request.commands.document);
        self.workspaces
            .panel(ui, &request.commands, area, detached, &mut selected);
        self.plane_selection
            .take_browser_choice(&mut self.workspace, &mut selected);
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
            self.part_context_open = !selector_context_gesture
                && !self.orientation.navigation_owns_viewport()
                && crate::part_context_menu::canvas_menu(
                    ui,
                    &request.commands,
                    area,
                    &mut menu_actions,
                    &mut repeat_wheel,
                    self.workspace.marking_menu_enabled,
                    &self.icons.presentation,
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
            self.workspace.selection_toolbar(
                ui,
                request,
                &mut selected,
                self.blend_dialog.active() || self.part_context_open,
            );
        } else {
            self.part_context_open = false;
        }
        feature_actions::open_drafts(
            &request.commands,
            &mut selected,
            &mut self.shell_feature_dialog,
            &mut self.hole_dialog,
            &mut self.move_body_dialog,
            &mut self.blend_dialog,
            &mut self.extrude_dialog,
        );
        self.home.panel(
            ui,
            &request.commands,
            area,
            &mut selected,
            &self.icons.presentation,
        );
        if let Some(path) = apply_home_requests(&mut self.home, &mut self.document_files) {
            self.document_tabs.menu.properties.request_file(&path);
        }
        let icons = &self.icons.presentation;
        let commands = &request.commands;
        self.document_files
            .panel(ui, commands, &mut selected, icons);
        self.preferences
            .panel(ui, commands, &mut selected, self.home.active, icons);
        Self::draw_asset_docks(
            (&mut self.material_browser, &mut self.appearance_browser),
            ui,
            commands,
            self.home.active,
            &mut selected,
        );
        self.extrude_dialog.panel(
            ui,
            commands,
            self.plane_selection.active,
            &mut selected,
            crate::extrude_dialog::ExtrudePanelControls {
                icons,
                parameters: &mut self.parameter_editor,
                popup_was_open: self.popup_was_open,
                area,
            },
        );
        self.blend_dialog.panel(ui, commands, &mut selected, area);
        self.hole_dialog.panel(ui, commands, area, &mut selected);
        self.shell_feature_dialog
            .panel(ui, commands, area, &mut selected);
        self.move_body_dialog
            .panel(ui, commands, area, &mut selected);
        self.offset_plane.panel(ui, commands, area, &mut selected);
        let interface = (&self.quick_access, &self.workspace);
        self.ribbon
            .customization_panel(ui, commands, &mut selected, interface);
        self.create_block
            .panel(ui, &request.commands, &mut selected);
        self.sketch_pattern
            .panel(ui, &request.commands, &mut selected);
        let editors = (&mut self.sketch_text, &mut self.curvature_settings);
        Self::draw_sketch_editors(editors, ui, &request.commands, &mut selected);
        crate::sketch::line::command_menu(ui, &request.commands, &mut selected);
        self.precise_input
            .panel(ui, &request.commands, &mut selected);
        crate::sketch::inspect::canvas_menu(
            ui,
            &request.commands,
            &mut selected,
            &mut self.sketch_popup_token,
            self.workspace.marking_menu_enabled,
            icons,
        );
        self.dimension_value_editor
            .panel(ui, request, &dimension_positions, &mut selected);
        external_panels::draw(
            &self.external_docks,
            ui,
            &request.commands,
            &mut self.precise_input,
            &mut selected,
        );
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
            .panel(ui, &request.commands)?;
        self.document_files
            .closing_panel(ui, &request.commands, &mut selected);
        super::dock_tab_actions::draw(
            ui,
            request.commands.browser.is_some(),
            (!self.home.active).then_some(area),
            super::dock_tab_actions::DockTabOwners {
                workspace: &mut self.workspace,
                material: &mut self.material_browser,
                appearance: &mut self.appearance_browser,
                parameters: &mut self.parameter_editor,
                selection: &mut self.selection_filter,
                history: &mut self.history_browser,
            },
        );
        let close_prompt = super::close_prompt_scale::scale_optional(
            self.document_files.close_prompt_semantics(),
            request.layout_scale(),
        );
        let cancelled_pick = self.ribbon.cancel_unrouted_model_pick(&selected);
        home_follows_actions(&mut self.home, &selected);
        self.known_documents.clone_from(&request.commands.documents);
        self.known_document_type = request.commands.document_type;
        self.workspaces.note_created(&selected);
        // A popup that closed this frame still owned the key that closed it.
        let (popup_open, doubled) = sketch_activation(ui, request, &mut self.popup_was_open);
        // A dialog over the workspace is not a picture of the document, nor of its tab.
        if popup_open {
            self.document_tabs.previews.showing = None;
        }
        self.document_tabs.previews.enqueue_picture(&mut selected);
        let constraint_hover =
            crate::sketch::constraint_glyphs::unobstructed_hover(ui, constraint_hover);
        self.external_docks.observe_actions(&selected);
        let dock_lifecycle_changed = self.dock_lifecycle.reconcile_native_tabs(ui)?;
        let snapshot = capture_frame(self.context.render(), request.extent, &self.resources)?;
        let mut activity = frame_activity()?;
        let mut capture = input_capture(self.context.io());
        capture.keyboard &= !keys.tool_keyboard;
        // Keep sketch shortcuts after ribbon clicks, except while a widget or popup owns the keys.
        capture.keyboard &= capture.text || self.context.io().nav_visible;
        capture.keyboard |= popup_open
            || self.ribbon.tips.consumed
            || self.workspace.browser_consumed_arrow()
            || self.ribbon.customization.navigation_consumed;
        let arrow_press = super::frame_inputs::has_arrow_press(request);
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
            || self.dock_editor_consumed_escape()
            || self.ribbon.tips.consumed
            || self.ribbon.customization.escape_consumed
            || self.ribbon.display.temporary_escape_consumed;
        capture.pointer |= self.dimension_value_editor.over_label || over_fields;
        activity.pending_layout |=
            dock_lifecycle_changed || cancelled_pick || self.extrude_dialog.needs_frame();
        activity.pending_layout |= self.orientation.needs_frame();
        activity.pending_layout |= self.ribbon.panels.pending_layout || self.ribbon.needs_frame();
        activity.pending_layout |= self.ribbon.customization.needs_frame();
        activity.pending_layout |= self.workspace.pending_layout
            || self.home.active != home_was_active
            || self.home.hover_changed;
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
            sketch_double_click: doubled,
            activity,
            document_area: area.framebuffer_region(request.layout_scale(), request.extent),
        })
    }
}
