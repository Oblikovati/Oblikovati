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
mod planes;
mod redefine;
mod reference;
mod rows;
mod scalars;

/// Methods use the existing API contract and the one file-owned datum program.
pub const DATUM_COMMAND_METHODS: [&str; 4] = [
    METHOD_WORK_POINTS_CREATE,
    METHOD_WORK_AXES_CREATE,
    METHOD_WORK_PLANES_CREATE,
    METHOD_WORK_PLANES_REDEFINE,
];
pub const DATUM_QUERY_METHODS: [&str; 3] = [
    METHOD_WORK_POINTS_LIST,
    METHOD_WORK_AXES_LIST,
    METHOD_WORK_PLANES_LIST,
];
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum DatumCollection {
    Axes,
    Points,
    Planes,
}
#[derive(Debug)]
pub(super) struct PreparedDatum {
    definition: PreparedGeometry,
    construction: bool,
    visible: bool,
    collection: DatumCollection,
}
#[derive(Debug)]
pub(super) enum PreparedGeometry {
    Definition(WorkGeometryDefinition),
    Offset {
        base: oblikovati_model::feature::work_plane::WorkPlaneReference,
        expression: String,
    },
}
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    if method == METHOD_WORK_PLANES_REDEFINE {
        return redefine::apply(files, decode(method, request)?);
    }
    create(files, method, request)
}
fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let (document, snapshot) = datum_snapshot(files)?;
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
    let feature =
        scalars::authored_feature(snapshot, ids, id, prepared.collection, prepared.definition)?;
    snapshot.work_geometry.append(feature).map_err(api_error)?;
    assign_metadata(snapshot, id, prepared.construction, prepared.visible)?;
    rows::creation_reply(snapshot, id, prepared.collection)
}
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let (_, snapshot) = datum_snapshot(files)?;
    match method {
        METHOD_WORK_AXES_LIST => rows::axes(snapshot, query_args(method, request)?),
        METHOD_WORK_POINTS_LIST => rows::points(snapshot, query_args(method, request)?),
        METHOD_WORK_PLANES_LIST => planes::list(snapshot, query_args(method, request)?),
        _ => Err(Error(format!(
            "datum query {method:?}: expected {DATUM_QUERY_METHODS:?}"
        ))),
    }
}
impl DatumCollection {
    fn contains(self, definition: WorkGeometryDefinition) -> bool {
        let kind = match definition {
            WorkGeometryDefinition::Axis(_) | WorkGeometryDefinition::AxisThroughPoints(_) => {
                Self::Axes
            }
            WorkGeometryDefinition::Point(_) | WorkGeometryDefinition::PointAtReference(_) => {
                Self::Points
            }
            WorkGeometryDefinition::Fixed(_) | WorkGeometryDefinition::Offset { .. } => {
                Self::Planes
            }
        };
        self == kind
    }
    fn origin_count(self) -> usize {
        match self {
            Self::Axes | Self::Planes => 3,
            Self::Points => 1,
        }
    }
    fn label(self) -> &'static str {
        match self {
            Self::Axes => "WorkAxis",
            Self::Points => "WorkPoint",
            Self::Planes => "WorkPlane",
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

fn datum_snapshot<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<(u64, &DocumentSnapshot), Error> {
    let document = active_part(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    Ok((document, snapshot))
}

fn assign_metadata(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    construction: bool,
    visible: bool,
) -> Result<(), Error> {
    snapshot
        .work_geometry
        .set_work_plane_construction(id, construction)
        .map_err(api_error)?;
    snapshot
        .work_geometry
        .set_datum_visible(id, visible)
        .map_err(api_error)
}
