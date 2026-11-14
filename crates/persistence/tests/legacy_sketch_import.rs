// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_persistence::package::DocumentPackage;

const GO_CUBE: &[u8] = include_bytes!(
    "../../../architecture/rust-port/receipts/2026-11-14-go-cube-fixture/go-cube.obk"
);

#[test]
fn go_cube_sketch_import_preserves_dimensions_and_is_fully_constrained() {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let imported = package.prepare_legacy_sketch_candidate().unwrap();
    let geometry = imported
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(geometry.points().len(), 4);
    assert_eq!(geometry.lines().len(), 4);
    assert_eq!(geometry.constraints().len(), 7);
    assert_eq!(
        geometry
            .analyze(&imported.parameters.parameters, 1e-9)
            .unwrap()
            .degrees_of_freedom,
        0
    );
    assert_eq!(imported.parameters.parameters.find("size"), Some(1));
    assert!(package.restore_sketches().is_err());
}

#[test]
fn unsupported_legacy_entity_is_refused_without_mutating_the_package() {
    let input =
        std::str::from_utf8(GO_CUBE)
            .unwrap()
            .replacen("kind: line", "kind: future-curve", 1);
    let package = DocumentPackage::decode(input.as_bytes()).unwrap();
    let original = package.clone();
    let error = package
        .prepare_legacy_sketch_candidate()
        .unwrap_err()
        .to_string();
    assert!(error.contains("future-curve"), "{error}");
    assert_eq!(package, original);
}

#[test]
fn imported_dimension_parameters_follow_the_saved_size_expression() {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let mut imported = package.prepare_legacy_sketch_candidate().unwrap();
    let size = imported.parameters.parameters.find("size").unwrap();
    let first = imported.parameters.parameters.find("d2").unwrap();
    let second = imported.parameters.parameters.find("d3").unwrap();
    imported
        .parameters
        .parameters
        .set_expression(size, "20 mm")
        .unwrap();
    assert_eq!(
        imported
            .parameters
            .parameters
            .get(first)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert_eq!(
        imported
            .parameters
            .parameters
            .get(second)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert_eq!(imported.creation_sequences, vec![(DocumentSketchId(1), 1)]);
}

#[test]
fn missing_dimension_source_is_declined() {
    let input = std::str::from_utf8(GO_CUBE).unwrap().replacen(
        "expression: size\n    features:",
        "expression: missing\n    features:",
        1,
    );
    let package = DocumentPackage::decode(input.as_bytes()).unwrap();
    assert!(package.prepare_legacy_sketch_candidate().is_err());
}

#[test]
fn unrecognized_sketch_metadata_is_not_discarded() {
    let input = std::str::from_utf8(GO_CUBE).unwrap().replacen(
        "          seq: 1\n",
        "          seq: 1\n          futureField: keep-me\n",
        1,
    );
    let package = DocumentPackage::decode(input.as_bytes()).unwrap();
    let error = package
        .prepare_legacy_sketch_candidate()
        .unwrap_err()
        .to_string();
    assert!(error.contains("futureField"), "{error}");
}

#[test]
fn invalid_line_operands_and_wrong_dimension_units_are_declined() {
    let source = std::str::from_utf8(GO_CUBE).unwrap();
    let missing_point = source.replacen("- 3\n            - id: 7", "- 99\n            - id: 7", 1);
    let wrong_unit = source.replacen(
        "expression: size\n    features:",
        "expression: 1 deg\n    features:",
        1,
    );
    for input in [missing_point, wrong_unit] {
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        assert!(package.prepare_legacy_sketch_candidate().is_err());
    }
}

#[test]
fn unknown_constraint_and_mixed_native_section_are_refused() {
    let source = std::str::from_utf8(GO_CUBE).unwrap();
    let unknown = source.replacen("kind: horizontal", "kind: future-relation", 1);
    let mixed = source.replacen(
        "    sketches:\n",
        "    planarSketches: {}\n    sketches:\n",
        1,
    );
    for input in [unknown, mixed] {
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        assert!(package.prepare_legacy_sketch_candidate().is_err());
    }
}

#[test]
fn go_sketch_curve_order_survives_candidate_and_native_recipe_round_trip() {
    use oblikovati_model::sketch::Sketch2D;
    let reordered = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-14-go-face-lineage/reordered.obk"
    );
    for (bytes, expected) in [
        (GO_CUBE, [6, 7, 8, 9]),
        (reordered.as_slice(), [7, 8, 9, 6]),
    ] {
        let package = DocumentPackage::decode(bytes).unwrap();
        let candidate = package.prepare_legacy_sketch_candidate().unwrap();
        let sketch = candidate
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry();
        let restored = Sketch2D::from_recipe(&sketch.recipe()).unwrap();
        for geometry in [sketch, &restored] {
            let ids: Vec<_> = geometry.lines().iter().map(|line| line.id.0).collect();
            assert_eq!(ids, expected);
        }
    }
}
