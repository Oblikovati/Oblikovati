// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn sketch_scene_resolves_healthy_linked_text_and_names_missing_parameters() {
    use oblikovati_api::types::MODEL_PARAM;
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    snapshot
        .parameters
        .insert_expression(42, "d0", MODEL_PARAM, "10 mm")
        .unwrap();
    let sketch = DocumentSketchId(1);
    let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
    add_scene_link(&mut geometry, 9010, 42);
    add_scene_link(&mut geometry, 9020, 99);
    snapshot
        .sketches
        .replace_geometry(sketch, geometry.clone())
        .unwrap();
    let mut update = HeadSketchSceneUpdate {
        source: Some((1, 0)),
        sketches: vec![],
        plane_declines: vec![],
        block_declines: vec![],
    };
    collect_sketches(&snapshot, "Part.opd", &[], None, &mut update);
    let [packet] = update.sketches.as_slice() else {
        panic!("expected one visible sketch");
    };
    let [healthy, failed] = packet.geometry.texts() else {
        panic!("expected two text entities");
    };
    assert_eq!(healthy.style.content, "10.000\nPart.opd");
    assert!(failed.style.content.is_empty());
    assert_eq!(packet.text_declines[0].entity, failed.id);
    assert!(packet.text_declines[0].cause.contains("99"));
    assert_eq!(snapshot.sketches.get(sketch).unwrap().geometry(), &geometry);
}

fn add_scene_link(geometry: &mut oblikovati_model::sketch::Sketch2D, entity: u64, parameter: u64) {
    use oblikovati_api::types::{
        SKETCH_TEXT_FIELD_FILE_NAME, SKETCH_TEXT_FIELD_MODEL_PARAMETER,
        SKETCH_TEXT_FIELD_PLACEHOLDER, SketchTextField, SketchTextStyle,
    };
    use oblikovati_model::sketch::{SketchEntityId, SketchText2D, SketchTextParameterBinding};
    let anchor = SketchEntityId(entity + 1);
    geometry.add_defining_point(anchor, [0., 0.]).unwrap();
    geometry
        .add_text(SketchText2D {
            id: SketchEntityId(entity),
            anchor,
            style: SketchTextStyle {
                content: format!(
                    "{SKETCH_TEXT_FIELD_PLACEHOLDER}\n{SKETCH_TEXT_FIELD_PLACEHOLDER}"
                ),
                height: 0.305,
                linked_fields: Some(vec![
                    SketchTextField {
                        at: 0,
                        kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
                        parameter: "d0".into(),
                        ..Default::default()
                    },
                    SketchTextField {
                        at: 2,
                        kind: SKETCH_TEXT_FIELD_FILE_NAME,
                        ..Default::default()
                    },
                ]),
                ..Default::default()
            },
            parameter_bindings: vec![SketchTextParameterBinding {
                at: 0,
                parameter_id: parameter,
            }],
        })
        .unwrap();
}
#[test]
fn invalid_plane_is_named_without_removing_other_sketches() {
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    snapshot
        .sketches
        .add_on_work_plane(
            DocumentSketchId(2),
            "Missing plane",
            oblikovati_model::feature::FeatureId(999),
            Default::default(),
        )
        .unwrap();
    let mut update = HeadSketchSceneUpdate {
        source: Some((1, 0)),
        sketches: vec![],
        plane_declines: vec![],
        block_declines: vec![],
    };
    collect_sketches(&snapshot, "Part.opd", &[], None, &mut update);
    assert_eq!(update.sketches.len(), 1);
    assert_eq!(update.plane_declines.len(), 1);
    assert_eq!(update.plane_declines[0].0, DocumentSketchId(2));
    assert!(update.plane_declines[0].1.contains("999"));
}

#[test]
fn hidden_sketches_leave_the_scene_before_plane_resolution() {
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    snapshot
        .sketches
        .add_on_work_plane(
            DocumentSketchId(2),
            "Missing plane",
            oblikovati_model::feature::FeatureId(999),
            Default::default(),
        )
        .unwrap();
    for id in [DocumentSketchId(1), DocumentSketchId(2)] {
        snapshot.sketches.set_visible(id, false).unwrap();
    }
    let mut update = HeadSketchSceneUpdate {
        source: Some((1, 0)),
        sketches: vec![],
        plane_declines: vec![],
        block_declines: vec![],
    };
    collect_sketches(&snapshot, "Part.opd", &[], None, &mut update);
    assert!(update.sketches.is_empty());
    assert!(update.plane_declines.is_empty());
    snapshot
        .sketches
        .set_visible(DocumentSketchId(1), true)
        .unwrap();
    collect_sketches(&snapshot, "Part.opd", &[], None, &mut update);
    assert_eq!(update.sketches.len(), 1);
}
#[test]
fn a_hidden_sketch_shows_only_while_it_is_called_up() {
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    let sketch = crate::scene_fixture::SKETCH;
    snapshot.sketches.set_visible(sketch, false).unwrap();
    let hidden = snapshot.sketches.get(sketch).unwrap();
    assert!(!crate::sketch_scene::sketch_shows(hidden, &[]));
    assert!(crate::sketch_scene::sketch_shows(hidden, &[sketch]));
}

#[test]
fn block_instance_is_sent_to_the_sketch_scene() {
    use oblikovati_model::{document_sketch::SketchBlockInstance, sketch::SketchEntityId};
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    let line = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .lines()[0]
        .id;
    snapshot
        .sketches
        .capture_block_definition("Rib", DocumentSketchId(1), &[line])
        .unwrap();
    let instance = SketchBlockInstance::new(SketchEntityId(99), "Rib", [0., 0.], 0., 1.).unwrap();
    snapshot
        .sketches
        .insert_block_instance(DocumentSketchId(1), instance)
        .unwrap();
    let mut update = HeadSketchSceneUpdate {
        source: Some((1, 0)),
        sketches: vec![],
        plane_declines: vec![],
        block_declines: vec![],
    };
    collect_sketches(&snapshot, "Part.opd", &[], None, &mut update);
    assert_eq!(update.sketches.len(), 1);
    assert!(update.block_declines.is_empty());
    assert_eq!(update.sketches[0].blocks.len(), 1);
    assert_eq!(update.sketches[0].blocks[0].geometry.lines().len(), 1);
    assert_eq!(
        update.sketches[0].blocks[0].instance.id(),
        SketchEntityId(99)
    );
}
