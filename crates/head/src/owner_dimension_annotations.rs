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
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        for sketch in snapshot.sketches.sketches() {
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
        oblikovati_model::parameter::ParameterHealth::Healthy => {
            format!("{} = {value}", state.name)
        }
        oblikovati_model::parameter::ParameterHealth::Failed(_) => {
            format!("{}: {value} (source unavailable)", state.name)
        }
    };
    Ok(Some(HeadDimensionAnnotation {
        plane,
        position,
        text,
        role,
    }))
}
