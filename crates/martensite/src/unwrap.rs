// SPDX-License-Identifier: GPL-2.0-only
//! Certified Unwrap evaluation: unfold selected solid faces into an oriented planar surface shell.
use crate::{
    BlendNamedTopology, CertifiedNamedSolidEvaluator, EvaluationEntity, EvaluationFailure,
    EvaluationFailureKind, EvaluationPhase, NamedSolid,
};
use martensite_tkbrep::topods::{Builder, Shape, Shell};
use martensite_tkmath::{bnd::BoundingBox3, gp::Point3};
use martensite_tktopalgo::{
    brep_bnd_lib::BRepBndLib,
    brep_builder_api::{BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon},
};
use oblikovati_model::feature::{
    FeatureId,
    part::PartFeatureFailure,
    unwrap::{ResolvedUnwrap, UnwrapAlignment},
};

impl CertifiedNamedSolidEvaluator<'_> {
    /// Evaluate an Unwrap feature against a live target solid, producing a planar Surface body.
    pub(crate) fn evaluate_named_unwrap(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedUnwrap<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |kind: EvaluationFailureKind, message: String| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(kind, EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        if input.faces.is_empty() {
            return Err(failure(
                EvaluationFailureKind::InvalidResult,
                "unwrap: expected at least one selected face".into(),
            ));
        }
        let live_faces = body
            .face_reference_keys(self.history, self.document)
            .map_err(|msg| failure(EvaluationFailureKind::InvalidReference, msg))?;
        for face in input.faces {
            if !live_faces
                .iter()
                .any(|live| live.as_slice() == face.as_slice())
            {
                return Err(failure(
                    EvaluationFailureKind::InvalidReference,
                    "unwrap face: expected a live named face on the target solid".into(),
                ));
            }
        }
        if let Some(vertex) = input.alignment_vertex {
            let live_subentities = body
                .subentity_reference_keys(self.history, self.document)
                .map_err(|msg| failure(EvaluationFailureKind::InvalidReference, msg))?;
            if !live_subentities
                .iter()
                .any(|live| live.as_slice() == vertex)
            {
                return Err(failure(
                    EvaluationFailureKind::InvalidReference,
                    "unwrap alignment vertex: expected a live vertex on the target solid".into(),
                ));
            }
        }
        let native = body
            .native_shape(self.history, self.document)
            .map_err(|msg| failure(EvaluationFailureKind::InvalidResult, msg))?;
        let (width, height, anchor) =
            unfolded_extents(native, input.faces.len(), input.auto_face_chain)
                .map_err(|msg| failure(EvaluationFailureKind::InvalidResult, msg))?;
        let shell = build_oriented_planar_shell(anchor, width, height, input.alignment)
            .map_err(|msg| failure(EvaluationFailureKind::InvalidResult, msg))?;
        let topology = BlendNamedTopology::from_unwrap_shell(
            self.history,
            self.document,
            feature,
            shell,
            None,
        )
        .map_err(|msg| failure(EvaluationFailureKind::InvalidReference, msg))?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
}

fn unfolded_extents(
    source: &Shape,
    face_count: usize,
    auto_face_chain: bool,
) -> Result<(f64, f64, [f64; 3]), String> {
    let mut bbox = BoundingBox3::default();
    BRepBndLib::add_optimal(source, &mut bbox, false, false).map_err(|cause| cause.to_string())?;
    let [xmin, ymin, zmin, xmax, ymax, zmax] = bbox.get().map_err(|cause| cause.to_string())?;
    let dx = (xmax - xmin).abs().max(0.5);
    let dy = (ymax - ymin).abs().max(0.5);
    let dz = (zmax - zmin).abs().max(0.5);
    let scale = if auto_face_chain && face_count > 1 {
        1.5
    } else {
        1.0
    };
    let width = (dx.max(dy) * std::f64::consts::FRAC_PI_2 * scale).max(1.0);
    let height = dz.max(1.0);
    let anchor = [xmax + 0.5, ymin, zmin];
    Ok((width, height, anchor))
}

fn build_oriented_planar_shell(
    anchor: [f64; 3],
    width: f64,
    height: f64,
    alignment: UnwrapAlignment,
) -> Result<Shape, String> {
    let (u, v) = match alignment {
        UnwrapAlignment::OriginXY => ([1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
        UnwrapAlignment::OriginXZ => ([1.0, 0.0, 0.0], [0.0, 0.0, 1.0]),
        UnwrapAlignment::OriginYZ => ([0.0, 1.0, 0.0], [0.0, 0.0, 1.0]),
        UnwrapAlignment::FlippedXY => ([0.0, 1.0, 0.0], [-1.0, 0.0, 0.0]),
    };
    let pt = |du: f64, dv: f64| {
        Point3::new(
            anchor[0] + u[0] * du + v[0] * dv,
            anchor[1] + u[1] * du + v[1] * dv,
            anchor[2] + u[2] * du + v[2] * dv,
        )
    };
    let mut poly = BRepBuilderAPIMakePolygon::new();
    poly.add_point(pt(0.0, 0.0)).map_err(|e| e.to_string())?;
    poly.add_point(pt(width, 0.0)).map_err(|e| e.to_string())?;
    poly.add_point(pt(width, height))
        .map_err(|e| e.to_string())?;
    poly.add_point(pt(0.0, height)).map_err(|e| e.to_string())?;
    poly.close().map_err(|e| e.to_string())?;
    let wire = poly.wire().map_err(|e| e.to_string())?;
    let face = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|e| e.to_string())?
        .face()
        .map_err(|e| e.to_string())?;
    let mut shell = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut shell);
    let mut shape = shell.into_shape();
    builder
        .add(&mut shape, &face.into_shape())
        .map_err(|e| e.to_string())?;
    Ok(shape)
}
