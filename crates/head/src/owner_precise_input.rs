// SPDX-License-Identifier: GPL-2.0-only
//! Exact points for a running command, and what the sketch shows of its relations.
use super::*;
use crate::HeadWindow;
use crate::{placement_input::SketchPlacementState, sketch_tool::SketchToolState};
use oblikovati_gui::{GuiPrecisePoint, GuiTriadCommand};
use oblikovati_model::{
    quantity::Dimension,
    sketch::{PlanarConstraint, Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Place the running command's next point at typed coordinates, in the document's units.
    /// Example: `owner.precise_point_from_gui(&point)` with X "5" and Y "6".
    pub fn precise_point_from_gui(&mut self, point: &GuiPrecisePoint) -> Result<bool, String> {
        let Some(SketchToolState::Placement(mut tool)) = self.sketch_tool.take() else {
            return Err(format!(
                "precise point {point:?}: expected a running sketch command"
            ));
        };
        let result = self.place_precise_point(&mut tool, point);
        self.sketch_tool = Some(SketchToolState::Placement(tool));
        result.map(|()| true)
    }
    fn place_precise_point(
        &mut self,
        tool: &mut SketchPlacementState,
        point: &GuiPrecisePoint,
    ) -> Result<(), String> {
        let position = self.precise_position(tool.document, point)?;
        match tool.tool {
            crate::SketchPlacementTool::Line => self.line_place_exact(tool, position),
            crate::SketchPlacementTool::Point => self.commit_point(tool, position, None),
            _ => self.place_exact_point(tool, position),
        }
    }
    fn precise_position(&self, document: u64, point: &GuiPrecisePoint) -> Result<[f64; 2], String> {
        use oblikovati_gui::GuiPreciseMode::{DistanceAndAngle, XAndAngle, XAndY, YAndAngle};
        let first_is_length = self.typed_value(document, &point.first, Dimension::Length)?;
        let second_dimension = match point.mode {
            XAndY => Dimension::Length,
            _ => Dimension::Angle,
        };
        let second = self.typed_value(document, &point.second, second_dimension)?;
        let offset = match point.mode {
            XAndY => [first_is_length, second],
            XAndAngle => [first_is_length, first_is_length * second.tan()],
            YAndAngle => [first_is_length / second.tan(), first_is_length],
            DistanceAndAngle => [
                first_is_length * second.cos(),
                first_is_length * second.sin(),
            ],
        };
        if offset.iter().any(|value| !value.is_finite()) {
            return Err(format!(
                "precise point {point:?}: expected an angle that reaches a point"
            ));
        }
        let origin = self
            .sketch_memory
            .triad
            .filter(|_| point.relative)
            .unwrap_or_default();
        Ok([origin[0] + offset[0], origin[1] + offset[1]])
    }
    fn typed_value(&self, document: u64, text: &str, dimension: Dimension) -> Result<f64, String> {
        self.files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .parameters
            .evaluate_scalar_expression(text, dimension)
            .map(|quantity| quantity.value)
            .map_err(|cause| format!("value {text:?}: expected a {dimension:?}: {cause}"))
    }
    /// Place one point and stay ready for the next, as the command does on every click.
    pub(super) fn commit_point(
        &mut self,
        tool: &mut SketchPlacementState,
        position: [f64; 2],
        coincident_with: Option<SketchEntityId>,
    ) -> Result<(), String> {
        let placement = tool.point_placement(position, coincident_with);
        self.files
            .commands(tool.document)
            .map_err(|cause| cause.to_string())?
            .create_sketch_point(tool.sketch, &placement)
            .map_err(|cause| format!("point {position:?}: {cause}"))?;
        let generation = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?
            .generation();
        tool.restart(generation);
        self.placement_notice(tool.prompt());
        Ok(())
    }
    /// A pointer click for Point: landing on a vertex shares its position and adds a relation.
    pub(super) fn place_point_at_pointer(
        &mut self,
        tool: &mut SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
    ) -> Result<(), String> {
        let Some(at) = window.sketch_point(pixel, tool.plane)? else {
            return Ok(());
        };
        let acquired = self.vertex_under_pointer(tool, window, pixel)?;
        let position = acquired.map_or(at, |(position, _)| position);
        self.commit_point(tool, position, acquired.map(|(_, vertex)| vertex))
    }
    /// What the toolbar acts on: any placement command's next point.
    pub(super) fn gui_precise_target(&self) -> Option<oblikovati_gui::GuiPreciseTarget> {
        let tool = self.placement_state()?;
        Some(oblikovati_gui::GuiPreciseTarget {
            pointer: self.precise_pointer(tool),
            triad: None,
        })
    }
    fn precise_pointer(&self, tool: &SketchPlacementState) -> Option<[String; 2]> {
        let at = tool.pointer_position()?;
        let session = self.files.workspace().session(tool.document).ok()?;
        let units = session.snapshot().parameters.units();
        let shown = at.map(|value| {
            let quantity = oblikovati_model::quantity::Quantity::new(value, Dimension::Length);
            // The display form ends in its unit; a field holds the bare number.
            units
                .format_display(quantity)
                .ok()
                .and_then(|text| text.split_whitespace().next().map(str::to_owned))
        });
        let [x, y] = shown;
        Some([x?, y?])
    }
    /// Where the triad sits on the running command's plane; example: for the frame to project.
    pub fn precise_triad_position(&self) -> Option<(SketchPlane, [f64; 2])> {
        let tool = self.placement_state()?;
        Some((tool.plane, self.sketch_memory.triad.unwrap_or_default()))
    }
    /// Move the triad by two picks, or put it back; example: from the toolbar's buttons.
    pub fn precise_triad_from_gui(&mut self, command: GuiTriadCommand) -> Result<bool, String> {
        match command {
            GuiTriadCommand::ResetToOrigin => self.sketch_memory.triad = None,
            GuiTriadCommand::Reposition => {
                self.sketch_memory.triad_pick = Some(TriadPick::Source);
                self.placement_notice("Select the triad's origin".into());
            }
        }
        Ok(true)
    }
    /// While the triad is being moved, a press belongs to it; Escape returns to the command.
    pub(super) fn triad_pick_event(
        &mut self,
        tool: &mut SketchPlacementState,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some(pick) = self.sketch_memory.triad_pick else {
            return Ok(false);
        };
        if cancels_pick(event) {
            self.sketch_memory.triad_pick = None;
            self.placement_notice(tool.prompt());
            return Ok(true);
        }
        let Some(pixel) = left_press(event) else {
            return Ok(false);
        };
        self.sketch_memory.triad_pick = match pick {
            TriadPick::Source => Some(TriadPick::Target),
            TriadPick::Target => {
                self.move_triad(tool, window, pixel)?;
                None
            }
        };
        let prompt = match self.sketch_memory.triad_pick {
            Some(_) => "Select the triad's new position".to_owned(),
            None => tool.prompt(),
        };
        self.placement_notice(prompt);
        Ok(true)
    }
    fn move_triad(
        &mut self,
        tool: &SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
    ) -> Result<(), String> {
        let acquired = self.vertex_under_pointer(tool, window, pixel)?;
        let at = acquired
            .map(|(position, _)| position)
            .or(window.sketch_point(pixel, tool.plane)?);
        self.sketch_memory.triad = at;
        Ok(())
    }
    /// Change what the next points are; a running Point command asks accordingly.
    pub fn toggle_center_point_format_from_gui(&mut self) -> Result<bool, String> {
        self.sketch_memory.center_point = !self.sketch_memory.center_point;
        let center = self.sketch_memory.center_point;
        let Some(SketchToolState::Placement(tool)) = self.sketch_tool.as_mut() else {
            return Ok(true);
        };
        tool.center_point = center;
        let prompt = tool.prompt();
        self.refresh_point_role_preview()?;
        self.placement_notice(prompt);
        Ok(true)
    }
    /// Change one point's role as a history step; example: from the canvas menu.
    pub fn set_center_point_from_gui(&mut self, point: u64, center: bool) -> Result<bool, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("point {point} center {center}: expected an edited sketch"))?;
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_center_point(edit.sketch, SketchEntityId(point), center)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    /// Remember what a right press landed on, for the canvas menu its release opens.
    pub(super) fn note_sketch_context(
        &mut self,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<(), String> {
        let ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Right,
            pressed: true,
            position,
            ..
        }) = event
        else {
            return Ok(());
        };
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(());
        };
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let geometry = snapshot
            .sketches
            .get(edit.sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        let target = context_target(geometry, edit.plane, position, window)?;
        self.sketch_memory.context = Some(target);
        Ok(())
    }
    pub(super) fn gui_sketch_context(&self) -> Option<oblikovati_gui::GuiSketchContext> {
        let target = self.sketch_memory.context.as_ref()?;
        let edit = self.active_sketch_edit()?;
        let session = self.files.workspace().session(edit.document).ok()?;
        let sketch = session.snapshot().sketches.get(edit.sketch).ok()?;
        let point = target
            .point
            .filter(|point| sketch.geometry().point(*point).is_ok());
        Some(oblikovati_gui::GuiSketchContext {
            point: point.map(|point| point.0),
            center_point: point.is_some_and(|point| sketch.geometry().is_center_point(point)),
            candidates: target.candidates.clone(),
        })
    }
    /// Show or hide the edited sketch's relation marks; the choice is a history step.
    pub(super) fn set_constraints_shown(&mut self, shown: bool) -> Result<(), String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("constraints shown {shown}: expected an edited sketch"))?;
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_constraints_shown(edit.sketch, shown)
            .map_err(|cause| cause.to_string())
    }
    /// Each shown relation with the sketch point its mark sits at, for the frame to project.
    pub fn sketch_constraint_marks(&self) -> Vec<HeadConstraintMark> {
        let Some(edit) = self.active_sketch_edit() else {
            return Vec::new();
        };
        let Ok(session) = self.files.workspace().session(edit.document) else {
            return Vec::new();
        };
        let Ok(sketch) = session.snapshot().sketches.get(edit.sketch) else {
            return Vec::new();
        };
        if !sketch.constraints_shown() {
            return Vec::new();
        }
        relation_marks(sketch.geometry())
            .into_iter()
            .map(|(at, text)| HeadConstraintMark {
                plane: edit.plane,
                at,
                text,
            })
            .collect()
    }
}
/// What the running sketch commands remember between events.
pub(super) struct SketchCommandMemory {
    /// New points are center points, unless a person turned that off.
    pub center_point: bool,
    pub repeat_tool: Option<oblikovati_gui::GuiSketchTool>,
    /// Where relative values are measured from; none means the sketch origin.
    pub triad: Option<[f64; 2]>,
    pub triad_pick: Option<TriadPick>,
    pub context: Option<SketchContextTarget>,
}
impl Default for SketchCommandMemory {
    fn default() -> Self {
        Self {
            center_point: true,
            repeat_tool: None,
            triad: None,
            triad_pick: None,
            context: None,
        }
    }
}
/// The two picks that move the triad.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum TriadPick {
    Source,
    Target,
}
/// What a right press landed on.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct SketchContextTarget {
    point: Option<SketchEntityId>,
    candidates: Vec<String>,
}
/// A right press acts on a point within this reach, in pixels.
const CONTEXT_POINT_REACH: f64 = 6.;

