// SPDX-License-Identifier: GPL-2.0-only
//! Name derived-part result topology from exact face, edge, and vertex incidence.
use super::{result_names::require_unique, section_body::incident_names, *};
use oblikovati_model::identity::SemanticReferenceName;

impl BlendNamedTopology {
    pub(crate) fn from_derived(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        let body =
            SemanticReferenceName::new(document, feature.0, 4, "derived-part", "body", &[1])?
                .encode()?;
        let source_faces = shapes(&shape, ShapeEnum::Face)?;
        let mut faces = Vec::with_capacity(source_faces.len());
        for (index, face) in source_faces.into_iter().enumerate() {
            let name = SemanticReferenceName::new(
                document,
                feature.0,
                1,
                "derived-part",
                "face",
                &[(index + 1) as u64],
            )?
            .encode()?;
            faces.push((name, face));
        }
        require_unique(&faces)?;
        let edges = incident_names(document, feature, &shape, ShapeEnum::Edge, 2, &faces)?;
        let vertices = incident_names(document, feature, &shape, ShapeEnum::Vertex, 3, &edges)?;
        let entries: Vec<_> = std::iter::once((body.clone(), shape.clone()))
            .chain(faces)
            .chain(edges)
            .chain(vertices)
            .collect();
        let bindings = ExactReferenceBindings::from_entries(
            document,
            history.evaluation_scope(),
            entries.clone(),
        )?;
        verify_round_trip(&bindings, history, document, &entries)?;
        let live = bindings
            .encoded_keys(history, document)?
            .map(<[u8]>::to_vec)
            .collect();
        Ok(Self {
            resolution,
            achieved_tolerance: None,
            live,
            deleted: Vec::new(),
            splits: BTreeMap::new(),
            bindings,
            active_body: body.clone(),
            primary_body: body,
            shape,
        })
    }

    pub(crate) fn from_unwrap_shell(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        let body =
            SemanticReferenceName::new(document, feature.0, 4, "unwrap", "body", &[1])?.encode()?;
        let faces = shapes(&shape, ShapeEnum::Face)?
            .into_iter()
            .enumerate()
            .map(|(index, face)| {
                let name = SemanticReferenceName::new(
                    document,
                    feature.0,
                    1,
                    "unwrap",
                    "face",
                    &[(index + 1) as u64],
                )?
                .encode()?;
                Ok((name, face))
            })
            .collect::<Result<Vec<_>, String>>()?;
        let edges = shapes(&shape, ShapeEnum::Edge)?
            .into_iter()
            .enumerate()
            .map(|(index, edge)| {
                let name = SemanticReferenceName::new(
                    document,
                    feature.0,
                    2,
                    "unwrap",
                    "edge",
                    &[(index + 1) as u64],
                )?
                .encode()?;
                Ok((name, edge))
            })
            .collect::<Result<Vec<_>, String>>()?;
        let vertices = incident_names(document, feature, &shape, ShapeEnum::Vertex, 3, &edges)?;
        let entries: Vec<_> = std::iter::once((body.clone(), shape.clone()))
            .chain(faces)
            .chain(edges)
            .chain(vertices)
            .collect();
        let bindings = ExactReferenceBindings::from_entries(
            document,
            history.evaluation_scope(),
            entries.clone(),
        )?;
        verify_round_trip(&bindings, history, document, &entries)?;
        let live = bindings
            .encoded_keys(history, document)?
            .map(<[u8]>::to_vec)
            .collect();
        Ok(Self {
            resolution,
            achieved_tolerance: None,
            live,
            deleted: Vec::new(),
            splits: BTreeMap::new(),
            bindings,
            active_body: body.clone(),
            primary_body: body,
            shape,
        })
    }
}
