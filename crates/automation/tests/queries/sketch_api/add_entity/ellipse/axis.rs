// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{AddConstraintArgs, AddConstraintResult};
fn align(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    id: u64,
    vertical: bool,
    major: Option<bool>,
) -> AddConstraintResult {
    Client::new(FileHost::new(files, source, "test"))
        .constrain()
        .add(&AddConstraintArgs {
            kind: if vertical { "vertical" } else { "horizontal" }.into(),
            entities: Some(vec![id]),
            use_ellipse_major_axis: major.map(std::boxed::Box::new),
            ..Default::default()
        })
        .unwrap()
}
#[test]
fn public_axis_alignment_rotates_both_axes_in_part_and_assembly_documents() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for vertical in [false, true] {
            for major in [None, Some(true), Some(false)] {
                check_constructor(kind, vertical, major);
            }
        }
    }
}
fn check_constructor(kind: DocumentType, vertical: bool, major: Option<bool>) {
    let (mut files, document, mut source) = configured_files(kind, Default::default());
    create_empty_sketch(&mut files, &mut source);
    let created = add_call(&mut files, &mut source, &input()).unwrap();
    let result = align(&mut files, &mut source, created.entity_id, vertical, major);
    assert_eq!(
        result.kind,
        if vertical { "vertical" } else { "horizontal" }
    );
    assert_eq!(result.dof, 4);
    assert_aligned(
        &files,
        document,
        created.entity_id,
        major.unwrap_or(true) == vertical,
    );
    check_enumerated(&files, created.entity_id, &result.kind);
}
fn check_enumerated(files: &DiscoveryWorkspace, id: u64, kind: &str) {
    let rows = Client::new(FileQueryHost::new(files))
        .sketch()
        .constraints(&Default::default())
        .unwrap();
    let rows = rows.constraints.unwrap();
    let relation = rows.iter().find(|row| row.kind == kind).unwrap();
    assert_eq!(relation.entities.as_deref(), Some([id].as_slice()));
}
fn assert_aligned(files: &DiscoveryWorkspace, document: u64, id: u64, major_vertical: bool) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let value = geometry.ellipse(SketchEntityId(id)).unwrap();
    assert!(value.major_axis()[usize::from(!major_vertical)].abs() < 1e-8);
    assert_eq!(value.radii(), [3.0, 1.0]);
    assert_eq!(geometry.point(value.center()).unwrap().position, [1.0, 2.0]);
}
#[test]
fn public_axis_alignment_uses_history_and_preserves_constraints_through_reopen_and_copy() {
    let (mut files, document, mut source) = fresh();
    let created = add_call(&mut files, &mut source, &input()).unwrap();
    align(
        &mut files,
        &mut source,
        created.entity_id,
        true,
        Some(false),
    );
    assert_aligned(&files, document, created.entity_id, false);
    files.commands(document).unwrap().undo().unwrap();
    assert_frame(&files, document, created.entity_id, 1.0);
    files.commands(document).unwrap().redo().unwrap();
    assert_aligned(&files, document, created.entity_id, false);
    check_reopened_alignment(&mut files, document, &mut source, created.entity_id);
}
fn check_reopened_alignment(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    id: u64,
) {
    let document = reopen_with_observer(files, document, source, Default::default());
    assert_aligned(files, document, id, false);
    activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/ellipse-axis-copy.obk"),
    );
    let copied = files.workspace().active_document().unwrap();
    assert_aligned(files, copied, id, false);
}

#[path = "axis/pairs.rs"]
mod pairs;
