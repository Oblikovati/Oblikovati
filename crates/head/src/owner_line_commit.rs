// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadWindow, placement_input::SketchPlacementState};
use oblikovati_gui::{GuiLineCommand, GuiLineCommandMenu, GuiSketchHud, GuiSketchHudField};
use oblikovati_model::{
    application_options::{SketchApplicationOptions, SketchPreferenceAccess},
    history::SketchLineChainCreation,
    quantity::Dimension,
    sketch::SketchEntityId,
};

/// A new start this close to a vertex is at that vertex.
const ACQUIRE_VERTEX_PIXELS: f64 = 12.;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Move or place with the pointer. A placing press with a start commits one segment.
    pub(super) fn line_pointer(
        &mut self,
        tool: &mut SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
        place: bool,
    ) -> Result<(), String> {
        let point = window.sketch_point(pixel, tool.plane)?;
        tool.cursor_pixel = Some(pixel);
        tool.line_cursor(point);
        if let (true, Some(point)) = (place, point) {
            if tool.line_awaits_start() {
                let acquired = self.vertex_under_pointer(tool, window, pixel)?;
                let at = acquired.map_or(point, |(position, _)| position);
                let vertex = acquired.map(|(_, vertex)| vertex);
                tool.line_start_on(at, vertex, Some(pixel))?;
            } else {
                self.commit_pending_line(tool, Some(pixel))?;
            }
        }
        self.refresh_line_fields(tool)
    }
    /// The nearest sketch vertex within reach: a start placed there joins it instead of adding one.
    fn vertex_under_pointer(
        &self,
        tool: &SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
    ) -> Result<Option<([f64; 2], SketchEntityId)>, String> {
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let geometry = snapshot
            .sketches
            .get(tool.sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        let mut nearest = None;
        let mut reach = ACQUIRE_VERTEX_PIXELS;
        for point in geometry.points() {
            let at = window.sketch_pixel(tool.plane, point.position)?;
            let distance = super::sketch_picking::screen_distance(at, pixel);
            if distance <= reach {
                reach = distance;
                nearest = Some((point.position, point.id));
            }
        }
        Ok(nearest)
    }
    /// Place the next point exactly where it was typed.
    pub(super) fn line_place_exact(
        &mut self,
        tool: &mut SketchPlacementState,
        position: [f64; 2],
    ) -> Result<(), String> {
        match tool.line_segment_to(position, None) {
            Some(input) => self.commit_line_input(tool, &input, None)?,
            None => tool.line_start_on(position, None, None)?,
        }
        self.refresh_line_fields(tool)
    }
    /// Accept what the numeric fields describe; a value that does not evaluate holds its place.
    pub(super) fn commit_line_fields(
        &mut self,
        tool: &mut SketchPlacementState,
    ) -> Result<(), String> {
        self.evaluate_line_fields(tool)?;
        if !tool.fields_invalid() {
            self.commit_pending_line(tool, None)?;
        }
        self.refresh_line_fields(tool)
    }
    fn commit_pending_line(
        &mut self,
        tool: &mut SketchPlacementState,
        pixel: Option<[i32; 2]>,
    ) -> Result<(), String> {
        let Some(input) = tool.line_input() else {
            return Ok(());
        };
        self.commit_line_input(tool, &input, pixel)
    }
    fn commit_line_input(
        &mut self,
        tool: &mut SketchPlacementState,
        input: &SketchLineChainCreation,
        pixel: Option<[i32; 2]>,
    ) -> Result<(), String> {
        // #3830: the command refuses zero-length edges; the typed line API deliberately admits them.
        oblikovati_model::sketch::SketchPlacementRecipe::line_chain(&input.points, input.closed)
            .map_err(|cause| cause.to_string())?;
        let report = self.commit_sketch_line_chain(tool.document, tool.sketch, input)?;
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        // The solver and snaps own the vertex; the next segment starts where it really is.
        let end = report
            .points
            .last()
            .map(|vertex| {
                session
                    .snapshot()
                    .sketches
                    .get(tool.sketch)
                    .and_then(|sketch| sketch.geometry().point(*vertex))
                    .map(|point| point.position)
            })
            .transpose()
            .map_err(|cause| cause.to_string())?
            .unwrap_or(input.points[input.points.len() - 1]);
        tool.accept_line_segment(&report, end, pixel, generation);
        Ok(())
    }
    /// A refused commit rolls back through history, so the command re-reads the generation.
    pub(super) fn rearm_line_after_failure(&self, tool: &mut SketchPlacementState) {
        if let Ok(session) = self.files.workspace().session(tool.document) {
            tool.generation = session.generation();
        }
    }
    /// Re-read typed values and the pending preview after anything that may change them.
    pub(super) fn refresh_line_fields(
        &mut self,
        tool: &mut SketchPlacementState,
    ) -> Result<(), String> {
        self.evaluate_line_fields(tool)?;
        tool.refresh_line_preview()?;
        tool.resolved_preview = self.resolved_placement_preview(tool)?;
        let prompt = self.line_status_prompt(tool);
        self.placement_notice(prompt);
        Ok(())
    }
    fn evaluate_line_fields(&self, tool: &mut SketchPlacementState) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        let parameters = &session.snapshot().parameters;
        tool.evaluate_line_values(&|index, source| {
            let dimension = line_field_dimension(index);
            let value = parameters
                .evaluate_scalar_expression(source, dimension)
                .ok()?
                .value;
            // An explicit zero length is refused here, before any geometry is built.
            (dimension != Dimension::Length || value > 0.).then_some(value)
        });
        Ok(())
    }
    /// With the fields hidden, the status bar is where the pending values stay readable.
    fn line_status_prompt(&self, tool: &SketchPlacementState) -> String {
        let prompt = tool.prompt();
        if self.sketch_hud_enabled() {
            return prompt;
        }
        match self.line_measured_fields(tool) {
            Some([length, angle]) => format!("{prompt}  Length {length}  Angle {angle}"),
            None => prompt,
        }
    }
    /// A fixture's way to build document state through the product's own commands.
    #[cfg(test)]
    pub(crate) fn document_commands(
        &mut self,
        document: u64,
    ) -> oblikovati_session::workspace::SessionCommands<'_, O> {
        self.files.commands(document).unwrap()
    }
    /// Attach the stored sketch options; example: `owner.with_sketch_preferences(Box::new(file))`.
    pub fn with_sketch_preferences(mut self, store: Box<dyn SketchPreferenceAccess>) -> Self {
        self.sketch_options = store.sketch_options().clone();
        self.sketch_store = Some(store);
        self
    }
    /// Persist first, so a failed write leaves the options in effect unchanged.
    pub(crate) fn change_sketch_options(
        &mut self,
        options: SketchApplicationOptions,
    ) -> Result<(), String> {
        if let Some(store) = self.sketch_store.as_mut() {
            store.set_sketch_options(options.clone())?;
        }
        self.sketch_options = options;
        Ok(())
    }
    /// How many dimensions the sketch under edit still needs; a failed solve shows none.
    pub(super) fn gui_dimensions_needed(&self, document: u64) -> Option<usize> {
        let edit = self
            .active_sketch_edit()
            .filter(|edit| edit.document == document)?;
        let session = self.files.workspace().session(document).ok()?;
        Some(session.analyze_sketch(edit.sketch).ok()?.degrees_of_freedom)
    }
    pub(super) fn sketch_hud_enabled(&self) -> bool {
        self.sketch_options.heads_up_display
    }
    /// The numeric fields beside the pointer; the option hides them without changing the command.
    pub(super) fn gui_sketch_hud(&self) -> Option<GuiSketchHud> {
        let tool = self.placement_state()?;
        if tool.tool != crate::SketchPlacementTool::Line || !self.sketch_hud_enabled() {
            return None;
        }
        tool.line_restart_available().then_some(())?;
        let measured = self.line_measured_fields(tool).unwrap_or_default();
        Some(GuiSketchHud {
            pixel: tool.cursor_pixel?,
            fields: std::array::from_fn(|index| {
                let view = tool.field_view(index);
                GuiSketchHudField {
                    text: view.typed.unwrap_or_else(|| measured[index].clone()),
                    active: view.active,
                    invalid: view.invalid,
                    selected: view.selected,
                }
            }),
        })
    }
    fn line_measured_fields(&self, tool: &SketchPlacementState) -> Option<[String; 2]> {
        let recipe = tool.preview.as_ref()?;
        let session = self.files.workspace().session(tool.document).ok()?;
        let units = session.snapshot().parameters.units();
        let length = recipe.fields.first()?.value;
        let angle = recipe.fields.get(1)?.value.to_degrees().rem_euclid(360.);
        Some([
            units
                .format_display(oblikovati_model::quantity::Quantity::new(
                    length,
                    Dimension::Length,
                ))
                .ok()?,
            format!("{angle:.2} deg"),
        ])
    }
    /// What the Line command's active menu offers now; example: Close needs two segments.
    pub(super) fn gui_line_command_menu(&self) -> Option<GuiLineCommandMenu> {
        let tool = self.placement_state()?;
        (tool.tool == crate::SketchPlacementTool::Line).then_some(())?;
        Some(GuiLineCommandMenu {
            close_enabled: tool.line_close_available(),
            restart_enabled: tool.line_restart_available(),
            auto_project: self.sketch_options.auto_project_origin,
        })
    }
    /// Carry out one active menu choice; example: `owner.line_command_from_gui(GuiLineCommand::Ok)`.
    pub fn line_command_from_gui(&mut self, command: GuiLineCommand) -> Result<bool, String> {
        let Some(crate::sketch_tool::SketchToolState::Placement(mut tool)) =
            self.sketch_tool.take()
        else {
            return Ok(false);
        };
        let result = self.apply_line_command(&mut tool, command);
        if result.is_err() {
            self.rearm_line_after_failure(&mut tool);
        }
        if !matches!(result, Ok(true)) {
            self.sketch_tool = Some(crate::sketch_tool::SketchToolState::Placement(tool));
        }
        result.map(|_| true)
    }
    /// True when the command ended.
    fn apply_line_command(
        &mut self,
        tool: &mut SketchPlacementState,
        command: GuiLineCommand,
    ) -> Result<bool, String> {
        match command {
            GuiLineCommand::Cancel | GuiLineCommand::Ok => {
                self.command_notice = Some(DesktopCommandNotice::Ready);
                return Ok(true);
            }
            GuiLineCommand::Close => {
                let input = tool.line_close_input().ok_or_else(|| {
                    "Line Close: expected two accepted segments in the chain".to_string()
                })?;
                self.commit_line_input(tool, &input, None)?;
                self.restart_line_chain(tool);
            }
            GuiLineCommand::Restart => self.restart_line_chain(tool),
            GuiLineCommand::ToggleAutoProject => self.toggle_auto_project()?,
        }
        Ok(false)
    }
    pub(super) fn restart_line_chain(&mut self, tool: &mut SketchPlacementState) {
        let generation = tool.generation;
        tool.restart_line(generation);
        self.placement_notice(tool.prompt());
    }
    fn toggle_auto_project(&mut self) -> Result<(), String> {
        let mut options = self.sketch_options.clone();
        options.auto_project_origin = !options.auto_project_origin;
        self.change_sketch_options(options)
    }
}
fn line_field_dimension(index: usize) -> Dimension {
    if index == 0 {
        Dimension::Length
    } else {
        Dimension::Angle
    }
}
