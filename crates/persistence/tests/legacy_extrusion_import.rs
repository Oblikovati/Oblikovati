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
fn go_face_dependent_plane_is_prepared_without_mutating_the_package() {
    let bytes = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let package = DocumentPackage::decode(bytes).unwrap();
    let original = package.clone();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    assert_eq!(candidate.face_offsets.len(), 1);
    assert_eq!(candidate.face_offsets[0].creation_sequence, 3);
    assert_eq!(candidate.face_offsets[0].face_key, b"\x03Extrusion1:side#1");
    assert_eq!(candidate.face_offsets[0].offset, 0.5);
    assert!(package.restore_work_geometry().is_err());
    assert_eq!(package, original);
    let restored = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    assert_eq!(restored, original);
}

#[test]
fn go_face_offset_is_prepared_with_its_original_reference_and_sequence() {
    let bytes = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let package = DocumentPackage::decode(bytes).unwrap();
    let candidate = package.prepare_legacy_face_offsets().unwrap();
    assert_eq!(candidate.len(), 1);
    assert_eq!(candidate[0].creation_sequence, 3);
    assert_eq!(candidate[0].face_key, b"\x03Extrusion1:side#1");
    assert_eq!(candidate[0].offset, 0.5);
    assert!(package.restore_work_geometry().is_err());
}

#[test]
fn go_raw_url_face_keys_are_decoded_without_loss() {
    // Generated independently with Go base64.RawURLEncoding.EncodeToString (#3935).
    for (encoded, expected) in [
        ("-w", &[0xfb][..]),
        ("__8", &[0xff, 0xff][..]),
        ("-_8A", &[0xfb, 0xff, 0x00][..]),
        ("APv_EA", &[0x00, 0xfb, 0xff, 0x10][..]),
        ("_-7dzLs", &[0xff, 0xee, 0xdd, 0xcc, 0xbb][..]),
    ] {
        let package = face_offset_with_reference(&format!("'face/{encoded}'"));
        assert_eq!(
            package.prepare_legacy_face_offsets().unwrap()[0].face_key,
            expected,
            "reference {encoded:?}"
        );
    }
    // Go DecodeString also accepts nonzero unused bits in a raw encoding.
    let package = face_offset_with_reference("'face/_x'");
    assert_eq!(
        package.prepare_legacy_face_offsets().unwrap()[0].face_key,
        [0xff]
    );
    let package = face_offset_with_reference("\"face/_\\r\\nw\"");
    assert_eq!(
        package.prepare_legacy_face_offsets().unwrap()[0].face_key,
        [0xff]
    );
}

#[test]
fn go_raw_url_face_keys_reject_padding_standard_alphabet_and_bad_lengths() {
    for encoded in ["_w==", "AQ==", "+w", "+w==", "/w", "A", "_w=", "_ w"] {
        let package = face_offset_with_reference(&format!("'face/{encoded}'"));
        let error = package
            .prepare_legacy_face_offsets()
            .unwrap_err()
            .to_string();
        assert!(error.contains(encoded), "reference {encoded:?}: {error}");
    }
}

fn face_offset_with_reference(reference: &str) -> DocumentPackage {
    let source = format!(
        "schemaVersion: 2\ndocumentType: 1\nmodel:\n  workFeatures:\n    - {{collection: plane, kind: plane-offset, seq: 1, refs: [{reference}]}}\n"
    );
    DocumentPackage::decode(source.as_bytes()).unwrap()
}

#[test]
fn go_face_offset_rejects_unsupported_values_without_mutation() {
    let bytes = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let source = std::str::from_utf8(bytes).unwrap();
    for modified in [
        source.replacen("kind: plane-offset", "kind: plane-angle", 1),
        source.replacen("seq: 3", "seq: 0", 1),
        source.replacen("face/A0V4dHJ1c2lvbjE6c2lkZSMx", "plane/1", 1),
        source.replacen("face/A0V4dHJ1c2lvbjE6c2lkZSMx", "face/", 1),
        source.replacen("offset: 0.5", "offset: .inf", 1),
        source.replacen("face/A0V4dHJ1c2lvbjE6c2lkZSMx", "face/!!!", 1),
        source.replacen("offset: 0.5", "offset: 0.5\n          flipped: true", 1),
    ] {
        let package = DocumentPackage::decode(modified.as_bytes()).unwrap();
        let before = package.clone();
        assert!(package.prepare_legacy_face_offsets().is_err());
        assert_eq!(package, before);
    }
}

