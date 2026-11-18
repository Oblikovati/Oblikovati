// SPDX-License-Identifier: GPL-2.0-only
//! Named-body replacement and fillet replay (#4045).
use super::*;
use oblikovati_model::{
    feature::{
        FeatureEvaluationStatus,
        fillet::FilletDefinition,
        part::{DeleteBodyDefinition, PartBlendKind, PartFeatureEvaluator, PartFeatureFailure},
        sketch_prism::ResolvedSketchPrism,
        work_plane::{WorkPlaneFaceReference, WorkPlaneReference},
    },
    identity::{DocumentGuid, SemanticReferenceName},
};
use std::collections::BTreeSet;

fn key(feature: u64, kind: u32, role: &str) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, feature, kind, "fixture", role, &[1])
        .unwrap()
        .encode()
        .unwrap()
}
#[derive(Clone, Debug)]
struct NamedBody {
    keys: BTreeSet<Vec<u8>>,
    deleted: BTreeSet<Vec<u8>>,
    radius: Option<f64>,
}
#[derive(Default)]
struct NamedFilletBackend {
    reject: bool,
    unsupported: bool,
    lose_names: bool,
}
impl PartFeatureEvaluator for NamedFilletBackend {
    type Body = NamedBody;
    type Failure = String;
    fn verify_body_reference(&self, body: &NamedBody, key: &[u8]) -> Result<(), String> {
        body.keys
            .contains(key)
            .then_some(())
            .ok_or_else(|| format!("body {key:?}: expected a current exact name"))
    }
    fn resolve_planar_face(&self, body: &NamedBody, key: &[u8]) -> Result<SketchPlane, String> {
        body.keys
            .contains(key)
            .then_some(SketchPlane::xy())
            .ok_or_else(|| format!("face {key:?}: expected a current exact name"))
    }
    fn evaluate_prism(
        &mut self,
        id: FeatureId,
        _: ResolvedSketchPrism<'_>,
    ) -> Result<NamedBody, String> {
        Ok(NamedBody {
            keys: [
                key(id.0, 4, "body"),
                key(id.0, 1, "face"),
                key(id.0, 2, "edge"),
                key(id.0, 2, "other-edge"),
            ]
            .into_iter()
            .collect(),
            deleted: BTreeSet::new(),
            radius: None,
        })
    }
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        id: FeatureId,
        body: &NamedBody,
        edges: &[Vec<u8>],
        radius: f64,
    ) -> Result<NamedBody, PartFeatureFailure<String>> {
        if self.unsupported || kind != PartBlendKind::Fillet {
            return Err(kind.unsupported());
        }
        let result = self
            .evaluate_fillet(id, body, edges, radius)
            .map_err(PartFeatureFailure::Kernel)?;
        self.verify_fillet_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
impl NamedFilletBackend {
    fn verify_fillet_lineage(&self, before: &NamedBody, after: &NamedBody) -> Result<(), String> {
        (before.keys.is_subset(&after.keys) && before.deleted.is_subset(&after.deleted))
            .then_some(())
            .ok_or_else(|| "fillet discarded input names".into())
    }

    fn evaluate_fillet(
        &mut self,
        id: FeatureId,
        body: &NamedBody,
        edges: &[Vec<u8>],
        radius: f64,
    ) -> Result<NamedBody, String> {
        if self.reject
            || edges
                .iter()
                .any(|edge| !body.keys.contains(edge) || body.deleted.contains(edge))
        {
            return Err("fillet rejected".into());
        }
        let mut result = body.clone();
        result.deleted.extend(edges.iter().cloned());
        result.keys.insert(key(id.0, 4, "body"));
        if self.lose_names {
            result.keys = [key(id.0, 4, "body")].into_iter().collect();
        }
        result.radius = Some(radius);
        Ok(result)
    }
}
fn document() -> DocumentSnapshot {
    let mut owner = part_owner();
    append_prism(&mut owner);
    owner.snapshot().clone()
}
fn add_fillet(snapshot: &mut DocumentSnapshot, id: u64, source: u64) {
    add_fillet_edge(snapshot, id, source, "edge");
}
fn add_fillet_edge(snapshot: &mut DocumentSnapshot, id: u64, source: u64, edge: &str) {
    let definition = FilletDefinition::new(
        key(source, 4, "body"),
        vec![key(2, 2, edge)],
        DatumLength::Constant(1.),
    )
    .unwrap();
    let reads = definition.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(id),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        ))
        .unwrap();
}
fn add_dependent_plane(snapshot: &mut DocumentSnapshot, id: u64) {
    let reference = WorkPlaneFaceReference::new(key(2, 1, "face")).unwrap();
    let plane = PartFeatureDefinition::Offset {
        base: WorkPlaneReference::Face(reference),
        distance: DatumLength::Constant(1.),
    };
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(id),
            "Dependent plane",
            plane,
        ))
        .unwrap();
}
#[test]
fn fillet_replaces_body_and_preserves_old_face_names_through_replay() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    add_dependent_plane(&mut snapshot, 4);
    let evaluated = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert!(evaluated.body(FeatureId(2)).is_none());
    assert_eq!(evaluated.body(FeatureId(3)).unwrap().radius, Some(1.));
    assert_eq!(evaluated.ordered_bodies().count(), 1);
    assert!(evaluated.plane(FeatureId(4)).is_ok());
    assert!(
        evaluated
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy)
    );
}
#[test]
fn chained_fillet_preserves_ancestral_body_key_for_deletion() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    add_fillet_edge(&mut snapshot, 4, 3, "other-edge");
    let delete =
        PartFeatureDefinition::DeleteBody(DeleteBodyDefinition::new(key(2, 4, "body")).unwrap());
    let reads = delete.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(FeatureId(5), "Delete", delete, reads))
        .unwrap();
    let evaluated = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert_eq!(evaluated.ordered_bodies().count(), 0);
    assert!(evaluated.deleted_body(FeatureId(5)).is_some());
}
#[test]
fn a_deleted_edge_name_cannot_be_used_for_a_second_fillet() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    add_fillet(&mut snapshot, 4, 3);
    let evaluated = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert!(evaluated.body(FeatureId(3)).is_some());
    assert!(evaluated.body(FeatureId(4)).is_none());
    assert!(matches!(
        &evaluated.health()[3].status,
        FeatureEvaluationStatus::Failed(_)
    ));
}
#[test]
fn failed_or_unavailable_fillet_does_not_consume_the_input_body() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    for mut backend in [
        NamedFilletBackend {
            reject: true,
            unsupported: false,
            lose_names: false,
        },
        NamedFilletBackend {
            reject: false,
            unsupported: true,
            lose_names: false,
        },
    ] {
        let evaluated = snapshot.evaluate_part_features(&mut backend).unwrap();
        assert!(evaluated.body(FeatureId(2)).is_some());
        assert!(evaluated.body(FeatureId(3)).is_none());
        match &evaluated.health()[2].status {
            FeatureEvaluationStatus::Failed(
                oblikovati_model::feature::part::PartFeatureFailure::Kernel(cause),
            ) => assert_eq!(cause, "fillet rejected"),
            FeatureEvaluationStatus::Failed(
                oblikovati_model::feature::part::PartFeatureFailure::Unsupported(cause),
            ) => assert!(cause.contains("fillet")),
            other => panic!(
                "fillet health {other:?}: expected kernel refusal or named unsupported capability"
            ),
        }
    }
}

