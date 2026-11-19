// SPDX-License-Identifier: GPL-2.0-only
//! Exact Shell inputs, recipe validation and parameter dependencies (#4106).
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        FeatureId, FeatureReadKey,
        shell::{ShellDefinition, ShellDirection, ShellFaceThickness, ShellRecipe},
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
fn key(document: DocumentGuid, feature: u64, kind: u32, id: u64) -> Vec<u8> {
    SemanticReferenceName::new(document, feature, kind, "fixture", "shell-test", &[id])
        .unwrap()
        .encode()
        .unwrap()
}
fn body() -> Vec<u8> {
    key(document(), 1, 4, 1)
}
fn face(feature: u64, id: u64) -> Vec<u8> {
    key(document(), feature, 1, id)
}
fn parameters(value: f64, dimension: Dimension) -> ParameterCollection<u64> {
    let mut result = ParameterCollection::new(UnitsOfMeasure::default());
    result
        .insert_value(3, "thickness", USER_PARAM, Quantity::new(value, dimension))
        .unwrap();
    result
}
#[test]
fn shell_recipe_is_canonical_and_tracks_sources_and_scalars() {
    let openings = vec![face(2, 3), face(1, 2)];
    let overrides = vec![ShellFaceThickness {
        face: face(3, 4),
        thickness: DatumLength::Parameter(4),
    }];
    let first = ShellDefinition::new(
        body(),
        openings.clone(),
        DatumLength::Parameter(3),
        ShellDirection::Inside,
        overrides.clone(),
    )
    .unwrap();
    let second = ShellDefinition::new(
        body(),
        openings.into_iter().rev().collect(),
        DatumLength::Parameter(3),
        ShellDirection::Inside,
        overrides,
    )
    .unwrap();
    assert_eq!(first, second);
    let encoded = serde_json::to_string(&first.recipe()).unwrap();
    let saved: ShellRecipe = serde_json::from_str(&encoded).unwrap();
    assert_eq!(ShellDefinition::from_recipe(&saved).unwrap(), first);
    for id in [1, 2, 3] {
        assert!(
            first
                .reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(id)))
        );
    }
    for id in [3, 4] {
        assert!(
            first
                .reads()
                .keys()
                .contains(&FeatureReadKey::Parameter(id))
        );
    }
    assert_eq!(
        first
            .resolved_thickness(&parameters(2., Dimension::Length))
            .unwrap(),
        2.
    );
}
#[test]
fn shell_rejects_invalid_and_ambiguous_references() {
    let foreign: DocumentGuid = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap();
    let bad = [
        (body(), vec![], DatumLength::Constant(1.), vec![]),
        (
            face(1, 1),
            vec![face(1, 2)],
            DatumLength::Constant(1.),
            vec![],
        ),
        (body(), vec![body()], DatumLength::Constant(1.), vec![]),
        (
            body(),
            vec![face(1, 2), face(1, 2)],
            DatumLength::Constant(1.),
            vec![],
        ),
        (
            body(),
            vec![key(foreign, 1, 1, 2)],
            DatumLength::Constant(1.),
            vec![],
        ),
        (body(), vec![face(1, 2)], DatumLength::Constant(0.), vec![]),
        (body(), vec![face(1, 2)], DatumLength::Parameter(0), vec![]),
        (
            body(),
            vec![face(1, 2)],
            DatumLength::Constant(1.),
            vec![ShellFaceThickness {
                face: face(1, 2),
                thickness: DatumLength::Constant(2.),
            }],
        ),
    ];
    for (body, faces, thickness, overrides) in bad {
        assert!(
            ShellDefinition::new(body, faces, thickness, ShellDirection::Inside, overrides)
                .is_err()
        );
    }
}
#[test]
fn shell_rejects_invalid_resolved_values_and_recipes() {
    let definition = ShellDefinition::new(
        body(),
        vec![face(1, 2)],
        DatumLength::Parameter(3),
        ShellDirection::Both,
        vec![],
    )
    .unwrap();
    assert!(
        definition
            .resolved_thickness(&parameters(-1., Dimension::Length))
            .is_err()
    );
    assert!(
        definition
            .resolved_thickness(&parameters(1., Dimension::Angle))
            .is_err()
    );
    let mut recipe = definition.recipe();
    recipe.version = 2;
    assert!(ShellDefinition::from_recipe(&recipe).is_err());
    let mut value = serde_json::to_value(definition.recipe()).unwrap();
    value
        .as_object_mut()
        .unwrap()
        .insert("unexpected".into(), true.into());
    assert!(serde_json::from_value::<ShellRecipe>(value).is_err());
}
