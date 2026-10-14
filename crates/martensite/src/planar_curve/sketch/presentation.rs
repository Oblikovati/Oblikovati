// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchLineDomain, SketchSamplingOptionsV1, sampling, shape};
use oblikovati_model::sketch::{OwnedSketchGeometry, Sketch2D, SketchEntityId, SketchReadIndex};
/// Native samples are display coordinates only; they never define model incidence or identity.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchDisplayPrimitive {
    Point([f64; 2]),
    Curve(Vec<[f64; 2]>),
}
/// One visible authored entity, with its independent construction and axis roles.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchDisplayEntity {
    pub entity: SketchEntityId,
    pub construction: bool,
    pub centerline: bool,
    pub primitive: SketchDisplayPrimitive,
}
/// A local display failure retains its model identity and does not hide other entities.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchDisplayDecline {
    pub entity: SketchEntityId,
    pub cause: String,
}
/// Complete display result in saved entity order, including every failed conversion.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct SketchDisplayReport {
    pub entities: Vec<SketchDisplayEntity>,
    pub declines: Vec<SketchDisplayDecline>,
    pub center_markers: Vec<oblikovati_model::sketch::SketchPoint2D>,
}
/// Sample committed geometry through the same native curve path as placement preview.
/// Example: `sample_sketch_display(&sketch, options)?` before publishing a sketch overlay.
pub fn sample_sketch_display(
    sketch: &Sketch2D,
    options: SketchSamplingOptionsV1,
) -> Result<SketchDisplayReport, String> {
    sampling::require_sampling_options(options)?;
    let index = sketch.read_index();
    let mut report = SketchDisplayReport {
        center_markers: index
            .implicit_circular_centers()
            .map_err(|cause| cause.to_string())?,
        ..Default::default()
    };
    for &entity in index.entity_ids() {
        match display_entity(&index, entity, options) {
            Ok(value) => report.entities.push(value),
            Err(cause) => report.declines.push(SketchDisplayDecline { entity, cause }),
        }
    }
    Ok(report)
}
fn display_entity(
    index: &SketchReadIndex<'_>,
    entity: SketchEntityId,
    options: SketchSamplingOptionsV1,
) -> Result<SketchDisplayEntity, String> {
    let primitive = display_primitive(index, entity, options)?;
    Ok(SketchDisplayEntity {
        entity,
        construction: index
            .is_construction(entity)
            .map_err(|cause| cause.to_string())?,
        centerline: index
            .is_centerline(entity)
            .map_err(|cause| cause.to_string())?,
        primitive,
    })
}

pub(super) fn display_primitive(
    index: &SketchReadIndex<'_>,
    entity: SketchEntityId,
    options: SketchSamplingOptionsV1,
) -> Result<SketchDisplayPrimitive, String> {
    match index.entity(entity).map_err(|cause| cause.to_string())? {
        OwnedSketchGeometry::Point(point) => Ok(SketchDisplayPrimitive::Point(point.position)),
        _ => {
            let curve =
                shape::resolve(index, entity, SketchLineDomain::Segment, options.deflection)
                    .map_err(|cause| cause.to_string())?;
            sampling::sample_definition(&curve.definition, options)
                .map(SketchDisplayPrimitive::Curve)
        }
    }
}