#[test]
fn face_offset_sequences_are_unique_and_nonzero() {
    let package = DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  workFeatures:\n    - {collection: plane, kind: plane-offset, seq: 3, refs: [face/AQ]}\n    - {collection: plane, kind: plane-offset, seq: 3, refs: [face/Ag]}\n",
    )
    .unwrap();
    let error = package.prepare_legacy_face_offsets().unwrap_err();
    assert!(error.to_string().contains("unique nonzero sequence"));
}

#[test]
fn absent_work_features_produce_no_face_offset_candidates() {
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    assert!(package.prepare_legacy_face_offsets().unwrap().is_empty());
}

#[test]
fn face_offset_cannot_share_a_creation_sequence_with_other_nodes() {
    let bytes = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let source = std::str::from_utf8(bytes).unwrap();
    for sequence in [1, 2] {
        let invalid = source.replacen("seq: 3", &format!("seq: {sequence}"), 1);
        let package = DocumentPackage::decode(invalid.as_bytes()).unwrap();
        let error = package
            .prepare_legacy_part_candidate(&GoCubeProfile)
            .unwrap_err();
        assert!(
            error
                .to_string()
                .contains(&format!("creation sequence {sequence}")),
            "{error}"
        );
    }
}

#[test]
fn face_key_identifies_the_go_extrusion_and_side_ordinal() {
    use oblikovati_persistence::package::LegacyExtrusionFaceRole;
    let bytes = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let package = DocumentPackage::decode(bytes).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let key = &candidate.face_offsets[0].face_key;
    let producer = candidate.extrusion_face_producer(key).unwrap();
    assert_eq!(producer.creation_sequence, 2);
    assert_eq!(producer.role, LegacyExtrusionFaceRole::Side(1));
}

#[test]
fn face_key_rejects_malformed_wrong_kind_and_unknown_producers() {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    for key in [
        &b"\x02Extrusion1:side#1"[..],
        &b"\x03Other:side#1"[..],
        &b"\x03Extrusion1:side#01"[..],
        &b"\x03Extrusion1:side#-1"[..],
        &b"\x03Extrusion1:start-cap#1"[..],
        &b"\x03Extrusion1:side#1/extra"[..],
    ] {
        assert!(candidate.extrusion_face_producer(key).is_err(), "{key:?}");
    }
}

#[test]
fn face_key_recognizes_go_caps_and_rejects_duplicate_producers() {
    use oblikovati_persistence::package::{LegacyExtrusionCandidate, LegacyExtrusionFaceRole};
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let mut candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    for (key, role) in [
        (
            &b"\x03Extrusion1:start-cap#0"[..],
            LegacyExtrusionFaceRole::StartCap,
        ),
        (
            &b"\x03Extrusion1:end-cap#0"[..],
            LegacyExtrusionFaceRole::EndCap,
        ),
    ] {
        assert_eq!(candidate.extrusion_face_producer(key).unwrap().role, role);
    }
    let first = &candidate.extrusions[0];
    candidate.extrusions.push(LegacyExtrusionCandidate {
        name: first.name.clone(),
        creation_sequence: 5,
        suppressed: false,
        definition: first.definition.clone(),
    });
    let error = candidate
        .extrusion_face_producer(b"\x03Extrusion1:side#1")
        .unwrap_err();
    assert!(error.to_string().contains("unambiguous"), "{error}");
}

#[test]
fn face_offset_rejects_a_missing_or_later_extrusion_producer() {
    let bytes = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let source = std::str::from_utf8(bytes).unwrap();
    for invalid in [
        source.replacen("name: Extrusion1", "name: Other", 1),
        source.replacen(
            "name: Extrusion1\n          seq: 2",
            "name: Extrusion1\n          seq: 4",
            1,
        ),
    ] {
        let package = DocumentPackage::decode(invalid.as_bytes()).unwrap();
        assert!(
            package
                .prepare_legacy_part_candidate(&GoCubeProfile)
                .is_err()
        );
    }
}

#[path = "legacy_extrusion_import/order.rs"]
mod creation_order;
