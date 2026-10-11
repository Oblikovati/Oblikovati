// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{parameter::ParameterCollection, sketch::SketchConstraintId};
fn dimensioned_owner() -> DocumentSketches {
    let mut owner = spline_sketches();
    let mut geometry = owner.get(DocumentSketchId(7)).unwrap().geometry().clone();
    geometry
        .add_offset_spline_dimension(SketchConstraintId(1), SketchEntityId(11), 1)
        .unwrap();
    owner
        .replace_geometry(DocumentSketchId(7), geometry)
        .unwrap();
    owner
}
fn packaged_offset_dimension(kind: DocumentType) -> DocumentPackage {
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "offset", USER_PARAM, "5 mm")
        .unwrap();
    DocumentPackage::empty(kind, "Driven offset")
        .with_sketches(&dimensioned_owner(), 7, 12)
        .unwrap()
        .with_parameters(&parameters, Default::default())
        .unwrap()
}
#[test]
fn native_package_preserves_offset_side_source_and_independent_parent_ids() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_dimension_package(kind);
    }
}
fn assert_dimension_package(kind: DocumentType) {
    let package = packaged_offset_dimension(kind);
    let bytes = package.encode().unwrap();
    let decoded = DocumentPackage::decode(&bytes).unwrap();
    let restored = decoded.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&dimensioned_owner()));
    let parameters = decoded.restore_parameters(Default::default()).unwrap();
    let geometry = restored
        .sketches
        .get(DocumentSketchId(7))
        .unwrap()
        .geometry();
    assert_eq!(geometry.recipe().version, 11);
    assert_saved_dimension_solve(geometry, &parameters.parameters);
    assert_eq!(decoded.encode().unwrap(), bytes);
}
fn assert_saved_dimension_solve(geometry: &Sketch2D, parameters: &ParameterCollection<u64>) {
    assert_eq!(parameters.get(1).unwrap().source, "5 mm");
    let outcome = geometry
        .solve(parameters, Default::default(), || false)
        .unwrap();
    let offset = outcome
        .candidate()
        .offset_spline(SketchEntityId(11))
        .unwrap();
    assert_eq!(offset.parent(), SketchEntityId(10));
    assert!((offset.distance() + 0.5).abs() < 1e-8);
    assert_eq!(outcome.candidate().points(), geometry.points());
}
