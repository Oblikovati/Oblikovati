// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;
fn sketches() -> DocumentSketches {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(2), [0.0, 0.0]).unwrap();
    geometry.add_point(SketchEntityId(4), [4.0, 0.0]).unwrap();
    geometry
        .add_line(SketchEntityId(7), [SketchEntityId(2), SketchEntityId(4)])
        .unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add(
            DocumentSketchId(3),
            "Mount: 日本語",
            SketchPlane::xz(),
            geometry,
        )
        .unwrap();
    sketches
}
fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\ndisplayName: Bracket\nmodel:\n  parameters: []\n  features: [{kind: legacy-retained}]\n  custom: {nested: [1, 2, 3]}\ndisplaySettings: {grid: true}\n").unwrap()
}
#[test]
fn block_definitions_without_sketches_survive_save_and_can_be_removed() {
    let mut sketches = DocumentSketches::default();
    sketches.define_empty_block("Rib").unwrap();
    let saved = package().with_sketches(&sketches, 0, 0).unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    assert_eq!(
        reopened
            .restore_sketches()
            .unwrap()
            .sketches
            .block_definitions()[0]
            .name(),
        "Rib"
    );
    sketches.delete_block_definition("Rib").unwrap();
    let saved = reopened.with_sketches(&sketches, 0, 0).unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    assert!(
        reopened
            .restore_sketches()
            .unwrap()
            .sketches
            .block_definitions()
            .is_empty()
    );
}
#[test]
fn captured_block_geometry_survives_yaml_round_trip_without_moving_the_source() {
    let mut sketches = sketches();
    sketches
        .capture_block_definition("Rib", DocumentSketchId(3), &[SketchEntityId(7)])
        .unwrap();
    let saved = package().with_sketches(&sketches, 9, 15).unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    let restored = reopened.restore_sketches().unwrap().sketches;
    assert!(restored.same_recipe(&sketches));
    assert_eq!(restored.block_definitions()[0].geometry().lines().len(), 1);
    assert_eq!(
        restored
            .get(DocumentSketchId(3))
            .unwrap()
            .geometry()
            .lines()
            .len(),
        1
    );
}
#[test]
fn complete_placed_recipes_and_reservations_round_trip_in_native_yaml_without_dropping_model_sections()
 {
    let original = package();
    let candidate = original.with_sketches(&sketches(), 9, 15).unwrap();
    assert_eq!(candidate.manifest().schema_version, 3);
    let encoded = candidate.encode().unwrap();
    let decoded = DocumentPackage::decode(&encoded).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert_eq!(restored.sketches, sketches());
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 10);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 16);
    let model = String::from_utf8(decoded.section_yaml("model").unwrap().unwrap()).unwrap();
    assert!(model.contains("legacy-retained"));
    assert!(model.contains("custom:"));
    assert!(model.contains("parameters: []"));
    assert_eq!(
        decoded.section_yaml("displaySettings").unwrap(),
        original.section_yaml("displaySettings").unwrap()
    );
    assert_ne!(
        decoded.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    assert_eq!(decoded.encode().unwrap(), encoded);
    assert_eq!(original.manifest().schema_version, 2);
}
#[test]
fn deleting_every_sketch_preserves_marks_and_reopening_does_not_reuse_deleted_ids() {
    let candidate = package().with_sketches(&sketches(), 9, 15).unwrap();
    let empty = DocumentSketches::default();
    let deleted = candidate.with_sketches(&empty, 9, 15).unwrap();
    let decoded = DocumentPackage::decode(&deleted.encode().unwrap()).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert!(restored.sketches.sketches().is_empty());
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 10);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 16);
    for (sketch_mark, entity_mark) in [(8, 15), (9, 14)] {
        assert!(
            deleted
                .with_sketches(&empty, sketch_mark, entity_mark)
                .unwrap_err()
                .to_string()
                .contains("retained mark")
        );
    }
}
#[test]
fn too_small_marks_and_unported_legacy_sections_are_refused_without_changing_the_package() {
    let original = package();
    for (sketch_mark, entity_mark) in [(2, 7), (3, 6)] {
        assert!(
            original
                .with_sketches(&sketches(), sketch_mark, entity_mark)
                .is_err()
        );
    }
    for legacy in ["[{id: 1, name: Sketch1}]", "{invalid: shape}", "false"] {
        let yaml = format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n  sketches: {legacy}\n");
        let package = DocumentPackage::decode(yaml.as_bytes()).unwrap();
        let encoded = package.encode().unwrap();
        assert!(
            package
                .restore_sketches()
                .unwrap_err()
                .to_string()
                .contains("legacy")
        );
        assert!(package.with_sketches(&sketches(), 3, 7).is_err());
        assert_eq!(package.encode().unwrap(), encoded);
    }
    for legacy in ["[]", "null"] {
        let yaml = format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n  sketches: {legacy}\n");
        let package = DocumentPackage::decode(yaml.as_bytes()).unwrap();
        assert!(
            package
                .restore_sketches()
                .unwrap()
                .sketches
                .sketches()
                .is_empty()
        );
    }
    assert_eq!(package(), original);
}
#[test]
fn empty_legacy_envelopes_need_no_new_section_and_bad_rust_section_shapes_are_refused() {
    let original = DocumentPackage::empty(DOCUMENT_PART, "Empty");
    let unchanged = original
        .with_sketches(&DocumentSketches::default(), 0, 0)
        .unwrap();
    assert_eq!(unchanged, original);
    for value in [
        "null",
        "[]",
        "{}",
        "{recipe: {version: 1, sketches: []}, sketchHighWater: 0, entityHighWater: 0, unknown: true}",
    ] {
        let yaml =
            format!("schemaVersion: 3\ndocumentType: 1\nmodel:\n  planarSketches: {value}\n");
        let package = DocumentPackage::decode(yaml.as_bytes()).unwrap();
        assert!(package.restore_sketches().is_err(), "{value}");
    }
    let yaml=b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  planarSketches: {recipe: {version: 1, sketches: []}, sketchHighWater: 0, entityHighWater: 0}\n";
    assert!(
        DocumentPackage::decode(yaml)
            .unwrap()
            .restore_sketches()
            .unwrap_err()
            .to_string()
            .contains("schema 3")
    );
}
#[test]
fn maximum_reservation_marks_survive_storage_and_refuse_allocator_overflow() {
    let candidate = package()
        .with_sketches(&sketches(), u64::MAX, u64::MAX)
        .unwrap();
    let mut restored = DocumentPackage::decode(&candidate.encode().unwrap())
        .unwrap()
        .restore_sketches()
        .unwrap();
    assert_eq!(restored.sketch_ids.high_water(), u64::MAX);
    assert_eq!(restored.entity_ids.high_water(), u64::MAX);
    assert!(restored.sketch_ids.allocate().is_err());
    assert!(restored.entity_ids.allocate().is_err());
}
#[test]
fn yaml_restores_exact_finite_coordinate_bits_and_custom_placement_options() {
    let mut geometry = Sketch2D::default();
    let coordinates = [
        -0.0,
        f64::from_bits(1),
        1.2345678901234567,
        f64::MAX,
        1e-200,
        1e200,
    ];
    for (index, value) in coordinates.iter().enumerate() {
        geometry
            .add_point(SketchEntityId(index as u64 + 1), [*value, *value])
            .unwrap();
    }
    let mut sketches = DocumentSketches::default();
    let options = oblikovati_model::sketch::SketchPlaneOptions {
        version: 1,
        unit_deviation: 0.01,
        angular_deviation: 0.01,
    };
    let plane =
        SketchPlane::new([1.0, 2.0, 3.0], [1.001, 0.0, 0.0], [0.0, 1.0, 0.0], options).unwrap();
    sketches
        .add(DocumentSketchId(1), "Exact", plane, geometry)
        .unwrap();
    let candidate = package().with_sketches(&sketches, 1, 6).unwrap();
    let restored = DocumentPackage::decode(&candidate.encode().unwrap())
        .unwrap()
        .restore_sketches()
        .unwrap();
    let placed = restored.sketches.get(DocumentSketchId(1)).unwrap();
    assert_eq!(placed.plane(None).unwrap().options(), options);
    for (point, expected) in placed.geometry().points().iter().zip(coordinates) {
        assert_eq!(point.position[0].to_bits(), expected.to_bits());
    }
}
#[test]
fn circular_dimensions_shared_contacts_and_retained_branches_survive_yaml_tags() {
    use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId, TangentDistanceOptions};
    let mut geometry = Sketch2D::default();
    for (id, xy) in [
        (1, [0.0, 0.0]),
        (2, [4.0, 0.0]),
        (3, [0.0, 3.0]),
        (4, [0.0, 6.0]),
    ] {
        geometry.add_point(SketchEntityId(id), xy).unwrap();
    }
    geometry
        .add_line(SketchEntityId(5), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    geometry
        .add_arc(
            SketchEntityId(6),
            SketchEntityId(3),
            [SketchEntityId(1), SketchEntityId(4)],
            false,
        )
        .unwrap();
    geometry
        .add_circle(SketchEntityId(7), SketchEntityId(3), 3.0)
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Diameter {
                curve: SketchEntityId(6),
                parameter: 91,
            },
        )
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::ArcLength {
                arc: SketchEntityId(6),
                parameter: 92,
            },
        )
        .unwrap();
    geometry
        .add_tangent_distance(
            SketchConstraintId(3),
            SketchEntityId(5),
            SketchEntityId(6),
            93,
            TangentDistanceOptions {
                far_side: true,
                linear_diameter: true,
            },
        )
        .unwrap();
    geometry
        .add_line_tangent(SketchConstraintId(4), SketchEntityId(5), SketchEntityId(6))
        .unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add(DocumentSketchId(1), "Shared", SketchPlane::xy(), geometry)
        .unwrap();
    let candidate = package().with_sketches(&sketches, 1, 7).unwrap();
    let decoded = DocumentPackage::decode(&candidate.encode().unwrap()).unwrap();
    let restored = decoded.restore_sketches().unwrap();
    assert_eq!(restored.sketches, sketches);
    assert_eq!(
        restored
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry()
            .constraints()
            .len(),
        4
    );
    assert!(
        !restored
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry()
            .arcs()[0]
            .counter_clockwise
    );
}
#[test]
fn fixed_native_schema_fixture_loads_and_invalid_recipe_versions_and_ids_are_refused() {
    let fixture = include_str!("fixtures/rust-placed-sketches-v1.yaml");
    let package = DocumentPackage::decode(fixture.as_bytes()).unwrap();
    let restored = package.restore_sketches().unwrap();
    assert_eq!(restored.sketches, sketches());
    assert_eq!(restored.sketch_ids.high_water(), 9);
    assert_eq!(restored.entity_ids.high_water(), 15);
    for broken in [
        fixture.replace("version: 1", "version: 99"),
        fixture.replace("id: 3", "id: 0"),
        fixture.replace("endpoints: [2, 4]", "endpoints: [2, 999]"),
    ] {
        let package = DocumentPackage::decode(broken.as_bytes()).unwrap();
        assert!(
            package
                .restore_sketches()
                .unwrap_err()
                .to_string()
                .contains("model.planarSketches")
        );
        assert!(
            package
                .with_sketches(&DocumentSketches::default(), 9, 15)
                .is_err()
        );
    }
}

