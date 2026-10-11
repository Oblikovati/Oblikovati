// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::quantity::Dimension;

pub(super) fn validate_offset(snapshot: &DocumentSnapshot, source: &str) -> Result<(), Error> {
    snapshot
        .parameters
        .evaluate_scalar_expression(source, Dimension::Length)
        .map(|_| ())
        .map_err(api_error)
}
pub(super) fn insert_offset(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    source: &str,
) -> Result<u64, Error> {
    let id = ids.allocate().map_err(api_error)?;
    let name = format!("feature{}_offset", owner.0);
    snapshot
        .parameters
        .insert_scalar_expression(id, &name, source, Dimension::Length)
        .map_err(api_error)?;
    snapshot.parameters.model_value(id).map_err(api_error)?;
    Ok(id)
}
pub(super) fn authored_feature(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    collection: DatumCollection,
    source: PreparedGeometry,
) -> Result<AuthoredFeature<WorkGeometryDefinition>, Error> {
    match source {
        PreparedGeometry::Definition(definition) => Ok(AuthoredFeature::datum(
            owner,
            collection.label(),
            definition,
        )),
        PreparedGeometry::Offset { base, expression } => {
            offset_feature(snapshot, ids, owner, base, &expression)
        }
    }
}
fn offset_feature(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    base: oblikovati_model::feature::work_plane::WorkPlaneReference,
    source: &str,
) -> Result<AuthoredFeature<WorkGeometryDefinition>, Error> {
    let parameter = insert_offset(snapshot, ids, owner, source)?;
    let definition = WorkGeometryDefinition::Offset {
        base,
        distance: oblikovati_model::feature::work_geometry::DatumLength::Parameter(parameter),
    };
    AuthoredFeature::datum(owner, DatumCollection::Planes.label(), definition)
        .with_owned_scalar("offset", parameter)
        .map_err(api_error)
}
