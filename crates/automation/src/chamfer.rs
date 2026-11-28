// SPDX-License-Identifier: GPL-2.0-only
//! Typed, certified equal-distance chamfer creation (#4046).
use crate::FilletAccess;
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Chamfer},
};
use oblikovati_model::{
    feature::{FeatureId, chamfer::ChamferDefinition},
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn FilletAccess>,
    mut replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input: Chamfer = serde_json::from_value(
        request.args.0.unwrap_or_else(|| serde_json::json!({})),
    )
    .map_err(|cause| {
        Error(format!(
            "chamfer arguments: expected the canonical Chamfer object: {cause}"
        ))
    })?;
    validate_options(&input)?;
    let edges = parse_edges(&input)?;
    let document = crate::part_owner::active_part(files)?;
    let backend = backend
        .ok_or_else(|| Error("chamfer: expected an injected certified geometry service".into()))?;
    let body = match &mut replay {
        Some(owner) => {
            crate::fillet::blend_input_body(files, backend, Some(&mut **owner), document, &edges)?
        }
        None => crate::fillet::blend_input_body(files, backend, None, document, &edges)?,
    };
    let workspace = files.workspace();
    let session = workspace.session(document).map_err(api_error)?;
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let prepared = prepare_chamfer(session.snapshot(), body, edges, &input.distance)?;
    let prefix = crate::fillet::matching_blend_prefix(files, backend, replay.as_deref(), document)?;
    let ((reply, candidate), transition) =
        create(files, document, guid, backend, prepared, prefix)?;
    crate::fillet::publish_blend_candidate(files, replay, document, candidate, &transition)?;
    Ok(reply)
}

fn validate_options(input: &Chamfer) -> Result<(), Error> {
    let mode = &input.chamfer_type;
    let strategy = &input.concave_strategy;
    if (!mode.is_empty() && !mode.eq_ignore_ascii_case("distance"))
        || (!strategy.is_empty() && !strategy.eq_ignore_ascii_case("outward"))
        || !input.distance2.is_empty()
        || !input.angle.is_empty()
        || !input.reference_face.is_empty()
        || !input.partial_start.is_empty()
        || !input.partial_length.is_empty()
        || input
            .edges_geom
            .as_ref()
            .is_some_and(|edges| !edges.is_empty())
    {
        return Err(Error(format!(
            "chamfer options {input:?}: expected equal distance on whole named edges"
        )));
    }
    Ok(())
}

fn parse_edges(input: &Chamfer) -> Result<Vec<Vec<u8>>, Error> {
    let edges = input
        .edge_refs
        .as_ref()
        .filter(|edges| !edges.is_empty())
        .ok_or_else(|| {
            Error(format!(
                "chamfer edgeRefs {:?}: expected one or more exact edge keys",
                input.edge_refs
            ))
        })?;
    edges
        .iter()
        .map(|key| crate::reference_bytes::hex_bytes(key))
        .collect()
}

/// A validated draft that can be applied to a private document candidate.
#[derive(Clone)]
pub struct PreparedChamfer {
    name: String,
    definition: ChamferDefinition,
}
impl PreparedChamfer {
    /// Construct from pre-validated name and definition.
    pub(crate) fn from_parts(name: String, definition: ChamferDefinition) -> Self {
        Self { name, definition }
    }
    /// Append the draft to a candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        candidate: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        candidate
            .create_chamfer(&self.name, self.definition, ids)
            .map_err(api_error)
    }
}

/// Prepare a named edge set and length expression; example: `prepare_chamfer(snapshot, body, edges, "1 mm")?`.
pub fn prepare_chamfer(
    snapshot: &DocumentSnapshot,
    body: Vec<u8>,
    edges: Vec<Vec<u8>>,
    distance: &str,
) -> Result<PreparedChamfer, Error> {
    let distance = crate::extrusion::positive_length(snapshot, distance, "chamfer distance")?;
    let definition = ChamferDefinition::new(body, edges, distance).map_err(Error)?;
    Ok(PreparedChamfer {
        name: crate::fillet::next_name(snapshot, "Chamfer")?,
        definition,
    })
}

fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    guid: oblikovati_model::identity::DocumentGuid,
    backend: &dyn FilletAccess,
    prepared: PreparedChamfer,
    prefix: Option<crate::CertifiedPartPrefix>,
) -> Result<crate::fillet::BlendTransaction, Error> {
    let name = prepared.name.clone();
    files.commands(document).map_err(api_error)?.run_private_candidate(
        "Create chamfer",
        |candidate, ids| prepared.create(candidate, ids),
        |candidate, id, history| {
            let (bodies, captured) = if let Some(prefix) = prefix {
                backend.capture_with_prefix(oblikovati_model::feature::part::PartBlendKind::Chamfer, history, guid, candidate, id, &prefix).map_err(Error)?
            } else { (backend.validate_chamfer(guid, candidate, id).map_err(Error)?, None) };
            let feature = candidate.work_geometry.get(id).map_err(api_error)?;
            let reply = crate::encode(&serde_json::json!({"id": id.0, "name": feature.name(), "kind": feature.definition().kind(), "suppressed": candidate.feature_suppressed(id).map_err(api_error)?, "health": "", "feature": name, "bodies": bodies, "healthy": true}))?;
            Ok((reply, captured))
        },
    ).map_err(api_error)
}
fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}

#[cfg(test)]
#[path = "../tests/chamfer/mod.rs"]
mod tests;
