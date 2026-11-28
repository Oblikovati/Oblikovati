// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Derived Part features against external source Part snapshots.
use crate::{
    BlendNamedTopology, CertifiedNamedSolidEvaluator, EvaluationEntity, EvaluationFailure,
    EvaluationFailureKind, EvaluationPhase, NamedSolid,
};
use martensite_tkbrep::topods::Shape;
use martensite_tkmath::gp::{Axis2, Direction3, Point3, Transform3};
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPIMakeShape, BRepBuilderAPITransform};
use oblikovati_model::{
    feature::{
        FeatureId,
        derived_part::{DeriveMirrorPlane, ResolvedDerivedPart},
        part::PartFeatureFailure,
    },
    history::SnapshotReplay,
    identity::DocumentGuid,
};

const SOURCE_DOC_GUID: &str = "00000000-0000-4000-8000-00000000d3a1";
const SCALE_EPSILON: f64 = 1e-12;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(crate) fn evaluate_named_derived_part(
        &mut self,
        feature: FeatureId,
        input: ResolvedDerivedPart,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |kind: EvaluationFailureKind, message: String| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(kind, EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        if !input.include_bodies {
            return Err(failure(
                EvaluationFailureKind::InvalidResult,
                "derive: expected at least one included solid body".into(),
            ));
        }
        let sources = self.derived_sources.ok_or_else(|| {
            failure(
                EvaluationFailureKind::InvalidReference,
                format!(
                    "derive source {:?}: expected resolved source snapshots on evaluator",
                    input.source_path
                ),
            )
        })?;
        let source_snapshot = sources.get(&input.source_path).ok_or_else(|| {
            failure(
                EvaluationFailureKind::InvalidReference,
                format!(
                    "derive source {:?}: source part snapshot was not loaded",
                    input.source_path
                ),
            )
        })?;
        let source_replay = SnapshotReplay::new(source_snapshot)
            .map_err(|cause| failure(EvaluationFailureKind::InvalidReference, cause.to_string()))?;
        let source_guid: DocumentGuid = SOURCE_DOC_GUID.parse().map_err(|cause| {
            failure(EvaluationFailureKind::InvalidReference, format!("{cause}"))
        })?;
        let policy = self.resolution_policy();
        let mut nested = CertifiedNamedSolidEvaluator::with_optional_policy(
            source_replay.history(),
            source_guid,
            policy,
        );
        let evaluation = nested
            .evaluate_private_candidate(source_replay.history().snapshot())
            .map_err(|cause| {
                failure(
                    EvaluationFailureKind::InvalidResult,
                    format!("derive source {:?}: {cause}", input.source_name),
                )
            })?;
        let source_body = evaluation
            .ordered_bodies()
            .find(|body| !body.is_surface())
            .ok_or_else(|| {
                failure(
                    EvaluationFailureKind::InvalidResult,
                    format!(
                        "derive source {:?}: expected at least one solid body",
                        input.source_name
                    ),
                )
            })?;
        let raw_shape = source_body
            .native_shape(source_replay.history(), source_guid)
            .map_err(|cause| failure(EvaluationFailureKind::InvalidResult, cause))?;
        let resolution = match source_body {
            NamedSolid::Prism(prism) => prism.certification().map(|item| item.resolution),
            NamedSolid::Derived(derived) => derived.resolution(),
        };
        let transformed =
            transform_derived_shape(raw_shape, input.scale, input.mirror).map_err(|cause| {
                failure(
                    EvaluationFailureKind::InvalidResult,
                    format!("derive transform: {cause}"),
                )
            })?;
        let topology = BlendNamedTopology::from_derived(
            self.history,
            self.document,
            feature,
            transformed,
            resolution,
        )
        .map_err(|cause| {
            failure(
                EvaluationFailureKind::InvalidReference,
                format!("derive topology: {cause}"),
            )
        })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
}

fn transform_derived_shape(
    source: &Shape,
    scale: f64,
    mirror: Option<DeriveMirrorPlane>,
) -> Result<Shape, String> {
    let mut copier = crate::checked_shape::copy_shape(source, "derived source")
        .map_err(|cause| cause.to_string())?;
    let mut shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    if (scale - 1.0).abs() > SCALE_EPSILON {
        let mut scale_trsf = Transform3::default();
        scale_trsf
            .set_scale(Point3::default(), scale)
            .map_err(|cause| cause.to_string())?;
        let mut scaled = BRepBuilderAPITransform::with_shape(&shape, scale_trsf, true, false)
            .map_err(|cause| cause.to_string())?;
        shape = scaled.shape().map_err(|cause| cause.to_string())?.clone();
    }
    if let Some(plane) = mirror {
        let [nx, ny, nz] = match plane {
            DeriveMirrorPlane::XY => [0.0, 0.0, 1.0],
            DeriveMirrorPlane::YZ => [1.0, 0.0, 0.0],
            DeriveMirrorPlane::XZ => [0.0, 1.0, 0.0],
        };
        let direction = Direction3::new(nx, ny, nz).map_err(|cause| cause.to_string())?;
        let frame = Axis2::new(Point3::default(), direction).map_err(|cause| cause.to_string())?;
        let mirror_trsf = Transform3::mirror_plane(frame);
        let mut mirrored = BRepBuilderAPITransform::with_shape(&shape, mirror_trsf, true, false)
            .map_err(|cause| cause.to_string())?;
        shape = mirrored.shape().map_err(|cause| cause.to_string())?.clone();
    }
    crate::checked_shape::require_geometry(&shape, "derived result")
        .map_err(|cause| cause.to_string())?;
    crate::checked_shape::analytic_properties(&shape, "derived result")
        .map_err(|cause| cause.to_string())?;
    Ok(shape)
}