fn context_target(
    geometry: &Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<SketchContextTarget, String> {
    let mut within = Vec::new();
    for point in geometry.points() {
        let at = window.sketch_pixel(plane, point.position)?;
        let distance = super::sketch_picking::screen_distance(at, pixel);
        if distance <= CONTEXT_POINT_REACH {
            within.push((distance, point.id));
        }
    }
    within.sort_by(|first, second| first.0.total_cmp(&second.0).then(first.1.cmp(&second.1)));
    Ok(SketchContextTarget {
        point: within.first().map(|(_, id)| *id),
        candidates: vec!["Point".to_owned(); within.len()],
    })
}
fn cancels_pick(event: ViewportEvent) -> bool {
    matches!(
        event,
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::Escape,
            pressed: true,
            ..
        }) | ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Cancel,
            pressed: true,
        })
    )
}
fn left_press(event: ViewportEvent) -> Option<[i32; 2]> {
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
/// One relation's mark as a sketch point on its plane, ready for a window to project.
#[derive(Clone, Debug, PartialEq)]
pub struct HeadConstraintMark {
    pub plane: oblikovati_model::sketch::SketchPlane,
    pub at: [f64; 2],
    pub text: String,
}
fn relation_marks(geometry: &Sketch2D) -> Vec<([f64; 2], String)> {
    geometry
        .constraints()
        .iter()
        .filter_map(|constraint| relation_mark(geometry, &constraint.relation))
        .collect()
}
fn relation_mark(geometry: &Sketch2D, relation: &PlanarConstraint) -> Option<([f64; 2], String)> {
    let (at, text) = match relation {
        PlanarConstraint::Coincident { points } => (point_at(geometry, points[0])?, "C"),
        PlanarConstraint::LineHorizontal { line } => (line_middle(geometry, *line)?, "H"),
        PlanarConstraint::LineVertical { line } => (line_middle(geometry, *line)?, "V"),
        PlanarConstraint::Lines { lines, .. } => (line_middle(geometry, lines[0])?, "//"),
        _ => return None,
    };
    Some((at, text.to_owned()))
}
fn point_at(geometry: &Sketch2D, point: SketchEntityId) -> Option<[f64; 2]> {
    geometry.point(point).ok().map(|point| point.position)
}
fn line_middle(geometry: &Sketch2D, line: SketchEntityId) -> Option<[f64; 2]> {
    let [start, end] = geometry.line(line).ok()?.endpoints;
    let (start, end) = (point_at(geometry, start)?, point_at(geometry, end)?);
    Some([(start[0] + end[0]) * 0.5, (start[1] + end[1]) * 0.5])
}

#[cfg(test)]
mod tests {
    use super::*;
    fn sketch_with_a_horizontal_line() -> Sketch2D {
        let mut sketch = Sketch2D::default();
        let [start, end, line] = [1, 2, 3].map(SketchEntityId);
        sketch.add_point(start, [0., 0.]).unwrap();
        sketch.add_point(end, [4., 0.]).unwrap();
        sketch.add_line(line, [start, end]).unwrap();
        sketch
            .add_constraint(
                oblikovati_model::sketch::SketchConstraintId(1),
                PlanarConstraint::LineHorizontal { line },
            )
            .unwrap();
        sketch
    }
    #[test]
    fn a_horizontal_relation_marks_the_middle_of_its_line() {
        assert_eq!(
            relation_marks(&sketch_with_a_horizontal_line()),
            [([2., 0.], "H".to_owned())]
        );
    }
    #[test]
    fn a_relation_with_no_mark_yet_is_left_out() {
        let sketch = Sketch2D::default();
        assert!(relation_marks(&sketch).is_empty());
    }
}
