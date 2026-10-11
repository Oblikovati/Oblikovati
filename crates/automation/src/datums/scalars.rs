// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::quantity::Dimension;

mod kind;
pub(in crate::datums) use kind::OwnedDatumScalar;

pub(super) fn validate_offset(snapshot: &DocumentSnapshot, source: &str) -> Result<(), Error> {
    OwnedDatumScalar::Offset.validate(snapshot, source)
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
        PreparedGeometry::LineAngle {
            line,
            base,
            expression,
        } => angle_feature(snapshot, ids, owner, line, base, &expression),
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
    let parameter = OwnedDatumScalar::Offset.insert(snapshot, ids, owner, source)?;
    let definition = WorkGeometryDefinition::Offset {
        base,
        distance: oblikovati_model::feature::work_geometry::DatumLength::Parameter(parameter),
    };
    AuthoredFeature::datum(owner, DatumCollection::Planes.label(), definition)
        .with_owned_scalar(OwnedDatumScalar::Offset.role(), parameter)
        .map_err(api_error)
}

fn angle_feature(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    line: oblikovati_model::feature::work_axis::WorkAxisReference,
    base: oblikovati_model::feature::work_plane::WorkPlaneReference,
    expression: &str,
) -> Result<AuthoredFeature<WorkGeometryDefinition>, Error> {
    use oblikovati_model::feature::work_geometry::{DatumAngle, WorkPlaneLineAngle};
    let parameter = OwnedDatumScalar::Angle.insert(snapshot, ids, owner, expression)?;
    let definition = WorkGeometryDefinition::AtAngleAboutAxis(WorkPlaneLineAngle {
        line,
        base,
        angle: DatumAngle::parameter(parameter),
    });
    AuthoredFeature::datum(owner, DatumCollection::Planes.label(), definition)
        .with_owned_scalar(OwnedDatumScalar::Angle.role(), parameter)
        .map_err(api_error)
}
