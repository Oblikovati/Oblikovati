// SPDX-License-Identifier: GPL-2.0-only
//! The sketch's inspection commands on the owner: Measure and Region Properties.
use super::sketch_picking::{SketchPickFilter, pick_witness};
use super::*;
use crate::{
    HeadWindow,
    sketch_inspect::{
        measure::{MeasureContext, MeasurePick, SketchMeasureState, measure_rows},
        region_properties::{SketchRegionState, region_result_rows},
    },
    sketch_tool::SketchToolState,
};
use oblikovati_gui::{GuiInspectAction, GuiMeasurePanel, GuiRegionPanel};
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};

/// A line's midpoint is a selection of its own within this reach of the pointer, in pixels.
const MIDPOINT_REACH: f64 = 6.;

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
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .edit_sketch_dimension(
                oblikovati_model::document_sketch::DocumentSketchId(value.sketch),
                oblikovati_model::sketch::SketchConstraintId(value.constraint),
                &oblikovati_model::history::SketchDimensionEdit {
                    target_expression: Some(value.expression.clone()),
                    reference: None,
                    drive_limits: None,
                },
            )
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    /// Carry out one inspection choice; example: `owner.inspect_from_gui(StartMeasure)`.
    pub fn inspect_from_gui(&mut self, action: GuiInspectAction) -> Result<bool, String> {
        match action {
            GuiInspectAction::StartMeasure => self.start_measure()?,
            GuiInspectAction::OpenRegionProperties => self.open_region_properties()?,
            GuiInspectAction::CalculateRegions => self.calculate_regions()?,
            GuiInspectAction::FinishMeasure | GuiInspectAction::FinishRegionProperties => {
                self.sketch_tool = None;
                self.command_notice = Some(DesktopCommandNotice::Ready);
            }
        }
        Ok(true)
    }
    /// The M key at Ready starts Measure; while a command runs the key is that command's.
    pub(super) fn measure_key(&mut self, event: ViewportEvent) -> Result<bool, String> {
        let ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::M,
            pressed: true,
            modifiers,
            ..
        }) = event
        else {
            return Ok(false);
        };
        if modifiers.control || self.active_sketch_edit().is_none() {
            return Ok(false);
        }
        self.start_measure()?;
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
        let Some(SketchToolState::Measure(tool)) = self.sketch_tool.as_ref() else {
            return None;
        };
        Some(GuiMeasurePanel {
            rows: tool.rows.clone(),
        })
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
        let Some(pick) = measure_target(geometry, tool.edit.plane, pixel, window)? else {
            return Ok(false);
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
/// Midpoints first: they sit on their curve, so the curve would otherwise always win.
fn measure_target(
    geometry: &oblikovati_model::sketch::Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<MeasurePick>, String> {
    for line in geometry.lines() {
        let [start, end] = line.endpoints.map(|point| geometry.point(point));
        let (start, end) = (
            start.map_err(|cause| cause.to_string())?.position,
            end.map_err(|cause| cause.to_string())?.position,
        );
        let middle = [(start[0] + end[0]) * 0.5, (start[1] + end[1]) * 0.5];
        let at = window.sketch_pixel(plane, middle)?;
        if super::sketch_picking::screen_distance(at, pixel) <= MIDPOINT_REACH {
            return Ok(Some(MeasurePick::Point(middle)));
        }
    }
    let Some(witness) = pick_witness(geometry, plane, pixel, window, SketchPickFilter::Any)? else {
        return Ok(None);
    };
    Ok(Some(match witness.curve {
        Some(_) => MeasurePick::Curve(witness.entity),
        None => MeasurePick::Point(witness.position),
    }))
}
