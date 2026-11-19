// SPDX-License-Identifier: GPL-2.0-only
//! Ordered Shell replay and atomic failure behavior (#4106).
use super::*;
use oblikovati_model::{
    feature::{
        FeatureEvaluationStatus,
        part::{PartBlendKind, PartFeatureEvaluator, PartFeatureFailure},
        shell::{ResolvedShell, ShellDefinition, ShellDirection},
        sketch_prism::ResolvedSketchPrism,
    },
    identity::{DocumentGuid, SemanticReferenceName},
};

fn key(kind: u32, role: &str) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, 2, kind, "fixture", role, &[1])
        .unwrap()
        .encode()
        .unwrap()
}
#[derive(Clone, Debug)]
struct FakeBody {
    shelled: bool,
}
#[derive(Default)]
struct FakeShellBackend {
    refuse: bool,
}
impl PartFeatureEvaluator for FakeShellBackend {
    type Body = FakeBody;
    type Failure = String;
    fn verify_body_reference(&self, _: &FakeBody, name: &[u8]) -> Result<(), String> {
        if name == key(4, "body") {
            Ok(())
        } else {
            Err("unknown body".into())
        }
    }
    fn resolve_planar_face(&self, _: &FakeBody, _: &[u8]) -> Result<SketchPlane, String> {
        Ok(SketchPlane::xy())
    }
    fn evaluate_prism(
        &mut self,
        _: FeatureId,
        _: ResolvedSketchPrism<'_>,
    ) -> Result<FakeBody, String> {
        Ok(FakeBody { shelled: false })
    }
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        _: FeatureId,
        _: &FakeBody,
        _: &[Vec<u8>],
        _: f64,
    ) -> Result<FakeBody, PartFeatureFailure<String>> {
        Err(kind.unsupported())
    }
    fn evaluate_shell(
        &mut self,
        _: FeatureId,
        _: &FakeBody,
        faces: &[Vec<u8>],
        input: ResolvedShell,
    ) -> Result<FakeBody, PartFeatureFailure<String>> {
        if self.refuse {
            return Err(PartFeatureFailure::Kernel("tolerance refused".into()));
        }
        assert_eq!(faces, [key(1, "face")]);
        assert_eq!(input.thickness, 0.1);
        Ok(FakeBody { shelled: true })
    }
}
fn with_shell() -> DocumentSnapshot {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut snapshot = owner.snapshot().clone();
    let shell = ShellDefinition::new(
        key(4, "body"),
        vec![key(1, "face")],
        DatumLength::Constant(0.1),
        ShellDirection::Inside,
        vec![],
    )
    .unwrap();
    let reads = shell.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Shell",
            PartFeatureDefinition::Shell(Box::new(shell)),
            reads,
        ))
        .unwrap();
    snapshot
}
#[test]
fn shell_recipe_roundtrips_and_replaces_the_source_body() {
    let snapshot = with_shell();
    let recipe = snapshot.work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 24);
    let restored = FeatureSequence::from_part_recipe(&recipe).unwrap();
    assert!(restored.same_work_plane_recipe(&snapshot.work_geometry));
    let result = snapshot
        .evaluate_part_features(&mut FakeShellBackend::default())
        .unwrap();
    assert!(result.body(FeatureId(2)).is_none());
    assert!(result.body(FeatureId(3)).unwrap().shelled);
}
#[test]
fn shell_refusal_keeps_the_input_body_and_reports_failed_health() {
    let result = with_shell()
        .evaluate_part_features(&mut FakeShellBackend { refuse: true })
        .unwrap();
    assert!(!result.body(FeatureId(2)).unwrap().shelled);
    assert!(result.body(FeatureId(3)).is_none());
    assert!(
        matches!(&result.health()[2].status, FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel(message)) if message == "tolerance refused")
    );
}
