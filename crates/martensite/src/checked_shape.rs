// SPDX-License-Identifier: GPL-2.0-only
//! Shared checked operations preserve native causes before legacy callers render text.
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, ReleaseProbeProperties,
};
use martensite_tkbrep::topods::Shape;
use martensite_tkgeombase::gprop::GProps;
use martensite_tktopalgo::{
    brep_builder_api::{BRepBuilderAPICopy, BRepBuilderAPIMakeShape},
    brep_check::BRepCheckAnalyzer,
    brep_gprop::BRepGProp,
};
/// Retain mapped validity failures without widening the check; example: `require_geometry(shape, "prism")?`.
pub(crate) fn require_geometry(shape: &Shape, context: &str) -> Result<(), EvaluationFailure> {
    let valid = BRepCheckAnalyzer::new(shape, true, false, true)
        .and_then(|checker| checker.is_valid())
        .map_err(|cause| {
            Phase::Validation.native(
                Entity::Body,
                format!("{context}: expected geometric validity"),
                cause,
            )
        })?;
    if !valid {
        return Err(Phase::Validation.refused(
            Kind::InvalidGeometry,
            Entity::Body,
            format!("{context}: expected a geometrically valid B-rep"),
        ));
    }
    Ok(())
}
/// Detach the same native topology, geometry, and mesh; example: `copy_shape(shape, "copy")?`.
pub(crate) fn copy_shape(
    source: &Shape,
    context: &str,
) -> Result<BRepBuilderAPICopy, EvaluationFailure> {
    // copy_geom=true is required: a Shape clone retains mutable geometry handles.
    let mut copier = BRepBuilderAPICopy::with_shape(source, true, true).map_err(|cause| {
        Phase::Copy.native(
            Entity::Body,
            format!("{context}: expected detached topology/geometry/mesh"),
            cause,
        )
    })?;
    require_detached_copy(&mut copier, source, context)?;
    Ok(copier)
}
fn require_detached_copy(
    copier: &mut BRepBuilderAPICopy,
    source: &Shape,
    context: &str,
) -> Result<(), EvaluationFailure> {
    let shape = copier.shape().map_err(|cause| {
        Phase::Copy.native(
            Entity::Body,
            format!("{context}: expected a completed B-rep"),
            cause,
        )
    })?;
    if shape.is_partner(source) {
        return Err(Phase::Copy.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("{context}: expected detached root topology"),
        ));
    }
    Ok(())
}
/// Retain native analytic integration causes; example: `volume_properties(shape, "prism")?`.
pub(crate) fn volume_properties(shape: &Shape, context: &str) -> Result<GProps, EvaluationFailure> {
    let mut properties = GProps::new();
    // Analytic integration explicitly disables the kernel's triangulation path.
    BRepGProp::volume_properties(shape, &mut properties, true, false, false).map_err(|cause| {
        Phase::Measurement.native(
            Entity::Body,
            format!("{context}: expected analytic properties"),
            cause,
        )
    })?;
    Ok(properties)
}
/// Check finite analytic results after native integration; example: `analytic_properties(shape, "prism")?`.
pub(crate) fn analytic_properties(
    shape: &Shape,
    context: &str,
) -> Result<ReleaseProbeProperties, EvaluationFailure> {
    let properties = volume_properties(shape, context)?;
    crate::analytic_body::geometry_integrals(&properties, context)
        .map_err(|message| Phase::Measurement.refused(Kind::InvalidResult, Entity::Body, message))
}
