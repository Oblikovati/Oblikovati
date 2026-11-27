// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::quantity::Dimension;

mod kind;
pub(in crate::datums) use kind::OwnedDatumScalar;

pub(super) fn validate_offset(snapshot: &DocumentSnapshot, source: &str) -> Result<(), Error> {
    OwnedDatumScalar::Offset.validate(snapshot, source)
}
/// The first free number: names stay distinct in the browser and after a datum is deleted.
fn next_datum_name(snapshot: &DocumentSnapshot, collection: DatumCollection) -> String {
    let taken = |name: &str| {
        snapshot
            .work_geometry
            .features()
            .iter()
            .any(|feature| feature.name() == name)
    };
    (1_u32..)
        .map(|number| format!("{}{number}", collection.label()))
        .find(|name| !taken(name))
        .expect("u32 names outnumber any document's datums")
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
            &next_datum_name(snapshot, collection),
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
    let name = next_datum_name(snapshot, DatumCollection::Planes);
    AuthoredFeature::datum(owner, &name, definition)
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
    let name = next_datum_name(snapshot, DatumCollection::Planes);
    AuthoredFeature::datum(owner, &name, definition)
        .with_owned_scalar(OwnedDatumScalar::Angle.role(), parameter)
        .map_err(api_error)
}
