// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadDimensionAnnotation, HeadDimensionAnnotations};
use oblikovati_gui::GuiDimensionLabelRole;
use oblikovati_model::{document_sketch::PlacedSketch, history::DocumentSnapshot};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn dimension_annotations(&self) -> Result<HeadDimensionAnnotations, String> {
        let mut output = HeadDimensionAnnotations::default();
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(output);
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let mut called_up = session.selected_browser_sketches();
        called_up.extend(self.sketch_opened_while_hidden());
        for sketch in snapshot
            .sketches
            .sketches()
            .iter()
            .filter(|sketch| crate::sketch_scene::sketch_shows(sketch, &called_up))
        {
            collect_dimension_annotations(snapshot, sketch, &mut output);
        }
        match self.dimension_preview_annotation() {
            Ok(Some(label)) => output.labels.push(label),
            Ok(None) => {}
            Err(cause) => output.declines.push(cause),
        }
        Ok(output)
    }
}
/// The stored text; a bare typed number reads with the unit the document gave it.
fn stored_expression(
    snapshot: &DocumentSnapshot,
    state: &oblikovati_model::parameter::ParameterState<u64>,
    dimension: oblikovati_model::quantity::Dimension,
) -> String {
    if state.source.trim().parse::<f64>().is_err() {
        return state.source.clone();
    }
    let unit = snapshot.parameters.units().preferred_name(dimension);
    format!("{} {unit}", state.source.trim())
}
/// A sketch reads a length without its document unit and an angle with two decimals.
fn annotation_value(quantity: oblikovati_model::quantity::Quantity, shown: String) -> String {
    if quantity.dimension == oblikovati_model::quantity::Dimension::Angle {
        return format!("{:.2}", quantity.value.to_degrees());
    }
    let number = shown.split_whitespace().next().unwrap_or(&shown);
    if !number.contains('.') {
        return number.to_owned();
    }
    number
        .trim_end_matches('0')
        .trim_end_matches('.')
        .to_owned()
}
fn collect_dimension_annotations(
    snapshot: &DocumentSnapshot,
    sketch: &PlacedSketch,
    output: &mut HeadDimensionAnnotations,
) {
    for (constraint, parameter) in sketch.geometry().dimensional_constraints() {
        match dimension_annotation(snapshot, sketch, constraint.id, parameter) {
            Ok(Some(label)) => output.labels.push(label),
            Ok(None) => {}
            Err(cause) => output.declines.push(format!(
                "sketch {:?}, dimension {:?}: {cause}",
                sketch.id(),
                constraint.id
            )),
        }
    }
}
fn dimension_annotation(
    snapshot: &DocumentSnapshot,
    sketch: &PlacedSketch,
    id: oblikovati_model::sketch::SketchConstraintId,
    parameter: u64,
) -> Result<Option<HeadDimensionAnnotation>, String> {
    let geometry = sketch.geometry();
    let Some(position) = geometry
        .dimension_text_point(id)
        .map_err(|cause| cause.to_string())?
    else {
        return Ok(None);
    };
    let plane = snapshot
        .resolve_sketch_plane(sketch.id())
        .map_err(|cause| cause.to_string())?;
    let quantity = geometry
        .measure_dimension(id)
        .map_err(|cause| cause.to_string())?;
    let value = snapshot
        .parameters
        .units()
        .format_display(quantity)
        .map_err(|cause| cause.to_string())?;
    let state = snapshot
        .parameters
        .get(parameter)
        .ok_or_else(|| format!("dimension parameter {parameter}: expected an owned source"))?;
    let role = if geometry
        .dimension_is_reference(id)
        .map_err(|cause| cause.to_string())?
    {
        GuiDimensionLabelRole::Driven
    } else {
        GuiDimensionLabelRole::Driving
    };
    let text = match &state.health {
        oblikovati_model::parameter::ParameterHealth::Healthy => annotation_value(quantity, value),
        oblikovati_model::parameter::ParameterHealth::Failed(_) => {
            format!("{}: {value} (source unavailable)", state.name)
        }
    };
    let edit = (role == GuiDimensionLabelRole::Driving).then(|| oblikovati_gui::GuiDimensionEdit {
        sketch: sketch.id().0,
        constraint: id.0,
        name: state.name.clone(),
        expression: stored_expression(snapshot, state, quantity.dimension),
    });
    Ok(Some(HeadDimensionAnnotation {
        plane,
        position,
        text,
        role,
        edit,
    }))
}
