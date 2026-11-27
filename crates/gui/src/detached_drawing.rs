// SPDX-License-Identifier: GPL-2.0-only
//! Draw the sheet and tab for one separately managed desktop window.
use crate::application::document_creation::workspaces::{
    WorkspaceArea, draw_active_sheet, draw_sheet_in_window,
};
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
    raster_scale: f32,
}

impl GuiDetachedDrawing {
    /// Prepare the drawing UI without changing the main frame. Example: `GuiDetachedDrawing::new(id)?`.
    pub fn new(document: u64) -> Result<Self, String> {
        let _restore = RestorePrimaryContext::park();
        let (context, resources) = drawing_context()?;
        Ok(Self {
            document,
            resources,
            raster_scale: 1.0,
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
        self.frame_scaled(extent, 1.0, state, events)
    }

    /// Draw a detached window at its monitor's physical pixel density.
    /// Example: `drawing.frame_scaled([900, 600], 1.5, &state, &[])?`.
    pub fn frame_scaled(
        &mut self,
        extent: [u32; 2],
        backing_scale: f32,
        state: &GuiCommandState,
        events: &[GuiInputEvent],
    ) -> Result<GuiDetachedDrawingFrame, String> {
        let scale = backing_scale * state.ui_scale.unwrap_or_default().font();
        if !scale.is_finite() || scale <= 0.0 || scale * 13.0 > 256.0 {
            return Err(format!(
                "detached drawing scale {scale}: expected a positive finite font size at most 256 pixels"
            ));
        }
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
        if self.raster_scale != scale {
            let atlas = crate::ui_scale::rebuild_font_atlas(&mut context, scale);
            self.resources = match self.resources.with_font(atlas) {
                Ok(resources) => resources,
                Err(error) => {
                    self.context = Some(context.suspend());
                    return Err(error);
                }
            };
            self.raster_scale = scale;
        }
        prepare(&mut context, extent, scale, state, events);
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
        let logical_size = context.io().display_size;
        let ui = context.frame();
        crate::widget_abi::window_geometry(ui, [0.; 2], logical_size, Condition::Always);
        ui.window("##detached-drawing")
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

/// Prepare a sheet-only frame for Save As. Example: `drawing_preview_frame([256, 144], &state)?`.
pub fn drawing_preview_frame(
    extent: [u32; 2],
    state: &GuiCommandState,
) -> Result<GuiFrame, String> {
    let _restore = RestorePrimaryContext::park();
    let (mut context, resources) = drawing_context()?;
    prepare(&mut context, extent, 1.0, state, &[]);
    let area = WorkspaceArea {
        left: 0.,
        top: 0.,
        right: extent[0] as f32,
        bottom: extent[1] as f32,
    };
    let drawn = draw_active_sheet(context.frame(), state, area);
    let draws = context.render();
    if drawn {
        crate::snapshot::capture_frame(draws, extent, &resources)
    } else {
        Err(format!(
            "drawing preview for {:?}: expected an active sheet",
            state.document
        ))
    }
}

fn drawing_context() -> Result<(Context, GuiTextureResources), String> {
    let mut context = crate::context_owner::create_owner_context()?;
    let atlas = crate::engine::configure_owner_context(&mut context)?;
    let resources = GuiTextureResources::new(atlas, [])?;
    Ok((context, resources))
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
    scale: f32,
    state: &GuiCommandState,
    events: &[GuiInputEvent],
) {
    if let Some(theme) = &state.theme {
        crate::theme::apply_chrome_theme(&mut context.style_mut().colors, theme.palette());
    }
    let io = context.io_mut();
    io.display_size = extent.map(|value| value as f32 / scale);
    io.display_framebuffer_scale = [scale; 2];
    io.delta_time = 1. / 60.;
    for event in events {
        crate::input::apply_input(io, event, [scale; 2]);
    }
}

fn activate(context: &mut Option<SuspendedContext>) -> Context {
    context
        .take()
        .expect("expected a parked drawing context")
        .activate()
        .expect("expected exclusive drawing context activation")
}

struct RestorePrimaryContext {
    primary: *mut sys::ImGuiContext,
    #[cfg(test)]
    _lease: crate::test_context::TestContextLease,
}

impl RestorePrimaryContext {
    fn park() -> Self {
        #[cfg(test)]
        let lease = crate::test_context::acquire();
        // All contexts belong to the same desktop thread and the primary frame has ended.
        let primary = unsafe { sys::igGetCurrentContext() };
        unsafe { sys::igSetCurrentContext(std::ptr::null_mut()) };
        Self {
            primary,
            #[cfg(test)]
            _lease: lease,
        }
    }
}

impl Drop for RestorePrimaryContext {
    fn drop(&mut self) {
        unsafe { sys::igSetCurrentContext(self.primary) };
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn detached_drawing_refreshes_fonts_at_monitor_density() {
        let state = GuiCommandState {
            documents: vec![DocumentInfo {
                id: 7,
                name: "Drawing1".into(),
                visible: true,
                ..Default::default()
            }],
            ..Default::default()
        };
        let mut drawing = GuiDetachedDrawing::new(7).unwrap();
        let standard = drawing.frame_scaled([900, 600], 1.0, &state, &[]).unwrap();
        let high_dpi = drawing.frame_scaled([900, 600], 1.5, &state, &[]).unwrap();
        assert_eq!(high_dpi.draws.extent, [900, 600]);
        assert_ne!(standard.draws.resources, high_dpi.draws.resources);
        let restored = drawing.frame_scaled([900, 600], 1.0, &state, &[]).unwrap();
        assert_eq!(standard.draws.resources, restored.draws.resources);
    }
}
