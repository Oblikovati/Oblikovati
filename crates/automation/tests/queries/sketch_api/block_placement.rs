// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn setup(scale: f64) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, mut source) = entities::configured_scaled(scale);
    insert_circle(&mut files, document);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .create_block_definition(&CreateBlockDefinitionArgs {
            name: "Rib".into(),
            ..Default::default()
        })
        .unwrap();
    (files, document, source)
}
fn add(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    args: &AddSketchBlockArgs,
) -> Result<AddSketchBlockResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .add_block_instance(args)
}
fn instances(files: &DiscoveryWorkspace) -> Vec<SketchBlockInfo> {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .block_instances(&SketchArgs::default())
        .unwrap()
        .instances
        .unwrap()
}
#[test]
fn placement_uses_centimetres_angles_and_default_scale_across_working_scales() {
    for scale in [0.1, 2.0] {
        let (mut files, document, mut source) = setup(scale);
        let request = AddSketchBlockArgs {
            definition: "Rib".into(),
            position: Some(vec![2., 3.]),
            rotation_angle: "90 deg".into(),
            ..Default::default()
        };
        let result = add(&mut files, &mut source, &request).unwrap();
        assert_eq!(result.entity_id, 3);
        let rows = instances(&files);
        assert_eq!(rows[0].position, Some(vec![2., 3.]));
        assert!((rows[0].rotation - std::f64::consts::FRAC_PI_2).abs() < 1e-12);
        assert_eq!(rows[0].scale, 1.);
        let snapshot = files.workspace().session(document).unwrap().snapshot();
        let placed = &snapshot
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .block_instances()[0];
        assert_eq!(placed.position(), [2. / scale, 3. / scale]);
    }
}
#[test]
fn invalid_placement_keeps_history_and_does_not_reserve_an_identity() {
    let (mut files, document, mut source) = setup(1.);
    let before = owner_revision(&files, document);
    let valid = AddSketchBlockArgs {
        definition: "Rib".into(),
        position: Some(vec![0., 0.]),
        ..Default::default()
    };
    assert!(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .add_block_instance(&valid)
            .is_err()
    );
    assert_eq!(owner_revision(&files, document), before);
    for invalid in [
        AddSketchBlockArgs {
            definition: "missing".into(),
            ..valid.clone()
        },
        AddSketchBlockArgs {
            position: Some(vec![1.]),
            ..valid.clone()
        },
        AddSketchBlockArgs {
            rotation_angle: "3 cm".into(),
            ..valid.clone()
        },
        AddSketchBlockArgs {
            scale: f64::INFINITY,
            ..valid.clone()
        },
        AddSketchBlockArgs {
            sketch_index: 99,
            ..valid.clone()
        },
    ] {
        assert!(add(&mut files, &mut source, &invalid).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert!(instances(&files).is_empty());
    }
    assert_eq!(add(&mut files, &mut source, &valid).unwrap().entity_id, 3);
}

#[test]
fn aborted_group_removes_the_instance_and_retires_its_identity() {
    let (mut files, document, mut source) = setup(1.);
    let request = AddSketchBlockArgs {
        definition: "Rib".into(),
        position: Some(vec![0., 0.]),
        ..Default::default()
    };
    files
        .commands(document)
        .unwrap()
        .begin("Place block group")
        .unwrap();
    let first = add(&mut files, &mut source, &request).unwrap();
    assert_eq!(instances(&files).len(), 1);
    files.commands(document).unwrap().abort().unwrap();
    assert!(instances(&files).is_empty());
    let second = add(&mut files, &mut source, &request).unwrap();
    assert!(second.entity_id > first.entity_id);
}
