// SPDX-License-Identifier: GPL-2.0-only
use super::{
    result_names::require_unique,
    section_body::{bind_section_body, incident_names},
    *,
};
use crate::sweep::{SweepCandidate, SweepFaceRole};
use oblikovati_model::identity::SemanticReferenceName;

mod surface;

impl BlendNamedTopology {
    pub(in crate::prism::named) fn sweep_boolean_tool(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<super::super::extrusion_boolean::BooleanTool, String> {
        let source = self.source_topology(history, document)?;
        Ok(super::super::extrusion_boolean::BooleanTool {
            shape: source.body,
            sources: source.entities,
            resolution: self.resolution,
        })
    }

    pub(in crate::prism::named) fn from_sweep(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        sources: [u64; 2],
        candidate: SweepCandidate,
    ) -> Result<Self, String> {
        let achieved_tolerance = Some(candidate.achieved_tolerance());
        let body = SemanticReferenceName::new(document, feature.0, 4, "sweep", "body", &sources)?
            .encode()?;
        let mut faces = Vec::with_capacity(candidate.faces.len());
        for (role, shape) in candidate.faces {
            let name = match role {
                SweepFaceRole::Start => {
                    SemanticReferenceName::new(document, feature.0, 1, "sweep", "start", &sources)
                }
                SweepFaceRole::End => {
                    SemanticReferenceName::new(document, feature.0, 1, "sweep", "end", &sources)
                }
                SweepFaceRole::Side { profile, path } => SemanticReferenceName::new(
                    document,
                    feature.0,
                    1,
                    "sweep",
                    "side",
                    &[sources[0], sources[1], profile.0, path.0],
                ),
            }?;
            faces.push((name.encode()?, shape));
        }
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
