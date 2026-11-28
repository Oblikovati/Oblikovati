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
        Self::from_derived_with_bodies(
            history,
            document,
            feature,
            body.clone(),
            body,
            Vec::new(),
            shape,
            resolution,
        )
    }

    pub(crate) fn from_derived_preserving_body(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        primary_body: Vec<u8>,
        deleted: Vec<Vec<u8>>,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        Self::from_tagged_preserving_body(
            history,
            document,
            feature,
            "thicken-offset",
            primary_body,
            deleted,
            shape,
            resolution,
        )
    }

    #[allow(clippy::too_many_arguments)]
    pub(crate) fn from_tagged_preserving_body(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        tag: &str,
        primary_body: Vec<u8>,
        deleted: Vec<Vec<u8>>,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        let active_body =
            SemanticReferenceName::new(document, feature.0, 4, tag, "body", &[1])?.encode()?;
        Self::from_derived_with_bodies(
            history,
            document,
            feature,
            primary_body,
            active_body,
            deleted,
            shape,
            resolution,
        )
    }

    #[allow(clippy::too_many_arguments)]
    fn from_derived_with_bodies(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        primary_body: Vec<u8>,
        active_body: Vec<u8>,
        deleted: Vec<Vec<u8>>,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
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
        let mut body_entries = vec![(active_body.clone(), shape.clone())];
        if primary_body != active_body {
            body_entries.push((primary_body.clone(), shape.clone()));
        }
        let entries: Vec<_> = body_entries
            .into_iter()
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
            deleted,
            splits: BTreeMap::new(),
            bindings,
            active_body,
            primary_body,
            shape,
            thread_overlays: Vec::new(),
            mark_overlays: Vec::new(),
            finish_face_colors: BTreeMap::new(),
        })
    }

    pub(crate) fn from_unwrap_shell(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        Self::from_tagged_shell(history, document, feature, "unwrap", shape, resolution)
    }

    pub(crate) fn from_offset_shell(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        Self::from_tagged_shell(
            history,
            document,
            feature,
            "thicken-offset",
            shape,
            resolution,
        )
    }

    pub(crate) fn from_extended_shell_preserving_body(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        primary_body: Vec<u8>,
        deleted: Vec<Vec<u8>>,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        let active_body =
            SemanticReferenceName::new(document, feature.0, 4, "extend-surface", "body", &[1])?
                .encode()?;
        Self::from_tagged_shell_with_bodies(
            history,
            document,
            feature,
            "extend-surface",
            primary_body,
            active_body,
            deleted,
            shape,
            resolution,
        )
    }

    pub(crate) fn from_boundary_patch_shell(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        Self::from_tagged_shell(
            history,
            document,
            feature,
            "boundary-patch",
            shape,
            resolution,
        )
    }

    pub(crate) fn from_ruled_surface_shell(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        Self::from_tagged_shell(
            history,
            document,
            feature,
            "ruled-surface",
            shape,
            resolution,
        )
    }

    fn from_tagged_shell(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        tag: &str,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        let body =
            SemanticReferenceName::new(document, feature.0, 4, tag, "body", &[1])?.encode()?;
        Self::from_tagged_shell_with_bodies(
            history,
            document,
            feature,
            tag,
            body.clone(),
            body,
            Vec::new(),
            shape,
            resolution,
        )
    }

    #[allow(clippy::too_many_arguments)]
    fn from_tagged_shell_with_bodies(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        tag: &str,
        primary_body: Vec<u8>,
        active_body: Vec<u8>,
        deleted: Vec<Vec<u8>>,
        shape: Shape,
        resolution: Option<PrismResolution>,
    ) -> Result<Self, String> {
        let faces = shapes(&shape, ShapeEnum::Face)?
            .into_iter()
            .enumerate()
            .map(|(index, face)| {
                let name = SemanticReferenceName::new(
                    document,
                    feature.0,
                    1,
                    tag,
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
                    tag,
                    "edge",
                    &[(index + 1) as u64],
                )?
                .encode()?;
                Ok((name, edge))
            })
            .collect::<Result<Vec<_>, String>>()?;
        let vertices = incident_names(document, feature, &shape, ShapeEnum::Vertex, 3, &edges)?;
        let mut body_entries = vec![(active_body.clone(), shape.clone())];
        if primary_body != active_body {
            body_entries.push((primary_body.clone(), shape.clone()));
        }
        let entries: Vec<_> = body_entries
            .into_iter()
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
            deleted,
            splits: BTreeMap::new(),
            bindings,
            active_body,
            primary_body,
            shape,
            thread_overlays: Vec::new(),
            mark_overlays: Vec::new(),
            finish_face_colors: BTreeMap::new(),
        })
    }
}
