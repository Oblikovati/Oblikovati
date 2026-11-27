// SPDX-License-Identifier: GPL-2.0-only
//! Manual retained-allocation comparison for sketch frame source evidence (#4107).
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::DocumentSnapshot,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use std::hint::black_box;

fn resident_kib() -> u64 {
    let status = std::fs::read_to_string("/proc/self/status").unwrap();
    status
        .lines()
        .find_map(|line| line.strip_prefix("VmRSS:"))
        .unwrap()
        .split_whitespace()
        .next()
        .unwrap()
        .parse()
        .unwrap()
}
fn fixture(count: u64) -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    for index in 0..count {
        let mut geometry = Sketch2D::default();
        geometry
            .add_point(SketchEntityId(index + 1), [1., 2.])
            .unwrap();
        snapshot
            .sketches
            .add(
                DocumentSketchId(index + 1),
                &format!("Sketch{index}"),
                SketchPlane::xy(),
                geometry,
            )
            .unwrap();
    }
    snapshot
}
#[test]
#[ignore = "manual Linux RSS comparison; run alone with --test-threads=1"]
fn measure_evidence_allocation_upper_bound() {
    for count in [2, 128] {
        let snapshot = fixture(count);
        let before = resident_kib();
        let copies: Vec<_> = (0..128).map(|_| snapshot.clone()).collect();
        let copied_kib = resident_kib().saturating_sub(before);
        black_box(&copies);
        eprintln!("{count} sketches: 128 full snapshot clones increase RSS by {copied_kib} KiB");
    }
}

struct FakePartEvaluator;
impl oblikovati_model::feature::part::PartFeatureEvaluator for FakePartEvaluator {
    type Body = ();
    type Failure = &'static str;
    fn verify_body_reference(&self, _: &(), key: &[u8]) -> Result<(), String> {
        Err(format!("body {key:?}: expected no body in this fixture"))
    }
    fn resolve_planar_face(&self, _: &(), key: &[u8]) -> Result<SketchPlane, String> {
        Err(format!("face {key:?}: expected no face in this fixture"))
    }
    fn evaluate_blend(
        &mut self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        _: oblikovati_model::feature::FeatureId,
        _: &(),
        _: &[Vec<u8>],
        _: f64,
    ) -> Result<(), oblikovati_model::feature::part::PartFeatureFailure<Self::Failure>> {
        Err(kind.unsupported())
    }
    fn evaluate_prism(
        &mut self,
        _: oblikovati_model::feature::FeatureId,
        _: oblikovati_model::feature::sketch_prism::ResolvedSketchPrism<'_>,
    ) -> Result<(), Self::Failure> {
        Err("expected no prism in this fixture")
    }
}
#[test]
#[ignore = "manual Linux RSS comparison; run alone with --test-threads=1"]
fn measure_shared_part_evidence_copies() {
    for count in [2, 128] {
        let snapshot = fixture(count);
        let part = snapshot
            .evaluate_part_features(&mut FakePartEvaluator)
            .unwrap();
        let evidence = part.datum_evaluation();
        let before = resident_kib();
        let copies: Vec<_> = (0..2048).map(|_| evidence.clone()).collect();
        let retained_kib = resident_kib().saturating_sub(before);
        black_box(&copies);
        eprintln!(
            "{count} sketches: 2048 cloned part datum evaluations increase RSS by {retained_kib} KiB"
        );
    }
}
