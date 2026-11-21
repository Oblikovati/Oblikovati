// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::prism_test_fixture as fixture;

use oblikovati_model::{
    feature::{FeatureId, sketch_prism::SketchDirectedCurve},
    identity::DocumentGuid,
    sketch::SketchEntityId,
};
use oblikovati_persistence::package::{DocumentPackage, LegacyExtrusionFaceRole};

const ORIGINAL: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-cube-fixture/go-cube.obk"
);
const REORDERED: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-face-lineage/reordered.obk"
);
const ORIGINAL_ROWS: &str = include_str!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-face-lineage/original.json"
);
const REORDERED_ROWS: &str = include_str!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-face-lineage/reordered.json"
);

#[derive(serde::Deserialize)]
struct Oracle {
    before: Vec<OracleEdge>,
}
#[derive(serde::Deserialize)]
struct OracleEdge {
    side_index: usize,
    source_line_id: u64,
}
struct GoOracleWitness {
    rows: Vec<OracleEdge>,
}
impl LegacyProfileOrderWitness for GoOracleWitness {
    fn boundary(
        &self,
        _: &LegacySketchCandidate,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String> {
        if sketch.0 != 1 || index != 0 {
            return Err("Go profile witness unavailable".into());
        }
        let mut outer = Vec::new();
        for (index, edge) in self.rows.iter().enumerate() {
            if edge.side_index != index {
                return Err("Go profile witness order invalid".into());
            }
            outer.push(SketchDirectedCurve {
                curve: SketchEntityId(edge.source_line_id),
                reversed: false,
            });
        }
        Ok(SketchPrismSelection {
            outer,
            holes: Vec::new(),
        })
    }
}

struct TestOnlyAssumedPrecision;
impl LegacyCapStageVerifier for TestOnlyAssumedPrecision {
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String> {
        fixture::solve_selected_sketches(snapshot);
        Ok(())
    }
    fn qualify_sketch(&self, _: &DocumentSnapshot, _: DocumentSketchId) -> Result<(), String> {
        Ok(())
    }
    fn verify(
        &self,
        _: &DocumentSnapshot,
        _: DocumentGuid,
        _: FeatureId,
        _: &[u8],
        _: LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        Err("cap verification not used".into())
    }
}
struct NoProductionPrecision;
impl LegacyCapStageVerifier for NoProductionPrecision {
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String> {
        TestOnlyAssumedPrecision.solve(snapshot)
    }
    fn verify(
        &self,
        _: &DocumentSnapshot,
        _: DocumentGuid,
        _: FeatureId,
        _: &[u8],
        _: LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        Err("cap verification not used".into())
    }
}
fn source(bytes: &[u8]) -> (DocumentPackage, DocumentSnapshot) {
    let package = DocumentPackage::decode(bytes).unwrap();
    let candidate = package.prepare_legacy_sketch_candidate().unwrap();
    let mut base = fixture::selected_sketch_document();
    base.sketches = candidate.sketches;
    base.parameters = candidate.parameters.parameters;
    base.work_geometry = Default::default();
    (package, base)
}
fn witness(rows: &str) -> GoOracleWitness {
    GoOracleWitness {
        rows: serde_json::from_str::<Oracle>(rows).unwrap().before,
    }
}

#[test]
fn witnessed_profiles_match_certified_regions_for_both_go_source_orders() {
    for (fixture, rows) in [(ORIGINAL, ORIGINAL_ROWS), (REORDERED, REORDERED_ROWS)] {
        let (package, base) = source(fixture);
        let witness = witness(rows);
        let resolver = WitnessedLegacyProfileResolver::new(
            &base,
            test_policy(),
            &TestOnlyAssumedPrecision,
            &witness,
        );
        let candidate = package.prepare_legacy_part_candidate(&resolver).unwrap();
        assert_eq!(candidate.extrusions.len(), 1);
        assert_eq!(
            candidate.extrusions[0].definition.selection().outer.len(),
            4
        );
    }
}

#[test]
fn missing_precision_or_source_witness_declines_without_changing_the_base() {
    let (package, base) = source(ORIGINAL);
    let sketch = package.prepare_legacy_sketch_candidate().unwrap();
    let before = base.clone();
    let witness = witness(ORIGINAL_ROWS);
    let resolver =
        WitnessedLegacyProfileResolver::new(&base, test_policy(), &NoProductionPrecision, &witness);
    let error = resolver
        .resolve(&sketch, DocumentSketchId(1), 0)
        .unwrap_err();
    assert!(error.contains("qualified spatial and angular"), "{error}");
    let resolver = WitnessedLegacyProfileResolver::new(
        &base,
        test_policy(),
        &TestOnlyAssumedPrecision,
        &witness,
    );
    assert!(resolver.resolve(&sketch, DocumentSketchId(1), 9).is_err());
    assert!(base.sketches.same_recipe(&before.sketches));
    assert!(base.parameters.same_recipe(&before.parameters));
    assert!(base.same_metadata_recipe(&before));
}

#[test]
fn incorrect_source_boundary_cannot_bind_a_different_region() {
    let (package, base) = source(ORIGINAL);
    let sketch = package.prepare_legacy_sketch_candidate().unwrap();
    let mut witness = witness(ORIGINAL_ROWS);
    witness.rows[1].source_line_id = 999;
    let resolver = WitnessedLegacyProfileResolver::new(
        &base,
        test_policy(),
        &TestOnlyAssumedPrecision,
        &witness,
    );
    let error = resolver
        .resolve(&sketch, DocumentSketchId(1), 0)
        .unwrap_err();
    assert!(error.contains("exactly one certified region"), "{error}");
}

fn test_policy() -> PrismResolutionPolicy {
    PrismResolutionPolicy::Fixed(oblikovati_martensite::PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    })
}

#[test]
fn duplicate_certified_region_matches_are_rejected() {
    let (package, base) = source(ORIGINAL);
    let sketch = package.prepare_legacy_sketch_candidate().unwrap();
    let snapshot = sketch
        .prepare_qualified_profile(&base, DocumentSketchId(1), &TestOnlyAssumedPrecision)
        .unwrap();
    let regions =
        certified_sketch_regions_with_policy(&snapshot, DocumentSketchId(1), test_policy())
            .unwrap();
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let expected = witness(ORIGINAL_ROWS)
        .boundary(&sketch, DocumentSketchId(1), 0)
        .unwrap();
    let duplicate = vec![regions[0].clone(), regions[0].clone()];
    let error = select_matching_region(&duplicate, geometry, &expected, DocumentSketchId(1), 0)
        .unwrap_err();
    assert!(error.contains("found 2"), "{error}");
}

#[test]
fn mismatched_source_snapshot_is_refused_before_profile_binding() {
    let (package, mut base) = source(ORIGINAL);
    let candidate = package.prepare_legacy_sketch_candidate().unwrap();
    let size = base.parameters.find("size").unwrap();
    base.parameters.set_expression(size, "20 mm").unwrap();
    let witness = witness(ORIGINAL_ROWS);
    let resolver = WitnessedLegacyProfileResolver::new(
        &base,
        test_policy(),
        &TestOnlyAssumedPrecision,
        &witness,
    );
    let error = resolver
        .resolve(&candidate, DocumentSketchId(1), 0)
        .unwrap_err();
    assert!(error.contains("source sketches and parameters"), "{error}");
}
