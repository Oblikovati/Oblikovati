// SPDX-License-Identifier: GPL-2.0-only
//! Draw the sheet and tab for one separately managed desktop window.
use crate::application::document_creation::workspaces::draw_sheet_in_window;
use crate::application::document_tabs::TabMenu;
use crate::{GuiCommandState, GuiFrame, GuiInputEvent, GuiShellAction, GuiTextureResources};
use imgui::{Condition, Context, SuspendedContext, Ui, WindowFlags, sys};
use oblikovati_api::wire::{ActivateDocumentArgs, DocumentInfo};

/// The renderer packet and shell actions from one drawing window.
pub struct GuiDetachedDrawingFrame {
    pub draws: GuiFrame,
    pub actions: Vec<GuiShellAction>,
    pub needs_frame: bool,
}

/// An independent UI for one drawing. Its ImGui context is active only between main frames.
pub struct GuiDetachedDrawing {
    document: u64,
    menu: TabMenu,
    context: Option<SuspendedContext>,
    resources: GuiTextureResources,
}

impl GuiDetachedDrawing {
    /// Prepare the drawing UI without changing the main frame. Example: `GuiDetachedDrawing::new(id)?`.
    pub fn new(document: u64) -> Result<Self, String> {
        let _restore = RestorePrimaryContext::park();
        let mut context = crate::context_owner::create_owner_context()?;
        let atlas = crate::engine::configure_owner_context(&mut context);
        let resources = GuiTextureResources::new(atlas, [])?;
        Ok(Self {
            document,
            resources,
            menu: TabMenu::default(),
            context: Some(context.suspend()),
        })
    }

    /// Capture owned draw data and actions. Example: `drawing.frame([640, 480], &state, &[])?`.
    pub fn frame(
        &mut self,
        extent: [u32; 2],
        state: &GuiCommandState,
        events: &[GuiInputEvent],
    ) -> Result<GuiDetachedDrawingFrame, String> {
        let row = state
            .documents
            .iter()
            .find(|row| row.id == self.document)
            .ok_or_else(|| {
                format!(
                    "detached document {}: expected an open drawing",
                    self.document
                )
            })?;
        let mut actions = Vec::new();
        let _restore = RestorePrimaryContext::park();
        let mut context = activate(&mut self.context);
        prepare(&mut context, extent, state, events);
        let result = self.draw_frame(&mut context, extent, state, row, &mut actions);
        self.context = Some(context.suspend());
        result.map(|(draws, needs_frame)| GuiDetachedDrawingFrame {
            draws,
            actions,
            needs_frame,
        })
    }

    fn draw_frame(
        &mut self,
        context: &mut Context,
        extent: [u32; 2],
        state: &GuiCommandState,
        row: &DocumentInfo,
        actions: &mut Vec<GuiShellAction>,
    ) -> Result<(GuiFrame, bool), String> {
        let ui = context.frame();
        ui.window("##detached-drawing")
            .position([0.; 2], Condition::Always)
            .size(extent.map(|value| value as f32), Condition::Always)
            .flags(
                WindowFlags::NO_DECORATION
                    | WindowFlags::NO_MOVE
                    | WindowFlags::NO_DOCKING
                    | WindowFlags::NO_SAVED_SETTINGS,
            )
            .build(|| self.sheet_and_tab(ui, state, row, actions));
        let properties = self.menu.properties.panel(ui, state);
        let draws = crate::snapshot::capture_frame(context.render(), extent, &self.resources);
        let needs_frame = crate::context_owner::frame_activity()?.needs_frame();
        properties?;
        Ok((draws?, needs_frame))
    }

    fn sheet_and_tab(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        row: &DocumentInfo,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let Some(sheet) = state.drawing_documents.get(&row.id) {
            draw_sheet_in_window(ui, &row.name, sheet, state.drawing_projections.get(&row.id));
        }
        if ui.is_window_hovered()
            && ui.is_mouse_clicked(imgui::MouseButton::Left)
            && state.document != Some(row.id)
        {
            actions.push(GuiShellAction::ActivateDocument(ActivateDocumentArgs {
                id: row.id,
            }));
        }
        if let Some(_bar) = ui.tab_bar("##detached-tabs") {
            let _tab = ui.tab_item(&row.name);
            self.menu.popup(ui, state, row, actions);
        }
    }

    /// Inspect a rendered auxiliary frame with its context selected.
    /// Example: `drawing.inspect_frame(|| read_diagnostic_draws())`.
    pub fn inspect_frame<T>(&mut self, read: impl FnOnce() -> T) -> T {
        let _restore = RestorePrimaryContext::park();
        let context = activate(&mut self.context);
        let output = read();
        self.context = Some(context.suspend());
        output
    }
}

impl Drop for GuiDetachedDrawing {
    fn drop(&mut self) {
        let _restore = RestorePrimaryContext::park();
        // imgui-rs 0.12 ends the current frame when any context drops.
        drop(self.context.take());
    }
}

fn prepare(
    context: &mut Context,
    extent: [u32; 2],
    state: &GuiCommandState,
    events: &[GuiInputEvent],
) {
    if let Some(theme) = &state.theme {
        crate::theme::apply_chrome_theme(&mut context.style_mut().colors, theme.palette());
    }
    let io = context.io_mut();
    io.display_size = extent.map(|value| value as f32);
    io.delta_time = 1. / 60.;
    for event in events {
        crate::input::apply_input(io, event, [1.; 2]);
    }
}

fn activate(context: &mut Option<SuspendedContext>) -> Context {
    context
        .take()
        .expect("expected a parked drawing context")
        .activate()
        .expect("expected exclusive drawing context activation")
}

struct RestorePrimaryContext(*mut sys::ImGuiContext);

impl RestorePrimaryContext {
    fn park() -> Self {
        // All contexts belong to the same desktop thread and the primary frame has ended.
        let primary = unsafe { sys::igGetCurrentContext() };
        unsafe { sys::igSetCurrentContext(std::ptr::null_mut()) };
        Self(primary)
    }
}

impl Drop for RestorePrimaryContext {
    fn drop(&mut self) {
        unsafe { sys::igSetCurrentContext(self.0) };
    }
}