#[path = "fillet/declines.rs"]
mod declines;

#[test]
fn fillet_recipe_roundtrips_in_the_ordered_part_program() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    let mut recipe = snapshot.work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 20);
    let restored = FeatureSequence::from_part_recipe(&recipe).unwrap();
    assert!(restored.same_work_plane_recipe(&snapshot.work_geometry));
    recipe.version = 19;
    assert!(FeatureSequence::from_part_recipe(&recipe).is_err());
}

#[test]
fn failed_fillet_quarantines_later_consumers_of_preserved_names() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    add_dependent_plane(&mut snapshot, 4);
    let evaluated = snapshot
        .evaluate_part_features(&mut NamedFilletBackend {
            reject: true,
            unsupported: false,
            lose_names: false,
        })
        .unwrap();
    assert!(evaluated.body(FeatureId(2)).is_some());
    assert!(matches!(
        &evaluated.health()[3].status,
        FeatureEvaluationStatus::Blocked { drivers } if drivers.contains(&FeatureId(3))
    ));
    assert!(evaluated.plane(FeatureId(4)).is_err());
}

#[test]
fn failed_fillet_keeps_independent_features_and_suppression_passes_through() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(4),
            "Independent",
            PartFeatureDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    add_dependent_plane(&mut snapshot, 5);
    let failed = snapshot
        .evaluate_part_features(&mut NamedFilletBackend {
            reject: true,
            unsupported: false,
            lose_names: false,
        })
        .unwrap();
    assert_eq!(failed.health()[3].status, FeatureEvaluationStatus::Healthy);
    assert!(matches!(
        failed.health()[4].status,
        FeatureEvaluationStatus::Blocked { .. }
    ));
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(3), true)
        .unwrap();
    let suppressed = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert!(suppressed.body(FeatureId(2)).is_some());
    assert!(suppressed.plane(FeatureId(5)).is_ok());
}

