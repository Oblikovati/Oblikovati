// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode, part_owner::active_part};
use oblikovati_api::{Error, types::*, wire::*};
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureEvaluationStatus, FeatureId,
        work_geometry::{WorkGeometryDefinition, WorkGeometryError, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
mod health;
mod input;
mod origins;
mod reference;
mod rows;

/// Methods use the existing API contract and the one file-owned datum program.
pub const DATUM_COMMAND_METHODS: [&str; 2] = [METHOD_WORK_POINTS_CREATE, METHOD_WORK_AXES_CREATE];
pub const DATUM_QUERY_METHODS: [&str; 2] = [METHOD_WORK_POINTS_LIST, METHOD_WORK_AXES_LIST];
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum DatumCollection {
    Axes,
    Points,
}
#[derive(Debug)]
pub(super) struct PreparedDatum {
    definition: WorkGeometryDefinition,
    construction: bool,
    collection: DatumCollection,
}
pub(super) fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let prepared = input::definition(method, request, snapshot)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run(prepared.collection.label(), |snapshot, ids| {
            insert(snapshot, ids, prepared)
        })
        .map_err(api_error)
}
fn insert(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    prepared: PreparedDatum,
) -> Result<Vec<u8>, Error> {
    let id = ids.allocate_feature().map_err(api_error)?;
    let feature = AuthoredFeature::datum(id, prepared.collection.label(), prepared.definition);
    snapshot.work_geometry.append(feature).map_err(api_error)?;
    snapshot
        .work_geometry
        .set_work_plane_construction(id, prepared.construction)
        .map_err(api_error)?;
    rows::creation_reply(snapshot, id, prepared.collection)
}
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    match method {
        METHOD_WORK_AXES_LIST => rows::axes(snapshot, query_args(method, request)?),
        METHOD_WORK_POINTS_LIST => rows::points(snapshot, query_args(method, request)?),
        _ => Err(Error(format!(
            "datum query {method:?}: expected {DATUM_QUERY_METHODS:?}"
        ))),
    }
}
impl DatumCollection {
    fn contains(self, definition: WorkGeometryDefinition) -> bool {
        match self {
            Self::Axes => matches!(
                definition,
                WorkGeometryDefinition::Axis(_) | WorkGeometryDefinition::AxisThroughPoints(_)
            ),
            Self::Points => matches!(
                definition,
                WorkGeometryDefinition::Point(_) | WorkGeometryDefinition::PointAtReference(_)
            ),
        }
    }
    fn origin_count(self) -> usize {
        match self {
            Self::Axes => 3,
            Self::Points => 1,
        }
    }
    fn label(self) -> &'static str {
        match self {
            Self::Axes => "WorkAxis",
            Self::Points => "WorkPoint",
        }
    }
}
fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}

#[cfg(test)]
mod tests;

fn query_args<T: serde::de::DeserializeOwned + Default>(
    method: &str,
    request: &[u8],
) -> Result<T, Error> {
    if request.is_empty() {
        return Ok(T::default());
    }
    decode::<Option<T>>(method, request).map(Option::unwrap_or_default)
}
