// SPDX-License-Identifier: GPL-2.0-only
//! Keep detached drawing windows on the desktop owner thread and renderer.
use super::{GUI_TICK, input::copied_gui_events};
use oblikovati_gui::{
    GuiCommandState, GuiDetachedDrawing, GuiDetachedDrawingFrame, GuiShellAction,
};
use oblikovati_viewport::{
    GuiFrameCompiler, Viewport, ViewportEvent, ViewportGuiSubmission, ViewportPresentation,
    ViewportRuntime, ViewportScene,
};
use std::{collections::BTreeMap, time::Duration};

#[derive(Default)]
pub(in crate::native) struct DrawingWindows(BTreeMap<u64, DrawingWindow>);

#[derive(Default)]
pub(super) struct DrawingWindowUpdate {
    pub(super) actions: Vec<GuiShellAction>,
    pub(super) restored: Vec<u64>,
    pub(super) wake_after: Option<Duration>,
    pub(super) failure: Option<(u64, String)>,
}

impl DrawingWindows {
    pub(in crate::native) fn targets(&self) -> Vec<&Viewport> {
        self.0.values().map(|drawing| &drawing.view).collect()
    }

    pub(in crate::native) fn close(&mut self) {
        self.0.clear();
    }

    pub(super) fn advance(
        &mut self,
        runtime: &ViewportRuntime,
        detached: &[u64],
        state: &GuiCommandState,
        now: Duration,
    ) -> DrawingWindowUpdate {
        let mut update = DrawingWindowUpdate::default();
        self.0.retain(|id, _| detached.contains(id));
        for id in detached {
            if let Some(row) = state.documents.iter().find(|row| row.id == *id) {
                let result = self.advance_window(runtime, row, state, now, &mut update);
                if let Err(cause) = result {
                    self.0.remove(id);
                    update.restored.push(*id);
                    update.failure.get_or_insert((*id, cause));
                }
            }
        }
        update
    }

    fn advance_window(
        &mut self,
        runtime: &ViewportRuntime,
        row: &oblikovati_api::wire::DocumentInfo,
        state: &GuiCommandState,
        now: Duration,
        update: &mut DrawingWindowUpdate,
    ) -> Result<(), String> {
        let drawing = match self.0.entry(row.id) {
            std::collections::btree_map::Entry::Occupied(entry) => entry.into_mut(),
            std::collections::btree_map::Entry::Vacant(entry) => {
                entry.insert(DrawingWindow::open(runtime, row)?)
            }
        };
        if drawing.advance(row, state, now, update)? {
            update.restored.push(row.id);
            self.0.remove(&row.id);
        }
        Ok(())
    }

    #[cfg(all(test, target_os = "linux"))]
    pub(in crate::native) fn inspect<R>(
        &mut self,
        id: u64,
        read: impl FnOnce(&Viewport) -> R,
    ) -> Option<R> {
        let drawing = self.0.get_mut(&id)?;
        Some(drawing.gui.inspect_frame(|| read(&drawing.view)))
    }
}

#[derive(Clone, PartialEq)]
struct DrawingSnapshot {
    row: oblikovati_api::wire::DocumentInfo,
    drawing: Option<oblikovati_model::drawing::DrawingDocument>,
    projection: Option<oblikovati_gui::GuiDrawingProjection>,
    theme: Option<oblikovati_gui::GuiThemeState>,
    ui_scale: Option<oblikovati_model::ui_scale::UiScaleOptions>,
}

impl DrawingSnapshot {
    fn matches(&self, row: &oblikovati_api::wire::DocumentInfo, state: &GuiCommandState) -> bool {
        self.row == *row
            && self.drawing.as_ref() == state.drawing_documents.get(&row.id)
            && self.projection.as_ref() == state.drawing_projections.get(&row.id)
            && self.theme == state.theme
            && self.ui_scale == state.ui_scale
    }
    fn read(row: &oblikovati_api::wire::DocumentInfo, state: &GuiCommandState) -> Self {
        Self {
            row: row.clone(),
            drawing: state.drawing_documents.get(&row.id).cloned(),
            projection: state.drawing_projections.get(&row.id).cloned(),
            theme: state.theme.clone(),
            ui_scale: state.ui_scale,
        }
    }
}

struct DrawingWindow {
    view: Viewport,
    gui: GuiDetachedDrawing,
    compiler: Option<GuiFrameCompiler>,
    raster_scale: f32,
    snapshot: Option<DrawingSnapshot>,
    last_draws: Option<oblikovati_gui::GuiFrame>,
    mapped: bool,
    deadline: Option<Duration>,
}

