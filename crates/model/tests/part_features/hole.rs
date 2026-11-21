// SPDX-License-Identifier: GPL-2.0-only
//! Hole transactions, named-body replacement and replay (#4093).
use super::*;
use oblikovati_model::{
    feature::{
        FeatureEvaluationStatus,
        hole::{HoleDefinition, HoleExtent, HolePosition, HoleRecipe, ResolvedHole},
        part::{PartBlendKind, PartFeatureEvaluator, PartFeatureFailure},
        sketch_prism::ResolvedSketchPrism,
    },
    identity::{DocumentGuid, SemanticReferenceName},
};
use std::collections::BTreeSet;

fn name(feature: u64, kind: u32) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, feature, kind, "hole-test", "source", &[1])
        .unwrap()
        .encode()
        .unwrap()
}
fn input() -> HoleDefinition {
    HoleDefinition::from_recipe(HoleRecipe {
        version: 1,
        body: name(2, 4),
        face: name(2, 1),
        position: HolePosition::FaceCentroid,
        diameter: DatumLength::Parameter(3),
        extent: HoleExtent::ThroughAll,
    })
    .unwrap()
}
#[derive(Clone, Debug)]
struct TestBody {
    names: BTreeSet<Vec<u8>>,
    diameter: Option<f64>,
}
#[derive(Default)]
struct TestHoleBackend {
    reject: bool,
    lose_face: bool,
}
impl PartFeatureEvaluator for TestHoleBackend {
    type Body = TestBody;
    type Failure = String;
    fn verify_body_reference(&self, body: &TestBody, key: &[u8]) -> Result<(), String> {
        body.names
            .contains(key)
            .then_some(())
            .ok_or_else(|| format!("body {key:?}: expected known name"))
    }
    fn resolve_planar_face(&self, body: &TestBody, key: &[u8]) -> Result<SketchPlane, String> {
        body.names
            .contains(key)
            .then_some(SketchPlane::xy())
            .ok_or_else(|| format!("face {key:?}: expected known name"))
    }
    fn evaluate_prism(
        &mut self,
        id: FeatureId,
        _: ResolvedSketchPrism<'_>,
    ) -> Result<TestBody, String> {
        Ok(TestBody {
            names: [name(id.0, 4), name(id.0, 1)].into_iter().collect(),
            diameter: None,
        })
    }
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        _: FeatureId,
        _: &TestBody,
        _: &[Vec<u8>],
        _: f64,
    ) -> Result<TestBody, PartFeatureFailure<String>> {
        Err(kind.unsupported())
    }
    fn evaluate_hole(
        &mut self,
        id: FeatureId,
        body: &TestBody,
        _: &[u8],
        input: ResolvedHole,
    ) -> Result<TestBody, PartFeatureFailure<String>> {
        if self.reject {
            return Err(PartFeatureFailure::Kernel("hole rejected".into()));
        }
        let mut result = body.clone();
        result.names.insert(name(id.0, 4));
        if self.lose_face {
            result.names.remove(&name(2, 1));
        }
        result.diameter = Some(input.diameter);
        Ok(result)
    }
}
fn owner_with_hole() -> DocumentEditHistory {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let token = owner.begin("Add hole").unwrap();
    assert_eq!(
        owner.create_hole(&token, "Hole", input()).unwrap(),
        FeatureId(3)
    );
    owner.commit(&token, false).unwrap();
    owner
}

#[test]
fn hole_replaces_current_body_and_round_trips_program_recipe() {
    let owner = owner_with_hole();
    let snapshot = owner.snapshot();
    let result = snapshot
        .evaluate_part_features(&mut TestHoleBackend::default())
        .unwrap();
    assert!(result.body(FeatureId(2)).is_none());
    assert_eq!(result.body(FeatureId(3)).unwrap().diameter, Some(2.));
    assert!(
        result
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy)
    );
    let mut recipe = snapshot.work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 22);
    let restored = FeatureSequence::from_part_recipe(&recipe).unwrap();
    assert!(snapshot.work_geometry.same_work_geometry_recipe(&restored));
    recipe.version = 21;
    assert!(FeatureSequence::from_part_recipe(&recipe).is_err());
}

#[test]
fn rejected_or_unnamed_hole_keeps_the_input_body() {
    let owner = owner_with_hole();
    for mut backend in [
        TestHoleBackend {
            reject: true,
            lose_face: false,
        },
        TestHoleBackend {
            reject: false,
            lose_face: true,
        },
    ] {
        let result = owner
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap();
        assert!(result.body(FeatureId(2)).is_some());
        assert!(result.body(FeatureId(3)).is_none());
        assert!(matches!(
            &result.health()[2].status,
            FeatureEvaluationStatus::Failed(_)
        ));
    }
}

#[test]
fn hole_history_undo_redo_restores_the_authored_recipe() {
    let mut owner = owner_with_hole();
    let original = owner.snapshot().work_geometry.part_recipe().unwrap();
    owner.undo().unwrap();
    assert!(owner.snapshot().work_geometry.get(FeatureId(3)).is_err());
    owner.redo().unwrap();
    assert_eq!(
        owner.snapshot().work_geometry.part_recipe().unwrap(),
        original
    );
}

#[test]
fn hole_replay_uses_the_current_driving_parameter() {
    use oblikovati_model::quantity::{Dimension, Quantity};
    let owner = owner_with_hole();
    let mut changed = owner.snapshot().clone();
    changed
        .parameters
        .set_value(3, Quantity::new(3., Dimension::Length))
        .unwrap();
    let result = changed
        .evaluate_part_features(&mut TestHoleBackend::default())
        .unwrap();
    assert_eq!(result.body(FeatureId(3)).unwrap().diameter, Some(3.));
    assert!(
        result
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy)
    );
}