#[path = "package_sketches/centerlines.rs"]
mod centerlines;

#[path = "package_sketches/line_constraints.rs"]
mod line_constraints;

#[test]
fn unsupported_go_3d_sketches_cannot_be_silently_omitted() {
    for section in [
        "[{id: 7, entities: [{kind: spline, handles: [{id: 9, endId: 10}]}]}]",
        "{}",
        "false",
    ] {
        let source =
            format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n  sketches3D: {section}\n");
        let package = DocumentPackage::decode(source.as_bytes()).unwrap();
        let original = package.encode().unwrap();
        assert!(
            package
                .restore_sketches()
                .unwrap_err()
                .to_string()
                .contains("sketches3D")
        );
        assert!(package.with_sketches(&sketches(), 3, 7).is_err());
        assert_eq!(package.encode().unwrap(), original);
    }
    for section in ["[]", "null"] {
        let source =
            format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n  sketches3D: {section}\n");
        let package = DocumentPackage::decode(source.as_bytes()).unwrap();
        assert!(
            package
                .restore_sketches()
                .unwrap()
                .sketches
                .sketches()
                .is_empty()
        );
    }
}

#[test]
fn merged_go_sketch_sections_cannot_bypass_the_import_guard() {
    for model in [
        "{<<: {sketches3D: [{id: 7}]}}",
        "{<<: {sketches: [{id: 1}]}}",
        "{<<: {<<: {sketches3D: [{id: 7}]}}}",
        "{<<: [{sketches3D: [{id: 7}]}, {sketches3D: []}]}",
        "{<<: 42}",
    ] {
        let source = format!("schemaVersion: 2\ndocumentType: 1\nmodel: {model}\n");
        let package = DocumentPackage::decode(source.as_bytes()).unwrap();
        assert!(package.restore_sketches().is_err(), "{model}");
        assert!(package.with_sketches(&sketches(), 3, 7).is_err(), "{model}");
    }
    let native = b"schemaVersion: 3\ndocumentType: 1\nmodel: {<<: {planarSketches: {recipe: {version: 1, sketches: []}, sketchHighWater: 0, entityHighWater: 0}}}\n";
    let package = DocumentPackage::decode(native).unwrap();
    assert!(
        package
            .restore_sketches()
            .unwrap_err()
            .to_string()
            .contains("model.<<")
    );
    let source = b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  <<: &defaults {sketches3D: [{id: 7}]}\n  sketches3D: []\n";
    let package = DocumentPackage::decode(source).unwrap();
    assert!(
        package
            .restore_sketches()
            .unwrap_err()
            .to_string()
            .contains("model.<<")
    );
}

