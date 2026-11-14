// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
        work_geometry::DatumLength,
    },
    sketch::SketchEntityId,
};
use oblikovati_persistence::package::{
    DocumentPackage, LegacyProfileResolver, LegacySketchCandidate,
};

const GO_CUBE: &[u8] = include_bytes!(
    "../../../architecture/rust-port/receipts/2026-11-14-go-cube-fixture/go-cube.obk"
);
struct GoCubeProfile;
impl LegacyProfileResolver for GoCubeProfile {
    fn resolve(
        &self,
        candidate: &LegacySketchCandidate,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String> {
        assert_eq!(sketch, DocumentSketchId(1));
        assert_eq!(index, 0);
        assert_eq!(
            candidate
                .sketches
                .get(sketch)
                .unwrap()
                .geometry()
                .lines()
                .len(),
            4
        );
        Ok(SketchPrismSelection {
            outer: (6..=9)
                .map(|id| SketchDirectedCurve {
                    curve: SketchEntityId(id),
                    reversed: false,
                })
                .collect(),
            holes: vec![],
        })
    }
}
#[test]
fn go_cube_extrusion_becomes_an_explicit_fixed_prism_candidate() {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    assert_eq!(candidate.extrusions.len(), 1);
    let feature = &candidate.extrusions[0];
    assert_eq!(feature.name, "Extrusion1");
    assert_eq!(feature.creation_sequence, 2);
    assert!(!feature.suppressed);
    assert_eq!(feature.definition.sketch(), DocumentSketchId(1));
    assert_eq!(
        feature.definition.displacement(),
        [
            DatumLength::Constant(0.0),
            DatumLength::Constant(0.0),
            DatumLength::Constant(1.0)
        ]
    );
    assert_eq!(feature.definition.selection().outer.len(), 4);
    assert!(package.restore_work_geometry().is_err());
}
#[test]
fn unrecognized_extrusion_modes_and_metadata_are_refused() {
    let source = std::str::from_utf8(GO_CUBE).unwrap();
    for modified in [
        source.replacen("operation: newBody", "operation: join", 1),
        source.replacen("direction: positive", "direction: symmetric", 1),
        source.replacen("distance: 1\n", "distance: 0\n", 1),
        source.replacen(
            "                - 0\n            operation:",
            "                - 0\n                - 1\n            operation:",
            1,
        ),
        source.replacen("distance: 1\n", "distance: 1\n            taper: 0.2\n", 1),
    ] {
        let package = DocumentPackage::decode(modified.as_bytes()).unwrap();
        assert!(
            package
                .prepare_legacy_part_candidate(&GoCubeProfile)
                .is_err()
        );
    }
}
struct RefusingProfile;
impl LegacyProfileResolver for RefusingProfile {
    fn resolve(
        &self,
        _: &LegacySketchCandidate,
        _: DocumentSketchId,
        _: usize,
    ) -> Result<SketchPrismSelection, String> {
        Err("legacy profile cannot be resolved unambiguously".into())
    }
}
#[test]
fn failed_profile_resolution_keeps_source_package_unchanged() {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let before = package.clone();
    let error = package
        .prepare_legacy_part_candidate(&RefusingProfile)
        .unwrap_err()
        .to_string();
    assert!(error.contains("unambiguously"), "{error}");
    assert_eq!(package, before);
}

struct WrongSketchBoundary;
impl LegacyProfileResolver for WrongSketchBoundary {
    fn resolve(
        &self,
        _: &LegacySketchCandidate,
        _: DocumentSketchId,
        _: usize,
    ) -> Result<SketchPrismSelection, String> {
        Ok(SketchPrismSelection {
            outer: (100..=103)
                .map(|id| SketchDirectedCurve {
                    curve: SketchEntityId(id),
                    reversed: false,
                })
                .collect(),
            holes: vec![],
        })
    }
}
#[test]
fn resolver_cannot_supply_a_boundary_from_another_sketch() {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let error = package
        .prepare_legacy_part_candidate(&WrongSketchBoundary)
        .unwrap_err()
        .to_string();
    assert!(error.contains("100"), "{error}");
}

#[test]
fn unconverted_model_sections_are_not_silently_ignored() {
    let source = std::str::from_utf8(GO_CUBE).unwrap();
    let input = source.replacen(
        "model:\n",
        "model:\n    futureFeatureData: {keep: true}\n",
        1,
    );
    let package = DocumentPackage::decode(input.as_bytes()).unwrap();
    let error = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap_err()
        .to_string();
    assert!(error.contains("futureFeatureData"), "{error}");
}

#[test]
fn go_face_dependent_plane_is_refused_without_losing_its_reference() {
    let bytes = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let package = DocumentPackage::decode(bytes).unwrap();
    let original = package.clone();
    let error = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap_err()
        .to_string();
    assert!(error.contains("workFeatures"), "unexpected error: {error}");
    assert_eq!(package, original);
    let encoded = package.encode().unwrap();
    let restored = DocumentPackage::decode(&encoded).unwrap();
    assert_eq!(restored, original);
}
