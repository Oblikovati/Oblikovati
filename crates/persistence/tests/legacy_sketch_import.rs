// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_persistence::package::DocumentPackage;

const GO_CUBE: &[u8] = include_bytes!(
    "../../../architecture/rust-port/receipts/2026-11-14-go-cube-fixture/go-cube.obk"
);
fn normalized_text(bytes: &[u8]) -> String {
    std::str::from_utf8(bytes).unwrap().replace("\r\n", "\n")
}

#[test]
fn go_center_point_roles_and_ownership_survive_import() {
    use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
    let source = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-16-go-center-points-3882/center-points.obk"
    );
    let observations = include_str!(
        "../../../architecture/rust-port/receipts/2026-11-16-go-center-points-3882/observations.json"
    );
    let rows: serde_yaml_ng::Value = serde_yaml_ng::from_str(observations).unwrap();
    let package = DocumentPackage::decode(source).unwrap();
    let candidate = package.prepare_legacy_sketch_candidate().unwrap();
    let sketch = candidate
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let restored = Sketch2D::from_recipe(&sketch.recipe()).unwrap();
    for row in rows.as_sequence().unwrap() {
        let id = SketchEntityId(row["id"].as_u64().unwrap());
        assert_eq!(
            restored.is_center_point(id),
            row["center"].as_bool().unwrap()
        );
        assert_eq!(
            restored.read_index().entity_ids().contains(&id),
            row["standalone"].as_bool().unwrap()
        );
    }
}

#[test]
fn invalid_center_point_marker_leaves_source_unchanged() {
    let source = include_str!(
        "../../../architecture/rust-port/receipts/2026-11-16-go-center-points-3882/center-points.obk"
    );
    let invalid = source.replacen("centerPoint: true", "centerPoint: unknown", 1);
    let package = DocumentPackage::decode(invalid.as_bytes()).unwrap();
    let original = package.clone();
    assert!(package.prepare_legacy_sketch_candidate().is_err());
    assert_eq!(package, original);
}

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
fn legacy_line_roles_survive_import_and_recipe_round_trip() {
    use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
    for (construction, centerline) in [(false, false), (true, false), (false, true), (true, true)] {
        let fields = format!(
            "kind: line\n              construction: {construction}\n              centerline: {centerline}"
        );
        let input = std::str::from_utf8(GO_CUBE)
            .unwrap()
            .replacen("kind: line", &fields, 1);
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        let candidate = package.prepare_legacy_sketch_candidate().unwrap();
        let geometry = candidate
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry();
        let mut restored = Sketch2D::from_recipe(&geometry.recipe()).unwrap();
        let line = SketchEntityId(6);
        assert_eq!(restored.is_centerline(line).unwrap(), centerline);
        assert_eq!(
            restored.is_construction(line).unwrap(),
            construction || centerline
        );
        assert_eq!(
            restored.profile_curve_ids().contains(&line),
            !(construction || centerline)
        );
        restored.set_line_centerline(line, false).unwrap();
        assert_eq!(restored.is_construction(line).unwrap(), construction);
    }
}

#[test]
fn go_saved_line_roles_match_reopened_go_observations() {
    use oblikovati_model::sketch::SketchEntityId;
    let source = include_bytes!(
        "../../../architecture/rust-port/receipts/2026-11-16-go-line-roles-3882/line-roles.obk"
    );
    let observations = include_str!(
        "../../../architecture/rust-port/receipts/2026-11-16-go-line-roles-3882/observations.json"
    );
    let rows: serde_yaml_ng::Value = serde_yaml_ng::from_str(observations).unwrap();
    let package = DocumentPackage::decode(source).unwrap();
    let candidate = package.prepare_legacy_sketch_candidate().unwrap();
    let sketch = candidate
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    for row in rows.as_sequence().unwrap() {
        let id = SketchEntityId(row["id"].as_u64().unwrap());
        let centerline = row["centerline"].as_bool().unwrap();
        let construction = row["effectiveConstruction"].as_bool().unwrap();
        assert_eq!(sketch.is_centerline(id).unwrap(), centerline);
        assert_eq!(sketch.is_construction(id).unwrap(), construction);
        let mut copy = sketch.clone();
        copy.set_line_centerline(id, false).unwrap();
        assert_eq!(
            copy.is_construction(id).unwrap(),
            row["explicitConstruction"].as_bool().unwrap()
        );
    }
}

#[test]
fn malformed_legacy_line_role_is_refused_without_changing_package() {
    let input = normalized_text(GO_CUBE).replacen(
        "kind: line",
        "kind: line\n              construction: not-a-boolean",
        1,
    );
    let package = DocumentPackage::decode(input.as_bytes()).unwrap();
    let original = package.clone();
    assert!(package.prepare_legacy_sketch_candidate().is_err());
    assert_eq!(package, original);
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
    let input = normalized_text(GO_CUBE).replacen(
        "expression: size\n    features:",
        "expression: missing\n    features:",
        1,
    );
    let package = DocumentPackage::decode(input.as_bytes()).unwrap();
    assert!(package.prepare_legacy_sketch_candidate().is_err());
}

#[test]
fn unrecognized_sketch_metadata_is_not_discarded() {
    let input = normalized_text(GO_CUBE).replacen(
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
    let source = normalized_text(GO_CUBE);
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
    let source = normalized_text(GO_CUBE);
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
