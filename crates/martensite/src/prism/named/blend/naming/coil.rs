// SPDX-License-Identifier: GPL-2.0-only
use super::{
    result_names::require_unique,
    section_body::{bind_section_body, incident_names},
    *,
};
use crate::coil::{CoilCandidate, CoilFaceRole};
use oblikovati_model::identity::SemanticReferenceName;

impl BlendNamedTopology {
    pub(in crate::prism::named) fn from_coil(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        sketch_id: u64,
        candidate: CoilCandidate,
    ) -> Result<Self, String> {
        let achieved_tolerance = Some(candidate.achieved_tolerance());
        let body =
            SemanticReferenceName::new(document, feature.0, 4, "coil", "body", &[sketch_id])?
                .encode()?;
        let faces = name_coil_faces(document, feature, sketch_id, candidate.faces)?;
        require_unique(&faces)?;
        let edges = incident_names(
            document,
            feature,
            &candidate.shape,
            ShapeEnum::Edge,
            2,
            &faces,
        )?;
        let vertices = incident_names(
            document,
            feature,
            &candidate.shape,
            ShapeEnum::Vertex,
            3,
            &edges,
        )?;
        let subentities = faces.into_iter().chain(edges).chain(vertices).collect();
        bind_section_body(
            history,
            document,
            body,
            candidate.shape,
            candidate.resolution,
            achieved_tolerance,
            subentities,
        )
    }
}

fn name_coil_faces(
    document: DocumentGuid,
    feature: FeatureId,
    sketch_id: u64,
    source_faces: Vec<(CoilFaceRole, Shape)>,
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let mut faces = Vec::with_capacity(source_faces.len());
    for (role, shape) in source_faces {
        let name = match role {
            CoilFaceRole::Start => {
                SemanticReferenceName::new(document, feature.0, 1, "coil", "start", &[sketch_id])
            }
            CoilFaceRole::End => {
                SemanticReferenceName::new(document, feature.0, 1, "coil", "end", &[sketch_id])
            }
            CoilFaceRole::Side(curve) => SemanticReferenceName::new(
                document,
                feature.0,
                1,
                "coil",
                "side",
                &[sketch_id, curve],
            ),
        }?;
        faces.push((name.encode()?, shape));
    }
    Ok(faces)
}
