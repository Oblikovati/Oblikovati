// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_geometry::WorkPlaneDisplaySettings;
use oblikovati_model::feature::{work_geometry::DatumLength, work_plane::WorkPlaneReference};
use oblikovati_model::quantity::Dimension;
mod display;
mod request;
pub(super) use request::placed_plane;

struct PlaneRevision {
    owner: FeatureId,
    index: i64,
    geometry: Option<PlaneGeometryRevision>,
    display: WorkPlaneDisplaySettings,
}
enum PlaneGeometryRevision {
    Offset(OffsetRevision),
    Definition(WorkGeometryDefinition),
}
struct OffsetRevision {
    base: WorkPlaneReference,
    distance: DatumLength,
    expression: Option<String>,
}

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: RedefineWorkPlaneArgs,
) -> Result<Vec<u8>, Error> {
    let (document, snapshot) = datum_snapshot(files)?;
    let revision = request::prepare(snapshot, args)?;
    if revision_unchanged(snapshot, &revision)? {
        return current_reply(snapshot, revision.owner, revision.index);
    }
    files
        .commands(document)
        .map_err(api_error)?
        .run("Redefine WorkPlane", |snapshot, ids| {
            install(snapshot, ids, revision)
        })
        .map_err(api_error)
}
fn revision_unchanged(
    snapshot: &DocumentSnapshot,
    revision: &PlaneRevision,
) -> Result<bool, Error> {
    let feature = snapshot
        .work_geometry
        .get(revision.owner)
        .map_err(api_error)?;
    let definition = feature.definition();
    let geometry_unchanged = revision.geometry.as_ref().is_none_or(|geometry| match geometry {
        PlaneGeometryRevision::Offset(geometry) => geometry.expression.is_none()
            && matches!(definition, WorkGeometryDefinition::Offset { base, .. } if *base == geometry.base),
        PlaneGeometryRevision::Definition(next) => definition.same_recipe(*next),
    });
    Ok(geometry_unchanged
        && feature
            .plane_display_settings()
            .same_recipe(&revision.display))
}

fn install(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    revision: PlaneRevision,
) -> Result<Vec<u8>, Error> {
    if let Some(geometry) = &revision.geometry {
        install_geometry(snapshot, ids, revision.owner, geometry)?;
    }
    snapshot
        .work_geometry
        .set_plane_display(revision.owner, revision.display)
        .map_err(api_error)?;
    current_reply(snapshot, revision.owner, revision.index)
}
fn install_offset(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    revision: &OffsetRevision,
) -> Result<(), Error> {
    let (distance, added) = revise_distance(snapshot, ids, owner, revision)?;
    let definition = WorkGeometryDefinition::Offset {
        base: revision.base,
        distance,
    };
    snapshot
        .work_geometry
        .replace(owner, definition, definition.reads())
        .map_err(api_error)?;
    if let Some(parameter) = added {
        assign_offset_owner(snapshot, owner, parameter)?;
    }
    Ok(())
}
fn assign_offset_owner(
    snapshot: &mut DocumentSnapshot,
    owner: FeatureId,
    parameter: u64,
) -> Result<(), Error> {
    snapshot
        .work_geometry
        .assign_scalar_owner(owner, "offset", parameter)
        .map(|_| ())
        .map_err(api_error)
}

fn revise_distance(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    revision: &OffsetRevision,
) -> Result<(DatumLength, Option<u64>), Error> {
    let Some(source) = &revision.expression else {
        return Ok((revision.distance, None));
    };
    if let Some(parameter) = owned_offset(snapshot, owner)? {
        snapshot
            .parameters
            .set_scalar_expression(parameter, source, Dimension::Length)
            .map_err(api_error)?;
        return Ok((DatumLength::Parameter(parameter), None));
    }
    // Legacy unowned drivers remain untouched when a feature first acquires an authored scalar (#3745).
    let parameter = scalars::insert_offset(snapshot, ids, owner, source)?;
    Ok((DatumLength::Parameter(parameter), Some(parameter)))
}

fn owned_offset(snapshot: &DocumentSnapshot, owner: FeatureId) -> Result<Option<u64>, Error> {
    Ok(snapshot
        .work_geometry
        .get(owner)
        .map_err(api_error)?
        .owned_scalars()
        .iter()
        .find(|binding| binding.role == "offset")
        .map(|binding| binding.parameter))
}

fn current_reply(
    snapshot: &DocumentSnapshot,
    owner: FeatureId,
    index: i64,
) -> Result<Vec<u8>, Error> {
    let plane = planes::current_info(snapshot, owner, index)?;
    encode(&RedefineWorkPlaneResult { plane })
}

fn install_geometry(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    revision: &PlaneGeometryRevision,
) -> Result<(), Error> {
    match revision {
        PlaneGeometryRevision::Offset(offset) => install_offset(snapshot, ids, owner, offset),
        PlaneGeometryRevision::Definition(definition) => snapshot
            .work_geometry
            .replace(owner, *definition, definition.reads())
            .map(|_| ())
            .map_err(api_error),
    }
}
