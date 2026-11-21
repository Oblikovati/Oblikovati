// SPDX-License-Identifier: GPL-2.0-only
//! Authored fillet references and scalar dependencies (#4045).
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        FeatureId, FeatureReadKey,
        fillet::{FilletDefinition, FilletRecipe},
        work_geometry::DatumLength,
    },
    identity::{DocumentGuid, SemanticReferenceName},
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn document() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}
fn key(document: DocumentGuid, feature: u64, kind: u32, role: &str, id: u64) -> Vec<u8> {
    SemanticReferenceName::new(document, feature, kind, "fixture", role, &[id])
        .unwrap()
        .encode()
        .unwrap()
}
fn body() -> Vec<u8> {
    key(document(), 1, 4, "body", 1)
}
fn edge(feature: u64, id: u64) -> Vec<u8> {
    key(document(), feature, 2, "edge", id)
}
fn lengths(value: f64, dimension: Dimension) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_value(3, "radius", USER_PARAM, Quantity::new(value, dimension))
        .unwrap();
    parameters
}

#[test]
fn fillet_recipe_is_canonical_and_tracks_every_reference_and_parameter() {
    let (a, b) = (edge(1, 2), edge(2, 3));
    let first = FilletDefinition::new(
        body(),
        vec![b.clone(), a.clone()],
        DatumLength::Parameter(3),
    )
    .unwrap();
    let second = FilletDefinition::new(body(), vec![a, b], DatumLength::Parameter(3)).unwrap();
    assert_eq!(first.recipe(), second.recipe());
    let encoded = serde_json::to_string(&first.recipe()).unwrap();
    let recipe: FilletRecipe = serde_json::from_str(&encoded).unwrap();
    assert_eq!(FilletDefinition::from_recipe(&recipe).unwrap(), first);
    let reads = first.reads();
    assert!(
        reads
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(1)))
    );
    assert!(
        reads
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(2)))
    );
    assert!(reads.keys().contains(&FeatureReadKey::Parameter(3)));
    assert!(!reads.has_unattributed_reads());
    let mut parameters = lengths(2., Dimension::Length);
    assert_eq!(first.resolved_radius(&parameters).unwrap(), 2.);
    parameters
        .set_value(3, Quantity::new(4., Dimension::Length))
        .unwrap();
    assert_eq!(first.resolved_radius(&parameters).unwrap(), 4.);
}

#[test]
fn fillet_rejects_bad_keys_duplicates_foreign_documents_and_invalid_sources() {
    let foreign: DocumentGuid = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap();
    let cases = [
        (body(), vec![], DatumLength::Constant(1.)),
        (edge(1, 1), vec![edge(1, 2)], DatumLength::Constant(1.)),
        (body(), vec![body()], DatumLength::Constant(1.)),
        (
            body(),
            vec![edge(1, 2), edge(1, 2)],
            DatumLength::Constant(1.),
        ),
        (
            body(),
            vec![key(foreign, 1, 2, "edge", 2)],
            DatumLength::Constant(1.),
        ),
        (body(), vec![edge(1, 2)], DatumLength::Constant(0.)),
        (body(), vec![edge(1, 2)], DatumLength::Constant(f64::NAN)),
        (body(), vec![edge(1, 2)], DatumLength::Parameter(0)),
    ];
    for (body, edges, radius) in cases {
        assert!(FilletDefinition::new(body, edges, radius).is_err());
    }
}

#[test]
fn fillet_rejects_invalid_resolved_radius_and_future_recipes() {
    let definition =
        FilletDefinition::new(body(), vec![edge(1, 2)], DatumLength::Parameter(3)).unwrap();
    assert!(
        definition
            .resolved_radius(&lengths(-2., Dimension::Length))
            .is_err()
    );
    assert!(
        definition
            .resolved_radius(&lengths(2., Dimension::Angle))
            .is_err()
    );
    let mut recipe = definition.recipe();
    recipe.version = 2;
    assert!(FilletDefinition::from_recipe(&recipe).is_err());
}

#[test]
fn fillet_recipe_revalidates_edges_and_refuses_unknown_fields() {
    let definition =
        FilletDefinition::new(body(), vec![edge(1, 2)], DatumLength::Constant(1.)).unwrap();
    let mut recipe = definition.recipe();
    recipe.edges.push(recipe.edges[0].clone());
    assert!(FilletDefinition::from_recipe(&recipe).is_err());
    let mut encoded = serde_json::to_value(definition.recipe()).unwrap();
    encoded
        .as_object_mut()
        .unwrap()
        .insert("unknown".into(), true.into());
    assert!(serde_json::from_value::<FilletRecipe>(encoded).is_err());
}

#[test]
fn chamfer_recipe_tracks_sources_and_parameter() {
    use oblikovati_model::feature::chamfer::{ChamferDefinition, ChamferRecipe};
    let definition = ChamferDefinition::new(
        body(),
        vec![edge(2, 3), edge(1, 2)],
        DatumLength::Parameter(3),
    )
    .unwrap();
    let recipe = definition.recipe();
    let encoded = serde_json::to_string(&recipe).unwrap();
    let restored: ChamferRecipe = serde_json::from_str(&encoded).unwrap();
    assert_eq!(
        ChamferDefinition::from_recipe(&restored).unwrap(),
        definition
    );
    assert!(
        definition
            .reads()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(2)))
    );
    assert!(
        definition
            .reads()
            .keys()
            .contains(&FeatureReadKey::Parameter(3))
    );
    assert_eq!(
        definition
            .resolved_distance(&lengths(2., Dimension::Length))
            .unwrap(),
        2.
    );
}

#[test]
fn chamfer_rejects_bad_inputs_and_recipe_versions() {
    use oblikovati_model::feature::chamfer::ChamferDefinition;
    let edge = edge(1, 2);
    for (edges, distance) in [
        (vec![], DatumLength::Constant(1.)),
        (vec![edge.clone(), edge.clone()], DatumLength::Constant(1.)),
        (vec![body()], DatumLength::Constant(1.)),
        (vec![edge.clone()], DatumLength::Constant(0.)),
        (vec![edge.clone()], DatumLength::Parameter(0)),
    ] {
        assert!(ChamferDefinition::new(body(), edges, distance).is_err());
    }
    let definition = ChamferDefinition::new(body(), vec![edge], DatumLength::Constant(1.)).unwrap();
    let mut recipe = definition.recipe();
    recipe.version = 2;
    assert!(ChamferDefinition::from_recipe(&recipe).is_err());
}
