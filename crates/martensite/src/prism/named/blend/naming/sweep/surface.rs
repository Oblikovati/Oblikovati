// SPDX-License-Identifier: GPL-2.0-only
//! Bind the trims of a single periodic profile independently of shell traversal order.
use super::super::section_body::name_open_shell_edges;
use super::*;
use crate::sweep::SurfaceSweepCandidate;

impl BlendNamedTopology {
    pub(in crate::prism::named) fn from_surface_sweep(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        sketches: [u64; 2],
        candidate: SurfaceSweepCandidate,
    ) -> Result<Self, String> {
        let [(SweepFaceRole::Side { profile, path }, face)] = candidate.faces() else {
            return Err("named surface Sweep requires a single swept face".into());
        };
        let sources = [sketches[0], sketches[1], profile.0, path.0];
        let name = |kind: u32, role: &str| {
            SemanticReferenceName::new(document, feature.0, kind, "surface-sweep", role, &sources)?
                .encode()
        };
        let faces = [(name(1, "side")?, face.clone())];
        let trims = named_trims(candidate.trims(), &name)?;
        let edges = name_open_shell_edges(candidate.shell(), trims, &name)?;
        let vertices = incident_names(
            document,
            feature,
            candidate.shell(),
            ShapeEnum::Vertex,
            3,
            &edges,
        )?;
        let subentities = faces.into_iter().chain(edges).chain(vertices).collect();
        bind_section_body(
            history,
            document,
            name(4, "body")?,
            candidate.shell().clone(),
            candidate.resolution,
            Some(candidate.achieved_tolerance()),
            subentities,
        )
    }
}

fn named_trims(
    [start, end]: &[Shape; 2],
    name: &impl Fn(u32, &str) -> Result<Vec<u8>, String>,
) -> Result<[(Vec<u8>, Shape); 2], String> {
    let edge = |wire| {
        let edges = shapes(wire, ShapeEnum::Edge)?;
        let [edge] = edges
            .try_into()
            .map_err(|_| "surface Sweep trim requires a single edge")?;
        Ok::<Shape, String>(edge)
    };
    Ok([
        (name(2, "start")?, edge(start)?),
        (name(2, "end")?, edge(end)?),
    ])
}
