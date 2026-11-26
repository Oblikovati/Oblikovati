// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn single_and_chain_offsets_commit_and_undo_with_the_original_geometry_intact() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let (single, chain, new_circle, new_arc) = {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        client.sketch().create(&Default::default()).unwrap();
        let lines = client
            .sketch()
            .add_entity(&AddSketchEntityArgs {
                kind: "polyline".into(),
                points: Some(
                    [[0., 0.], [4., 0.], [4., 3.]]
                        .into_iter()
                        .map(|point| Some(point.to_vec()))
                        .collect(),
                ),
                ..Default::default()
            })
            .unwrap()
            .entity_i_ds
            .unwrap();
        let single = client
            .sketch()
            .offset(&OffsetSketchArgs {
                entity: lines[0],
                distance: "1 cm".into(),
                ..Default::default()
            })
            .unwrap();
        let chain = client
            .sketch()
            .offset(&OffsetSketchArgs {
                entities: Some(lines),
                distance: "1 cm".into(),
                ..Default::default()
            })
            .unwrap();
        let circle = client
            .sketch()
            .add_entity(&AddSketchEntityArgs {
                kind: "circle".into(),
                radius: "2 cm".into(),
                points: Some(vec![Some(vec![10., 10.])]),
                ..Default::default()
            })
            .unwrap()
            .entity_id;
        let new_circle = client
            .sketch()
            .offset(&OffsetSketchArgs {
                entity: circle,
                distance: "-1 cm".into(),
                ..Default::default()
            })
            .unwrap();
        assert!(
            client
                .sketch()
                .offset(&OffsetSketchArgs {
                    entity: circle,
                    distance: "-2 cm".into(),
                    ..Default::default()
                })
                .is_err()
        );
        let arc = client
            .sketch()
            .add_entity(&AddSketchEntityArgs {
                kind: "arc".into(),
                ccw: true,
                points: Some(vec![
                    Some(vec![0., 0.]),
                    Some(vec![8., 0.]),
                    Some(vec![0., 8.]),
                ]),
                ..Default::default()
            })
            .unwrap()
            .entity_id;
        let new_arc = client
            .sketch()
            .offset(&OffsetSketchArgs {
                entity: arc,
                distance: "1 cm".into(),
                ..Default::default()
            })
            .unwrap();
        client.transactions().undo().unwrap();
        client.transactions().redo().unwrap();
        (
            single.entity_id,
            chain.created.unwrap(),
            new_circle.entity_id,
            new_arc.entity_id,
        )
    };
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let sketch = snapshot.sketches.sketches().first().unwrap().geometry();
    let segment = |id| sketch.line(SketchEntityId(id)).unwrap().endpoints;
    let position = |id| sketch.point(id).unwrap().position;
    assert_eq!(segment(single).map(position), [[0., 1.], [4., 1.]]);
    assert_eq!(chain.len(), 2);
    let [a, join] = segment(chain[0]);
    assert_eq!(segment(chain[1])[0], join);
    assert_eq!([a, join].map(position), [[0., 1.], [3., 1.]]);
    assert_eq!(
        sketch.circle(SketchEntityId(new_circle)).unwrap().radius,
        1.
    );
    let arc = sketch.arc(SketchEntityId(new_arc)).unwrap();
    assert_eq!(position(arc.start), [9., 0.]);
    assert_eq!(position(arc.end), [0., 9.]);
    assert!(arc.counter_clockwise);
}
