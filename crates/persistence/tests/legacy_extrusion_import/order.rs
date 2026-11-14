// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn unstamped_standalone_legacy_sketch_is_retained_without_inventing_order() {
    let source = std::str::from_utf8(GO_CUBE).unwrap();
    let sketches_only = source.split("    features:\n").next().unwrap();
    for stamp in ["", "          seq: 0\n"] {
        let source = sketches_only.replace("          seq: 1\n", stamp);
        let package = DocumentPackage::decode(source.as_bytes()).unwrap();
        let candidate = package
            .prepare_legacy_part_candidate(&GoCubeProfile)
            .unwrap();
        assert_eq!(
            candidate.sketches.creation_sequences,
            [(DocumentSketchId(1), 0)]
        );
        assert!(candidate.extrusions.is_empty());
    }
}

#[test]
fn extrusion_requires_unique_stamps_and_a_prior_source_sketch() {
    let source = std::str::from_utf8(GO_CUBE).unwrap();
    for (name, modified) in [
        (
            "duplicate",
            source.replacen("          seq: 2\n", "          seq: 1\n", 1),
        ),
        (
            "missing sketch",
            source.replacen("          seq: 1\n", "", 1),
        ),
        (
            "missing extrusion",
            source.replacen("          seq: 2\n", "", 1),
        ),
        (
            "forward sketch",
            source.replacen("          seq: 1\n", "          seq: 3\n", 1),
        ),
        (
            "both missing",
            source
                .replace("          seq: 1\n", "")
                .replace("          seq: 2\n", ""),
        ),
    ] {
        let package = DocumentPackage::decode(modified.as_bytes()).unwrap();
        let before = package.clone();
        let error = package
            .prepare_legacy_part_candidate(&GoCubeProfile)
            .unwrap_err()
            .to_string();
        assert!(error.contains("sequence"), "{name}: {error}");
        assert_eq!(package, before, "{name}");
    }
}

#[test]
fn duplicate_sketch_stamps_are_rejected_even_without_features() {
    let source = b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  sketches:\n    - {id: 1, name: A, seq: 7, plane: {origin: [0, 0, 0], xAxis: [1, 0, 0], yAxis: [0, 1, 0]}}\n    - {id: 2, name: B, seq: 7, plane: {origin: [0, 0, 0], xAxis: [1, 0, 0], yAxis: [0, 1, 0]}}\n";
    let package = DocumentPackage::decode(source).unwrap();
    let before = package.clone();
    let error = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap_err()
        .to_string();
    assert!(error.contains("sequence 7"), "{error}");
    assert_eq!(package, before);
}

#[test]
fn missing_sketch_reference_declines_before_profile_resolution() {
    let source = std::str::from_utf8(GO_CUBE).unwrap().replacen(
        "            sketch: 0\n",
        "            sketch: 8\n",
        1,
    );
    let package = DocumentPackage::decode(source.as_bytes()).unwrap();
    let before = package.clone();
    let error = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap_err()
        .to_string();
    assert!(error.contains("sketch index 8"), "{error}");
    assert_eq!(package, before);
}

#[test]
fn duplicate_extrusion_stamps_are_rejected_before_resolving_profiles() {
    let source = std::str::from_utf8(GO_CUBE).unwrap();
    let feature = source.split_once("    features:\n").unwrap().1;
    let modified = format!(
        "{source}{}",
        feature.replace("name: Extrusion1", "name: Extrusion2")
    );
    let package = DocumentPackage::decode(modified.as_bytes()).unwrap();
    let before = package.clone();
    let error = package
        .prepare_legacy_part_candidate(&RefusingProfile)
        .unwrap_err()
        .to_string();
    assert!(error.contains("sequence 2"), "{error}");
    assert_eq!(package, before);
}
