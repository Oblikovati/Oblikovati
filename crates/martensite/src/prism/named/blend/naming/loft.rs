// SPDX-License-Identifier: GPL-2.0-only
use super::{
    section_body::{bind_section_body, incident_names},
    *,
};
use oblikovati_model::{feature::sketch_loft::SketchLoftSection, identity::SemanticReferenceName};

impl BlendNamedTopology {
    pub(in crate::prism::named) fn from_loft(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        [first, last]: [SketchLoftSection; 2],
        candidate: crate::loft::LoftCandidate,
    ) -> Result<Self, String> {
        let first = [first.sketch.0, first.circle.0];
        let last = [last.sketch.0, last.circle.0];
        let sources = [first[0], first[1], last[0], last[1]];
        let name = |kind, role, ids: &[u64]| {
            SemanticReferenceName::new(document, feature.0, kind, "loft", role, ids)?.encode()
        };
        let body = name(4, "body", &sources)?;
        let [start, end, wall] = candidate.faces;
        let faces = vec![
            (name(1, "start", &first)?, start),
            (name(1, "end", &last)?, end),
            (name(1, "side", &sources)?, wall),
        ];
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
        bind_section_body(
            history,
            document,
            body,
            candidate.shape,
            candidate.resolution,
            Some(candidate.achieved),
            faces.into_iter().chain(edges).chain(vertices).collect(),
        )
    }
}
