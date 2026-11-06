// SPDX-License-Identifier: GPL-2.0-only
//! Exact points for a running command, and what the sketch shows of its relations.
use super::*;
use crate::{placement_input::SketchPlacementState, sketch_tool::SketchToolState};
use oblikovati_gui::GuiPrecisePoint;
use oblikovati_model::{
    quantity::Dimension,
    sketch::{PlanarConstraint, Sketch2D, SketchEntityId},
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
    /// Example: `owner.precise_point_from_gui(&point)` with `x` "5" and `y` "6".
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
        let position = [
            self.typed_length(tool.document, &point.x)?,
            self.typed_length(tool.document, &point.y)?,
        ];
        if tool.tool != crate::SketchPlacementTool::Line {
            return Err(format!(
                "precise point {point:?} for {}: expected the Line command",
                tool.tool.name()
            ));
        }
        self.line_place_exact(tool, position)
    }
    fn typed_length(&self, document: u64, text: &str) -> Result<f64, String> {
        self.files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .parameters
            .evaluate_scalar_expression(text, Dimension::Length)
            .map(|quantity| quantity.value)
            .map_err(|cause| format!("coordinate {text:?}: expected a length: {cause}"))
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
