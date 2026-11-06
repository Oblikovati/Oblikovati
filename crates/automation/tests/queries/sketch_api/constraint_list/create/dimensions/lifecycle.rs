// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::file_discovery::sketch_api::entities;
use oblikovati_api::client::Caller;
#[test]
fn empty_list_is_an_array_and_structural_origin_sources_are_not_dimensions() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
    let before = owner_revision(&files, document);
    let bytes = FileQueryHost::new(&files)
        .call(METHOD_SKETCH_DIMENSIONS, b"{}")
        .unwrap();
    assert_eq!(
        serde_json::from_slice::<serde_json::Value>(&bytes).unwrap(),
        serde_json::json!({"dimensions":[]})
    );
    assert_eq!(owner_revision(&files, document), before);
    assert!(FILE_QUERY_METHODS.contains(&METHOD_SKETCH_DIMENSIONS));
}
#[test]
fn malformed_request_wrong_kind_and_indices_do_not_mutate_the_owner() {
    let (files, document, _) = populated(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let client = Client::new(FileQueryHost::new(&files));
    for sketch_index in [-1, 1, i64::MAX] {
        expect_api_decline(
            client.sketch().dimensions(&SketchArgs { sketch_index }),
            "ordered index",
        );
    }
    assert_malformed_query(&files);
    assert_eq!(owner_revision(&files, document), before);
    let (drawing, _, _) = configured_files(DOCUMENT_DRAWING, Default::default());
    expect_api_decline(
        Client::new(FileQueryHost::new(&drawing))
            .sketch()
            .dimensions(&Default::default()),
        "part or assembly",
    );
}
#[test]
fn updated_expression_is_current_but_measurement_does_not_turn_into_its_target() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    let original = dimensions(&files);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&numeric("width", "2 cm"))
        .unwrap();
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    let actual = dimensions(&files);
    assert_eq!(actual[1].value, original[1].value);
    assert_eq!(actual[1].value, 4.0);
    assert_eq!(actual[1].expression, "2 cm");
    assert_eq!(owner_revision(&files, document), before);
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
}
#[test]
fn length_values_are_centimetres_and_angles_remain_radians_at_every_working_scale() {
    for scale in [1e-6, 1.0, 1e6] {
        assert_scaled(scale);
    }
}
fn assert_scaled(scale: f64) {
    let files = populated_scaled(scale);
    let rows = dimensions(&files);
    for (row, (_, value, _, _)) in rows.iter().zip(expected_rows()) {
        let expected = if row.kind == "angle" || row.kind == "threePointAngle" {
            value
        } else {
            value * scale
        };
        let tolerance = SolveOptions::default().relative_tolerance * expected.abs().max(1.0);
        assert!(
            (row.value - expected).abs() <= tolerance,
            "{row:?}, expected {expected}"
        );
    }
}
#[test]
fn failed_and_incompatible_parameter_sources_are_named_declines_without_stale_values() {
    for (source, expected) in [("missing", "width"), ("90 deg", "Length")] {
        let (mut files, document, mut guids) = populated(DOCUMENT_PART);
        Client::new(FileHost::new(&mut files, &mut guids, "test"))
            .parameters()
            .set(&numeric("width", source))
            .unwrap();
        let before = owner_revision(&files, document);
        expect_api_decline(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .dimensions(&Default::default()),
            expected,
        );
        assert_eq!(owner_revision(&files, document), before);
    }
}
#[test]
fn missing_backing_parameter_is_reported_by_id() {
    let (mut files, document, _) = fresh(DOCUMENT_PART);
    files
        .commands(document)
        .unwrap()
        .run("Broken dimension", install_missing_source)
        .unwrap();
    expect_api_decline(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .dimensions(&Default::default()),
        "parameter 99",
    );
}
#[test]
fn history_reopen_and_copy_preserve_dimension_order_sources_and_measurements() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    let before = dimensions(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert!(dimensions(&files).is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(dimensions(&files), before);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(dimensions(&files), before);
    entities::activate_copy(
        &mut files,
        reopened,
        &mut source,
        Path::new("/fixtures/dimension-copy.obk"),
    );
    assert_eq!(dimensions(&files), before);
}

fn assert_malformed_query(files: &DiscoveryWorkspace) {
    for input in [b"{".as_slice(), b"[]", br#"{"sketchIndex":"bad"}"#] {
        expect_api_decline(
            FileQueryHost::new(files).call(METHOD_SKETCH_DIMENSIONS, input),
            METHOD_SKETCH_DIMENSIONS,
        );
    }
}

pub(super) fn populated_scaled(scale: f64) -> DiscoveryWorkspace {
    let (mut files, document, _) = entities::configured_scaled(scale);
    files
        .commands(document)
        .unwrap()
        .run("Scaled operands", install_operands)
        .unwrap();
    files
        .commands(document)
        .unwrap()
        .run("Scaled dimensions", install_dimensions)
        .unwrap();
    files
}

fn install_missing_source(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let sketch = DocumentSketchId(1);
    let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
    geometry
        .add_constraint(
            ids.allocate_sketch_constraint(sketch).unwrap(),
            Radius {
                curve: entity(9),
                parameter: 99,
            },
        )
        .unwrap();
    snapshot
        .sketches
        .replace_geometry(sketch, geometry)
        .map_err(|cause| cause.to_string())
}
