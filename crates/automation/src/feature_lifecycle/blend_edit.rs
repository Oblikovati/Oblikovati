// SPDX-License-Identifier: GPL-2.0-only
//! Validate edited blend features before adopting their recipes (#4045, #4046).
use super::*;
use oblikovati_model::feature::{chamfer::ChamferDefinition, fillet::FilletDefinition};

#[derive(Clone, Copy, PartialEq, Eq)]
pub(super) enum BlendEditKind {
    Fillet,
    Chamfer,
}

pub(super) fn kind<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    id: u64,
) -> Result<Option<BlendEditKind>, Error> {
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let (feature, _) = locate(&snapshot.work_geometry, id)?;
    Ok(match feature.definition() {
        PartFeatureDefinition::Fillet(_) => Some(BlendEditKind::Fillet),
        PartFeatureDefinition::Chamfer(_) => Some(BlendEditKind::Chamfer),
        _ => None,
    })
}

pub(super) fn edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    backend: Option<&dyn crate::FilletAccess>,
    request: &EditFeatureArgs,
    kind: BlendEditKind,
) -> Result<u64, Error> {
    let backend =
        backend.ok_or_else(|| Error("blend edit: expected a certified geometry service".into()))?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let label = match kind {
        BlendEditKind::Fillet => "Edit fillet",
        BlendEditKind::Chamfer => "Edit chamfer",
    };
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            label,
            |snapshot, _| apply_blend_edit(snapshot, request, kind),
            |snapshot, id| {
                match kind {
                    BlendEditKind::Fillet => backend.validate(guid, snapshot, id),
                    BlendEditKind::Chamfer => backend.validate_chamfer(guid, snapshot, id),
                }
                .map_err(Error)?;
                Ok(id.0)
            },
        )
        .map_err(api_error)
}

/// Update a private Fillet candidate; example: `apply_fillet_radius_edit(candidate, &request)?`.
pub fn apply_fillet_radius_edit(
    snapshot: &mut DocumentSnapshot,
    request: &EditFeatureArgs,
) -> Result<FeatureId, Error> {
    apply_blend_edit(snapshot, request, BlendEditKind::Fillet)
}
/// Update a private Chamfer candidate; example: `apply_chamfer_distance_edit(candidate, &request)?`.
pub fn apply_chamfer_distance_edit(
    snapshot: &mut DocumentSnapshot,
    request: &EditFeatureArgs,
) -> Result<FeatureId, Error> {
    apply_blend_edit(snapshot, request, BlendEditKind::Chamfer)
}
fn apply_blend_edit(
    snapshot: &mut DocumentSnapshot,
    request: &EditFeatureArgs,
    kind: BlendEditKind,
) -> Result<FeatureId, Error> {
    require_empty_repicks(request)?;
    let value = single_distance_edit(request)?;
    let distance = crate::extrusion::positive_length(snapshot, value, "blend distance")?;
    let (feature, _) = locate(&snapshot.work_geometry, request.id)?;
    let definition = revised_blend(feature.definition(), request.id, distance, kind)?;
    let reads = definition.reads();
    let id = FeatureId(request.id);
    snapshot
        .work_geometry
        .replace(id, definition, reads)
        .map_err(api_error)?;
    Ok(id)
}
fn revised_blend(
    previous: &PartFeatureDefinition,
    id: u64,
    distance: oblikovati_model::feature::work_geometry::DatumLength,
    kind: BlendEditKind,
) -> Result<PartFeatureDefinition, Error> {
    match (previous, kind) {
        (PartFeatureDefinition::Fillet(old), BlendEditKind::Fillet) => {
            FilletDefinition::new(old.body().to_vec(), old.edges().to_vec(), distance)
                .map(|value| PartFeatureDefinition::Fillet(Box::new(value)))
                .map_err(api_error)
        }
        (PartFeatureDefinition::Chamfer(old), BlendEditKind::Chamfer) => {
            ChamferDefinition::new(old.body().to_vec(), old.edges().to_vec(), distance)
                .map(|value| PartFeatureDefinition::Chamfer(Box::new(value)))
                .map_err(api_error)
        }
        _ => Err(Error(format!(
            "feature {id}, kind {}: expected the selected blend kind",
            previous.kind()
        ))),
    }
}