#[test]
fn origin_plane_identity_round_trips_and_old_fixed_frames_stay_fixed() {
    use oblikovati_model::{
        document_sketch::SketchPlacement, feature::work_plane::WorkPlaneOrigin,
    };
    let mut owner = DocumentSketches::default();
    owner
        .add_on_origin_plane(
            DocumentSketchId(1),
            "XY",
            WorkPlaneOrigin::XY,
            Sketch2D::default(),
        )
        .unwrap();
    owner
        .add(
            DocumentSketchId(2),
            "Legacy",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
        .unwrap();
    let saved = package().with_sketches(&owner, 2, 0).unwrap();
    let bytes = saved.encode().unwrap();
    assert!(String::from_utf8_lossy(&bytes).contains("schemaVersion: 3"));
    let reopened = DocumentPackage::decode(&bytes)
        .unwrap()
        .restore_sketches()
        .unwrap()
        .sketches;
    assert_eq!(
        reopened.get(DocumentSketchId(1)).unwrap().placement(),
        SketchPlacement::OriginPlane(WorkPlaneOrigin::XY)
    );
    assert!(matches!(
        reopened.get(DocumentSketchId(2)).unwrap().placement(),
        SketchPlacement::Fixed(_)
    ));
    let mut ambiguous = owner.recipe();
    ambiguous.sketches[0].plane = Some(SketchPlane::xy().recipe());
    ambiguous.sketches[0].plane_options = Some(SketchPlane::xy().options());
    assert!(DocumentSketches::from_recipe(&ambiguous).is_err());
    let mut old = owner.recipe();
    old.version = 12;
    assert!(DocumentSketches::from_recipe(&old).is_err());
    let yaml = serde_yaml_ng::to_string(&owner.recipe()).unwrap();
    let unknown = yaml.replace("plane_origin: XY", "plane_origin: FuturePlane");
    assert_ne!(unknown, yaml);
    assert!(
        serde_yaml_ng::from_str::<oblikovati_model::document_sketch::DocumentSketchRecipe>(
            &unknown
        )
        .is_err()
    );
}

#[path = "package_sketches/spatial.rs"]
mod spatial;

#[path = "package_sketches/constraint_glyphs.rs"]
mod constraint_glyphs;
