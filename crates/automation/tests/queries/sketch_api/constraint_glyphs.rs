// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn glyph_fixture() -> (
    DiscoveryWorkspace,
    u64,
    DocumentSketchId,
    QueryGuids,
    SketchConstraintId,
) {
    let (mut files, document, sketch, source) = files_with_circle(Default::default());
    let relation = files
        .commands(document)
        .unwrap()
        .run("Glyph fixture", |snapshot, ids| {
            let point = ids
                .allocate_sketch_entity()
                .map_err(|error| error.to_string())?;
            let relation = ids
                .allocate_sketch_constraint(sketch)
                .map_err(|error| error.to_string())?;
            snapshot
                .sketches
                .replace_geometry(sketch, point_geometry(point, vec![relation]))
                .map_err(|error| error.to_string())?;
            Ok::<_, String>(relation)
        })
        .unwrap();
    (files, document, sketch, source, relation)
}
fn view(files: &DiscoveryWorkspace) -> ConstraintGlyphsVisibilityResult {
    Client::new(FileQueryHost::new(files))
        .constrain()
        .glyph_visibility(&ConstraintGlyphsVisibilityArgs { sketch_index: 0 })
        .unwrap()
}
#[test]
fn typed_glyph_edits_round_trip_and_show_all_clears_hidden_ids() {
    let (mut files, document, sketch, mut source, relation) = glyph_fixture();
    let before = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .clone();
    let host = FileHost::new(&mut files, &mut source, "test");
    let client = Client::new(host);
    client
        .constrain()
        .set_all_glyphs_visible(&AllConstraintGlyphsVisibilityArgs {
            sketch_index: 0,
            visible: true,
        })
        .unwrap();
    client
        .constrain()
        .set_glyph_visible(&ConstraintGlyphVisibilityArgs {
            sketch_index: 0,
            constraint_id: relation.0,
            visible: false,
        })
        .unwrap();
    drop(client);
    assert_eq!(view(&files).relation_i_ds, Some(vec![relation.0]));
    assert_eq!(view(&files).hidden_i_ds, Some(vec![relation.0]));
    let revision = owner_revision(&files, document);
    let host = FileHost::new(&mut files, &mut source, "test");
    let client = Client::new(host);
    assert!(
        client
            .constrain()
            .set_glyph_visible(&ConstraintGlyphVisibilityArgs {
                sketch_index: 0,
                constraint_id: u64::MAX,
                visible: false
            })
            .is_err()
    );
    drop(client);
    assert_eq!(owner_revision(&files, document), revision);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .constrain()
        .set_all_glyphs_visible(&AllConstraintGlyphsVisibilityArgs {
            sketch_index: 0,
            visible: true,
        })
        .unwrap();
    assert_eq!(view(&files).hidden_i_ds, Some(vec![]));
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(view(&files).hidden_i_ds, Some(vec![relation.0]));
    assert_eq!(
        files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry(),
        &before
    );
}