#[test]
fn result_that_loses_input_names_is_rejected_before_body_replacement() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    let evaluated = snapshot
        .evaluate_part_features(&mut NamedFilletBackend {
            lose_names: true,
            ..Default::default()
        })
        .unwrap();
    assert!(evaluated.body(FeatureId(2)).is_some());
    assert!(evaluated.body(FeatureId(3)).is_none());
    assert!(matches!(&evaluated.health()[2].status,
        FeatureEvaluationStatus::Failed(oblikovati_model::feature::part::PartFeatureFailure::BodyReference(error))
        if error.contains("discarded input names")));
}

#[test]
fn parameter_drives_fillet_replay_and_invalid_values_keep_the_original_body() {
    use oblikovati_model::quantity::{Dimension, Quantity};
    let mut snapshot = document();
    let definition = FilletDefinition::new(
        key(2, 4, "body"),
        vec![key(2, 2, "edge")],
        DatumLength::Parameter(3),
    )
    .unwrap();
    let reads = definition.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        ))
        .unwrap();
    for value in [2., 4.] {
        snapshot
            .parameters
            .set_value(3, Quantity::new(value, Dimension::Length))
            .unwrap();
        let result = snapshot
            .evaluate_part_features(&mut NamedFilletBackend::default())
            .unwrap();
        assert_eq!(result.body(FeatureId(3)).unwrap().radius, Some(value));
    }
    snapshot
        .parameters
        .set_value(3, Quantity::new(-1., Dimension::Length))
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert!(result.body(FeatureId(2)).is_some());
    assert!(matches!(
        result.health()[2].status,
        FeatureEvaluationStatus::Failed(
            oblikovati_model::feature::part::PartFeatureFailure::FilletInput(_)
        )
    ));
}

#[test]
fn edge_from_another_body_is_refused_without_changing_either_body() {
    let mut snapshot = document();
    let prism =
        AuthoredFeature::sketch_prism(FeatureId(3), "Independent", prism_source(), &snapshot)
            .unwrap();
    snapshot.work_geometry.append(prism).unwrap();
    let fillet = FilletDefinition::new(
        key(2, 4, "body"),
        vec![key(3, 2, "edge")],
        DatumLength::Constant(1.),
    )
    .unwrap();
    let reads = fillet.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(4),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(fillet)),
            reads,
        ))
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert!(result.body(FeatureId(2)).is_some());
    assert!(result.body(FeatureId(3)).is_some());
    assert!(result.body(FeatureId(4)).is_none());
    assert!(matches!(
        result.health()[3].status,
        FeatureEvaluationStatus::Failed(_)
    ));
}

#[test]
fn invalid_body_source_does_not_block_an_independent_datum_reader() {
    let mut snapshot = document();
    let fillet = FilletDefinition::new(
        key(1, 4, "body"),
        vec![key(2, 2, "edge")],
        DatumLength::Constant(1.),
    )
    .unwrap();
    let reads = fillet.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Bad fillet",
            PartFeatureDefinition::Fillet(Box::new(fillet)),
            reads,
        ))
        .unwrap();
    let plane = PartFeatureDefinition::Offset {
        base: WorkPlaneReference::Feature(FeatureId(1)),
        distance: DatumLength::Constant(1.),
    };
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(4),
            "Independent plane",
            plane,
        ))
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert!(matches!(
        result.health()[2].status,
        FeatureEvaluationStatus::Failed(_)
    ));
    assert_eq!(result.health()[3].status, FeatureEvaluationStatus::Healthy);
    assert!(result.plane(FeatureId(4)).is_ok());
}

#[test]
fn replay_does_not_write_derived_replacement_dependencies_to_the_recipe() {
    let mut snapshot = document();
    add_fillet(&mut snapshot, 3, 2);
    add_dependent_plane(&mut snapshot, 4);
    let before = snapshot.work_geometry.part_recipe().unwrap();
    snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert_eq!(snapshot.work_geometry.part_recipe().unwrap(), before);
}
