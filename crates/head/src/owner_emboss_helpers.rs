// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed formatting, geometry predicates, and recipe helpers for desktop Emboss.
use oblikovati_api::Error;
use oblikovati_automation::EmbossInputs;
use oblikovati_gui::{GuiEmbossEditSeed, GuiEmbossRequest};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        sketch_emboss::SketchEmbossDefinition,
        sketch_prism::SketchPrismSelection,
        work_geometry::{DatumAngle, DatumLength},
    },
    history::DocumentSnapshot,
    quantity::{Dimension, Quantity},
    sketch::SketchPlane,
};
use std::collections::BTreeMap;

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiEmbossEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::SketchEmboss(definition) = feature.definition() else {
                return None;
            };
            let depth = length_field(snapshot, definition.depth()).unwrap_or_else(|| "1 mm".into());
            let taper = taper_field(snapshot, definition.taper());
            Some((
                feature.id(),
                GuiEmbossEditSeed {
                    depth,
                    emboss_type: definition.emboss_type(),
                    direction_reversed: definition.direction_reversed(),
                    taper,
                    wrap_to_face: definition.wrap_to_face(),
                    wrap_face: definition.wrap_face().map(<[u8]>::to_vec),
                },
            ))
        })
        .collect()
}

fn length_field(snapshot: &DocumentSnapshot, length: DatumLength) -> Option<String> {
    match length.parameter() {
        Some(id) => Some(snapshot.parameters.get(id)?.name.clone()),
        None => {
            let value = length.resolve(&snapshot.parameters).ok()?;
            let units = snapshot.parameters.units();
            let preferred = units.to_preferred(Quantity::new(value, Dimension::Length));
            let unit = units.preferred_name(Dimension::Length);
            Some(format!("{} {unit}", format_number(preferred)))
        }
    }
}

fn taper_field(snapshot: &DocumentSnapshot, angle: DatumAngle) -> String {
    if let Some(id) = angle.parameter_id()
        && let Some(param) = snapshot.parameters.get(id)
    {
        return param.name.clone();
    }
    let degrees = angle
        .resolve(&snapshot.parameters)
        .map(f64::to_degrees)
        .unwrap_or(0.0);
    if degrees.abs() < 1e-9 {
        "0,0".into()
    } else {
        format!("{} deg", format_number(degrees))
    }
}

fn format_number(value: f64) -> String {
    let text = format!("{value:.4}");
    let trimmed = text.trim_end_matches('0').trim_end_matches('.');
    if trimmed == "-0" || trimmed.is_empty() {
        "0".into()
    } else {
        trimmed.into()
    }
}

pub(super) fn is_range_error(message: &str) -> bool {
    message.contains("out of the expected range")
}

pub(super) fn has_open_visible_sketch(snapshot: &DocumentSnapshot) -> bool {
    snapshot.sketches.sketches().iter().any(|sketch| {
        sketch.visible()
            && (!sketch.geometry().lines().is_empty()
                || !sketch.geometry().arcs().is_empty()
                || !sketch.geometry().splines().is_empty())
    })
}

pub(super) fn planes_coplanar(a: SketchPlane, b: SketchPlane) -> bool {
    let na = a.normal();
    let nb = b.normal();
    let dot = na[0] * nb[0] + na[1] * nb[1] + na[2] * nb[2];
    if dot < 0.9999 {
        return false;
    }
    let oa = a.recipe().origin;
    let ob = b.recipe().origin;
    let dist = (oa[0] - ob[0]) * na[0] + (oa[1] - ob[1]) * na[1] + (oa[2] - ob[2]) * na[2];
    dist.abs() < 1e-4
}

pub(super) fn emboss_inputs<'a>(
    request: &'a GuiEmbossRequest,
    sketch: DocumentSketchId,
    selection: SketchPrismSelection,
    wrap_face: Option<Vec<u8>>,
    target_body: Vec<u8>,
) -> EmbossInputs<'a> {
    EmbossInputs {
        sketch,
        selection,
        depth: &request.depth,
        emboss_type: request.emboss_type,
        direction_reversed: request.direction_reversed,
        taper: &request.taper,
        wrap_to_face: request.wrap_to_face,
        wrap_face,
        target_body,
    }
}

pub(super) fn emboss_definition(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<&SketchEmbossDefinition, String> {
    let authored = snapshot
        .work_geometry
        .get(id)
        .map_err(|cause| cause.to_string())?;
    let PartFeatureDefinition::SketchEmboss(definition) = authored.definition() else {
        return Err(format!("feature {id:?}: expected an Emboss"));
    };
    Ok(definition)
}

pub(super) fn replace_emboss(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SketchEmbossDefinition,
) -> Result<(), Error> {
    let reads = definition.reads(snapshot).map_err(Error)?;
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::SketchEmboss(Box::new(definition)),
            reads,
        )
        .map_err(|cause| Error(cause.to_string()))
        .map(|_| ())
}
