// SPDX-License-Identifier: GPL-2.0-only
//! Detached cylindrical cut and exact Boolean provenance for Hole (#4093).
use super::*;
use martensite_tkmath::gp::{Axis2, Direction3, Point3};
use martensite_tkprim::brep_prim_api::{BRepPrimAPIMakeCylinder, BRepPrimAPIMakeOneAxis};
use oblikovati_model::identity::DerivedReferenceName;

pub(in crate::prism::named) struct HoleRequest<'a> {
    pub feature: FeatureId,
    pub face: &'a [u8],
    pub axis: [[f64; 3]; 2],
    pub diameter: f64,
    pub depth: f64,
}

impl NamedPrismSnapshot {
    pub(in crate::prism::named) fn hole_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        request: &HoleRequest<'_>,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_hole_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            request,
        )
    }
}

pub(super) fn build_hole_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    request: &HoleRequest<'_>,
) -> Result<BlendCandidate, String> {
    let HoleRequest {
        feature,
        face: face_key,
        axis,
        diameter,
        depth,
    } = *request;
    if !diameter.is_finite() || diameter <= 0. || !depth.is_finite() || depth <= 0. {
        return Err(format!(
            "hole diameter {diameter}, depth {depth}: expected finite positive lengths"
        ));
    }
    let mut maker = cylinder(axis, diameter, depth)?;
    let tool = maker.shape().map_err(|error| error.to_string())?.clone();
    let tool_faces = tool_faces(&mut maker, document, feature, face_key)?;
    super::solid_boolean::build_boolean_candidate(
        input,
        history,
        document,
        super::solid_boolean::BooleanTool {
            shape: &tool,
            sources: tool_faces,
            resolution: None,
        },
        oblikovati_model::feature::sketch_prism::PrismBoolean::Cut,
        "hole",
    )
}

fn cylinder(
    axis: [[f64; 3]; 2],
    diameter: f64,
    depth: f64,
) -> Result<BRepPrimAPIMakeCylinder, String> {
    let [origin, vector] = axis;
    let direction =
        Direction3::new(vector[0], vector[1], vector[2]).map_err(|error| error.to_string())?;
    let frame = Axis2::new(Point3::new(origin[0], origin[1], origin[2]), direction)
        .map_err(|error| error.to_string())?;
    Ok(BRepPrimAPIMakeCylinder::with_axes(
        frame,
        diameter / 2.,
        depth,
    ))
}

fn tool_faces(
    maker: &mut BRepPrimAPIMakeCylinder,
    document: DocumentGuid,
    feature: FeatureId,
    parent: &[u8],
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let axis = maker.one_axis();
    let faces = [
        ("hole-wall", axis.lateral_face()),
        ("hole-end", axis.top_face()),
        ("hole-start", axis.bottom_face()),
    ];
    faces
        .into_iter()
        .map(|(role, face)| {
            let name = DerivedReferenceName::new(document, feature.0, 1, role, &[parent.to_vec()])?
                .encode()?;
            Ok((
                name,
                face.map_err(|error| format!("hole {role}: expected a tool face: {error}"))?
                    .into(),
            ))
        })
        .collect()
}
