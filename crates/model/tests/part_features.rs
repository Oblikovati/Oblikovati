// SPDX-License-Identifier: GPL-2.0-only
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
use fixture::*;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        part::PartFeatureDefinition,
        sketch_prism::SketchPrismDefinition,
        work_geometry::DatumLength,
        work_plane::{WorkPlaneDefinition, WorkPlaneResolver},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
    sketch::SketchPlane,
};
fn part_document() -> DocumentSnapshot {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(1),
            "Base",
            WorkPlaneDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    document
        .sketches
        .attach_work_plane(SKETCH, FeatureId(1))
        .unwrap();
    document
}
fn part_owner() -> DocumentEditHistory {
    let document = part_document();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    DocumentEditHistory::new(document, ids).unwrap()
}
fn prism_source() -> SketchPrismDefinition {
    SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
fn append_prism(owner: &mut DocumentEditHistory) -> FeatureId {
    let token = owner.begin("Add prism inputs").unwrap();
    let id = owner
        .create_sketch_prism(&token, "Extrude", prism_source())
        .unwrap();
    owner.commit(&token, false).unwrap();
    id
}
fn assert_prism_definition(owner: &DocumentEditHistory) {
    assert!(matches!(
        owner
            .snapshot()
            .work_geometry
            .get(FeatureId(2))
            .unwrap()
            .definition(),
        PartFeatureDefinition::SketchPrism(_)
    ));
}
#[test]
fn body_and_datums_share_one_namespace_order_and_undo_owner() {
    let mut owner = part_owner();
    assert_eq!(append_prism(&mut owner), FeatureId(2));
    let stored = owner.snapshot().work_geometry.clone();
    owner.undo().unwrap();
    assert_eq!(owner.snapshot().work_geometry.features().len(), 1);
    assert_eq!(owner.feature_high_water(), 2);
    owner.redo().unwrap();
    assert!(
        owner
            .snapshot()
            .work_geometry
            .same_work_geometry_recipe(&stored)
    );
    assert_prism_definition(&owner);
    owner.undo().unwrap();
    assert_eq!(append_prism(&mut owner), FeatureId(3));
}
#[test]
fn mixed_program_round_trip_uses_one_serializer_and_version_gate() {
    let mut owner = part_owner();
    let initial = owner.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(initial.version, 1);
    append_prism(&mut owner);
    let recipe = owner.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 16);
    let encoded = serde_json::to_string(&recipe).unwrap();
    let reopened =
        FeatureSequence::from_part_recipe(&serde_json::from_str(&encoded).unwrap()).unwrap();
    assert!(reopened.same_work_geometry_recipe(&owner.snapshot().work_geometry));
    let mut old = recipe;
    old.version = 15;
    assert!(FeatureSequence::from_part_recipe(&old).is_err());
}
#[test]
fn body_inputs_protect_their_sketch_and_host_datum_from_removal_or_forward_reorder() {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut candidate = owner.snapshot().clone();
    assert!(candidate.delete_sketch(SKETCH).is_err());
    assert!(candidate.work_geometry.remove(FeatureId(1)).is_err());
    assert!(candidate.work_geometry.reorder(FeatureId(2), 0).is_err());
    assert!(
        candidate
            .work_geometry
            .same_work_geometry_recipe(&owner.snapshot().work_geometry)
    );
}
#[test]
fn datum_projection_keeps_independent_frames_and_names_body_scope() {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let view = owner
        .snapshot()
        .work_geometry
        .evaluate_work_geometry(&owner.snapshot().parameters)
        .unwrap();
    assert_eq!(view.plane(FeatureId(1)).unwrap(), SketchPlane::xy());
    assert!(view.plane(FeatureId(2)).is_err());
    assert!(owner.snapshot().require_resolved_sketch(SKETCH).is_ok());
}
#[path = "part_features/reads.rs"]
mod reads;
#[path = "part_features/refusals.rs"]
mod refusals;