impl DrawingWindow {
    fn open(
        runtime: &ViewportRuntime,
        row: &oblikovati_api::wire::DocumentInfo,
    ) -> Result<Self, String> {
        let mut view = runtime.open_document_window(&row.name, [640, 480])?;
        view.submit(&ViewportScene {
            background: [0.; 3],
            triangles: Vec::new(),
        })?;
        Ok(Self {
            gui: GuiDetachedDrawing::new(row.id)?,
            view,
            compiler: None,
            raster_scale: 1.0,
            snapshot: None,
            last_draws: None,
            mapped: true,
            deadline: None,
        })
    }

    fn advance(
        &mut self,
        row: &oblikovati_api::wire::DocumentInfo,
        state: &GuiCommandState,
        now: Duration,
        update: &mut DrawingWindowUpdate,
    ) -> Result<bool, String> {
        let events = self.view.poll_events(32)?;
        if events.contains(&ViewportEvent::CloseRequested) {
            return Ok(true);
        }
        if let Some(mapped) = events.iter().rev().find_map(window_mapping) {
            self.mapped = mapped;
        }
        if self.mapped {
            self.redraw_if_needed(row, state, now, &events, &mut update.actions)?;
            update.wake_after = update
                .wake_after
                .into_iter()
                .chain(self.deadline.map(|deadline| deadline.saturating_sub(now)))
                .min();
        }
        Ok(false)
    }

    fn redraw_if_needed(
        &mut self,
        row: &oblikovati_api::wire::DocumentInfo,
        state: &GuiCommandState,
        now: Duration,
        events: &[ViewportEvent],
        actions: &mut Vec<GuiShellAction>,
    ) -> Result<(), String> {
        let current = self
            .snapshot
            .as_ref()
            .is_some_and(|snapshot| snapshot.matches(row, state));
        if current && events.is_empty() && self.deadline.is_none_or(|deadline| now < deadline) {
            return Ok(());
        }
        if self
            .snapshot
            .as_ref()
            .is_some_and(|current| current.row.name != row.name)
        {
            self.view.set_title(&row.name)?;
        }
        if !current {
            self.snapshot = Some(DrawingSnapshot::read(row, state));
        }
        let mut frame = self.draw_events(state, events, actions)?;
        actions.append(&mut frame.actions);
        let needs_frame = frame.needs_frame;
        let redraw = events.iter().any(|event| {
            matches!(
                event,
                ViewportEvent::Resized(_) | ViewportEvent::Mapped(true) | ViewportEvent::Expose
            )
        });
        let presented = !redraw && self.last_draws.as_ref() == Some(&frame.draws)
            || self.present_frame(frame)?;
        self.deadline = if !presented {
            Some(now)
        } else {
            needs_frame.then(|| now + GUI_TICK)
        };
        Ok(())
    }

    fn draw_events(
        &mut self,
        state: &GuiCommandState,
        events: &[ViewportEvent],
        actions: &mut Vec<GuiShellAction>,
    ) -> Result<GuiDetachedDrawingFrame, String> {
        let extent = self.view.drawable_extent()?;
        let backing_scale =
            super::advance::desktop_gui_scale(self.view.backing_scale(), state.ui_scale);
        let raster_scale = backing_scale * state.ui_scale.unwrap_or_default().font();
        if self.raster_scale != raster_scale {
            self.compiler = None;
            self.raster_scale = raster_scale;
        }
        let mut frame = None;
        for event in events {
            let copied = copied_gui_events(Some(event));
            if !copied.is_empty() {
                let mut next = self
                    .gui
                    .frame_scaled(extent, backing_scale, state, &copied)?;
                actions.append(&mut next.actions);
                frame = Some(next);
            }
        }
        frame.map_or_else(
            || self.gui.frame_scaled(extent, backing_scale, state, &[]),
            Ok,
        )
    }

    fn present_frame(&mut self, frame: GuiDetachedDrawingFrame) -> Result<bool, String> {
        if self.compiler.is_none() {
            self.compiler = Some(GuiFrameCompiler::with_resources(&frame.draws.resources)?);
        }
        let compiler = self.compiler.as_ref().expect("drawing frame compiler");
        if self.view.submit_gui_frame_checked(compiler, &frame.draws)?
            != ViewportGuiSubmission::Accepted
        {
            self.last_draws = None;
            return Ok(false);
        }
        let complete = matches!(self.view.present()?, ViewportPresentation::Completed(_));
        self.last_draws = complete.then_some(frame.draws);
        Ok(complete)
    }
}

fn window_mapping(event: &ViewportEvent) -> Option<bool> {
    match event {
        ViewportEvent::Mapped(mapped) => Some(*mapped),
        _ => None,
    }
}
