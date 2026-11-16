// SPDX-License-Identifier: GPL-2.0-only
//! The sketch's inspection commands on the owner: Measure and Region Properties.
use super::sketch_picking::{SketchPickFilter, pick_witness};
#[path = "owner_inspect/measure_projection.rs"]
mod measure_projection;
use super::*;
use crate::{
    HeadWindow,
    sketch_inspect::{
        measure::{MeasureContext, MeasurePick, SketchMeasureState, measure_rows},
        region_properties::{SketchRegionState, region_result_rows},
    },
    sketch_tool::SketchToolState,
};
use measure_projection::first_near;
use oblikovati_gui::{GuiInspectAction, GuiMeasurePanel, GuiRegionPanel};
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};

/// A vertex is a selection of its own within this reach of the pointer, in pixels.
const VERTEX_REACH: f64 = 4.;
/// A line's midpoint is too, nearer in: a short segment's middle is also where it is clicked.
const MIDPOINT_REACH: f64 = 3.;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Accept a typed value into its dimension, one history step; an error keeps the old value.
    pub(super) fn edit_dimension_value(
        &mut self,
        value: &oblikovati_gui::GuiDimensionValue,
    ) -> Result<bool, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("dimension value {value:?}: expected an edited sketch"))?;
        if value.sketch != edit.sketch.0 {
            return Err(format!(
                "dimension value {value:?}: expected the active sketch {:?}",
                edit.sketch
            ));
        }
        if let Some(tool) = self.dimension_state() {
            if !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation) {
                return Err(format!(
                    "dimension value {value:?}: expected the current sketch generation"
                ));
            }
            if tool.placed.is_some()
                && tool.committed.as_ref().is_none_or(|owned| {
                    owned.sketch != value.sketch || owned.constraint != value.constraint
                })
            {
                return Err(format!(
                    "dimension value {value:?}: expected the active placed dimension"
                ));
            }
        }
        let pending = self.dimension_state().and_then(|tool| {
            let owned = tool.committed.as_ref()?;
            (owned.sketch == value.sketch && owned.constraint == value.constraint)
                .then_some((tool.serial, owned.expression.clone()))
        });
        if let Some((id, _)) = &pending {
            self.checked_dimension_draft(*id)?;
        }
        if pending
            .as_ref()
            .is_none_or(|(_, original)| *original != value.expression)
        {
            self.files
                .commands(edit.document)
                .map_err(|cause| cause.to_string())?
                .edit_sketch_dimension(
                    oblikovati_model::document_sketch::DocumentSketchId(value.sketch),
                    oblikovati_model::sketch::SketchConstraintId(value.constraint),
                    &oblikovati_model::history::SketchDimensionEdit {
                        target_expression: Some(value.expression.clone()),
                        resolve_once: false,
                        reference: None,
                        drive_limits: None,
                    },
                )
                .map_err(|cause| cause.to_string())?;
        }
        if let Some((id, _)) = pending {
            self.reset_dimension_draft(id)?;
        } else if self.dimension_state().is_some() {
            self.clear_geometric_picks(edit)?;
            let generation = self.capture_sketch_tool_context()?.1;
            if let Some(SketchToolState::Dimension(tool)) = self.sketch_tool.as_mut() {
                tool.generation = generation;
                tool.cursor = None;
            }
        }
        Ok(true)
    }
    /// Carry out one inspection choice; example: `owner.inspect_from_gui(StartMeasure)`.
    pub fn inspect_from_gui(&mut self, action: GuiInspectAction) -> Result<bool, String> {
        match action {
            GuiInspectAction::StartMeasure => {
                if self.active_sketch_edit().is_some() {
                    self.part_measure = None;
                    self.start_measure()?;
                } else {
                    self.start_part_measure()?;
                }
            }
            GuiInspectAction::OpenRegionProperties => self.open_region_properties()?,
            GuiInspectAction::CalculateRegions => self.calculate_regions()?,
            GuiInspectAction::FinishMeasure | GuiInspectAction::FinishRegionProperties => {
                self.sketch_tool = None;
                self.part_measure = None;
                self.command_notice = Some(DesktopCommandNotice::Ready);
            }
        }
        Ok(true)
    }
    /// At Ready, M starts Measure and F8 and F9 show and hide every relation's mark; while a
    /// command runs these keys are that command's.
    pub(super) fn sketch_ready_key(&mut self, event: ViewportEvent) -> Result<bool, String> {
        let ViewportEvent::Input(ViewportInput::PhysicalKey {
            key,
            pressed: true,
            modifiers,
            ..
        }) = event
        else {
            return Ok(false);
        };
        if self.active_sketch_edit().is_none() {
            return Ok(false);
        }
        if modifiers.control && !modifiers.shift {
            return Ok(false);
        }
        match key {
            ViewportPhysicalKey::D
                if !modifiers.control
                    && !modifiers.shift
                    && !modifiers.alt
                    && !modifiers.super_key =>
            {
                self.apply_dimension_intent(&oblikovati_gui::GuiDimensionAction::Start)?;
            }
            ViewportPhysicalKey::C if modifiers.control && modifiers.shift => {
                crate::HeadSketchWork::place_sketch_tool_from_gui(
                    self,
                    oblikovati_gui::GuiSketchTool::Circle,
                )?;
            }
            ViewportPhysicalKey::M => self.start_measure()?,
            ViewportPhysicalKey::F8 => self.set_constraints_shown(true)?,
            ViewportPhysicalKey::F9 => self.set_constraints_shown(false)?,
            _ => return Ok(false),
        }
        Ok(true)
    }
    fn start_measure(&mut self) -> Result<(), String> {
        let (edit, generation) = self.capture_sketch_tool_context()?;
        self.retire_sketch_tool()?;
        self.sketch_tool = Some(SketchToolState::Measure(Box::new(SketchMeasureState {
            edit,
            generation,
            picks: Vec::new(),
            rows: Vec::new(),
        })));
        self.placement_notice("Select geometry to measure".into());
        Ok(())
    }
    fn open_region_properties(&mut self) -> Result<(), String> {
        let (edit, generation) = self.capture_sketch_tool_context()?;
        self.retire_sketch_tool()?;
        self.sketch_tool = Some(SketchToolState::Region(Box::new(SketchRegionState {
            edit,
            generation,
            loops: Vec::new(),
            results: Vec::new(),
        })));
        self.placement_notice("Select a closed region".into());
        Ok(())
    }
    fn calculate_regions(&mut self) -> Result<(), String> {
        let Some(SketchToolState::Region(tool)) = self.sketch_tool.as_ref() else {
            return Ok(());
        };
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let rows = region_result_rows(&tool.loops, session.snapshot().parameters.units());
        if let Some(SketchToolState::Region(tool)) = self.sketch_tool.as_mut() {
            tool.results = rows;
        }
        Ok(())
    }
    pub(super) fn gui_measure_panel(&self) -> Option<GuiMeasurePanel> {
        if let Some(SketchToolState::Measure(tool)) = self.sketch_tool.as_ref() {
            return Some(GuiMeasurePanel {
                rows: tool.rows.clone(),
            });
        }
        self.gui_part_measure()
    }
    pub(super) fn gui_region_panel(&self) -> Option<GuiRegionPanel> {
        let Some(SketchToolState::Region(tool)) = self.sketch_tool.as_ref() else {
            return None;
        };
        Some(GuiRegionPanel {
            loops: tool.loop_names(),
            results: tool.results.clone(),
        })
    }
    pub(super) fn measure_candidate(
        &self,
        edit: crate::HeadSketchEdit,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<Option<oblikovati_viewport::ViewportSketchFeedback>, String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let sketch = session
            .snapshot()
            .sketches
            .get(edit.sketch)
            .map_err(|cause| cause.to_string())?;
        let pick = measure_target(sketch.geometry(), edit.plane, pixel, window)?;
        let pick = pick.filter(|candidate| match self.sketch_tool.as_ref() {
            Some(SketchToolState::Measure(tool)) => !tool.picks.contains(candidate),
            _ => false,
        });
        Ok(pick.map(|pick| match pick {
            MeasurePick::Curve(entity) => oblikovati_viewport::ViewportSketchFeedback::Entities(
                oblikovati_viewport::ViewportSketchHover {
                    sketch: edit.sketch,
                    entities: vec![entity],
                },
            ),
            MeasurePick::Point(position) | MeasurePick::Middle(position) => {
                oblikovati_viewport::ViewportSketchFeedback::Point {
                    sketch: edit.sketch,
                    position,
                }
            }
        }))
    }
    /// One event while Measure runs; Escape ends it, a press selects.
    pub(super) fn process_measure_event(
        &mut self,
        mut tool: Box<SketchMeasureState>,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        if ends_inspection(event) {
            self.command_notice = Some(DesktopCommandNotice::Ready);
            return true;
        }
        let changed = match selection_pixel(event) {
            Some(pixel) => self
                .measure_press(&mut tool, pixel, window)
                .unwrap_or_else(|cause| {
                    self.command_notice = Some(DesktopCommandNotice::SelectionFailed { cause });
                    true
                }),
            None => false,
        };
        self.sketch_tool = Some(SketchToolState::Measure(tool));
        changed
    }
    fn measure_press(
        &self,
        tool: &mut SketchMeasureState,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let geometry = snapshot
            .sketches
            .get(tool.edit.sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        // A click on nothing starts the selection over, as Done would without closing the panel.
        let Some(pick) = measure_target(geometry, tool.edit.plane, pixel, window)? else {
            tool.picks.clear();
            tool.rows.clear();
            return Ok(true);
        };
        tool.add_pick(pick);
        let context = MeasureContext {
            sketch: geometry,
            plane: tool.edit.plane,
            units: snapshot.parameters.units(),
        };
        tool.rows = measure_rows(&context, &tool.picks)?;
        Ok(true)
    }
    /// One event while Region Properties is open; a press accepts or releases the loop under it.
    pub(super) fn process_region_event(
        &mut self,
        mut tool: Box<SketchRegionState>,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        if ends_inspection(event) {
            self.command_notice = Some(DesktopCommandNotice::Ready);
            return true;
        }
        let changed = match selection_pixel(event) {
            Some(pixel) => self
                .region_press(&mut tool, pixel, window)
                .unwrap_or_else(|cause| {
                    self.command_notice = Some(DesktopCommandNotice::SelectionFailed { cause });
                    true
                }),
            None => false,
        };
        self.sketch_tool = Some(SketchToolState::Region(tool));
        changed
    }
    fn region_press(
        &self,
        tool: &mut SketchRegionState,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some(point) = window.sketch_point(pixel, tool.edit.plane)? else {
            return Ok(false);
        };
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let geometry = session
            .snapshot()
            .sketches
            .get(tool.edit.sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        let region = oblikovati_martensite::sketch_region::sketch_region_at(geometry, point)
            .map_err(|cause| cause.to_string())?;
        let Some(region) = region else {
            return Ok(false);
        };
        tool.toggle_loop(region);
        Ok(true)
    }
}
fn ends_inspection(event: ViewportEvent) -> bool {
    matches!(
        event,
        ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Cancel,
            pressed: true,
        }) | ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::Escape,
            pressed: true,
            ..
        })
    )
}
fn selection_pixel(event: ViewportEvent) -> Option<[i32; 2]> {
    match event {
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed: true,
            position,
            ..
        }) => Some(position),
        _ => None,
    }
}
/// Points first: they sit on their curves, so a curve would otherwise always win.
fn measure_target(
    geometry: &oblikovati_model::sketch::Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<MeasurePick>, String> {
    if let Some(point) = measure_point(geometry, plane, pixel, window)? {
        return Ok(Some(point));
    }
    let picked = pick_witness(geometry, plane, pixel, window, SketchPickFilter::Curves)?;
    Ok(picked.map(|witness| MeasurePick::Curve(witness.entity)))
}
fn measure_point(
    geometry: &oblikovati_model::sketch::Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<MeasurePick>, String> {
    let points = geometry
        .points()
        .iter()
        .map(|point| Ok(point.position))
        .collect();
    if let Some(point) = first_near(plane, pixel, VERTEX_REACH, points, window)? {
        return Ok(Some(MeasurePick::Point(point)));
    }
    let lines = geometry
        .lines()
        .iter()
        .map(|line| line_middle(geometry, line))
        .collect();
    if let Some(point) = first_near(plane, pixel, MIDPOINT_REACH, lines, window)? {
        return Ok(Some(MeasurePick::Middle(point)));
    }
    let arcs = geometry
        .arcs()
        .iter()
        .map(|arc| arc_middle(geometry, arc))
        .collect();
    Ok(first_near(plane, pixel, MIDPOINT_REACH, arcs, window)?.map(MeasurePick::Middle))
}
fn line_middle(
    geometry: &oblikovati_model::sketch::Sketch2D,
    line: &oblikovati_model::sketch::SketchLine2D,
) -> Result<[f64; 2], String> {
    let [start, end] = line.endpoints.map(|point| geometry.point(point));
    let start = start.map_err(|cause| cause.to_string())?.position;
    let end = end.map_err(|cause| cause.to_string())?.position;
    Ok([(start[0] + end[0]) * 0.5, (start[1] + end[1]) * 0.5])
}
/// The point halfway along an arc, the way it runs.
fn arc_middle(
    geometry: &oblikovati_model::sketch::Sketch2D,
    arc: &oblikovati_model::sketch::SketchArc2D,
) -> Result<[f64; 2], String> {
    let position = |point| {
        geometry
            .point(point)
            .map(|point| point.position)
            .map_err(|cause| cause.to_string())
    };
    let (center, start, end) = (
        position(arc.center)?,
        position(arc.start)?,
        position(arc.end)?,
    );
    let from = (start[1] - center[1]).atan2(start[0] - center[0]);
    let to = (end[1] - center[1]).atan2(end[0] - center[0]);
    let turn = if arc.counter_clockwise {
        (to - from).rem_euclid(std::f64::consts::TAU)
    } else {
        -(from - to).rem_euclid(std::f64::consts::TAU)
    };
    let radius = (start[0] - center[0]).hypot(start[1] - center[1]);
    let middle = from + turn * 0.5;
    Ok([
        center[0] + radius * middle.cos(),
        center[1] + radius * middle.sin(),
    ])
}
#[cfg(test)]
#[path = "../tests/owner/measure_projection.rs"]
mod projection_tests;
