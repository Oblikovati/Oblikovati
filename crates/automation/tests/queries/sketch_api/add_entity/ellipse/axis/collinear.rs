// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn fix_center(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, ellipse: u64) {
    let document = files.workspace().active_document().unwrap();
    let center = owned_geometry(files, document).sketches()[0]
        .geometry()
        .ellipse(SketchEntityId(ellipse))
        .unwrap()
        .center();
    Client::new(FileHost::new(files, source, "test"))
        .constrain()
        .add(&AddConstraintArgs {
            kind: "fix".into(),
            entities: Some(vec![center.0]),
            ..Default::default()
        })
        .unwrap();
}
fn collinear_request(ellipse: u64, line: u64, major: bool, reverse: bool) -> AddConstraintArgs {
    let ids = if reverse {
        [ellipse, line]
    } else {
        [line, ellipse]
    };
    let mut request = pair_request(ids, false, Some(major), reverse);
    request.kind = "collinear".into();
    request
}
#[test]
fn collinear_api_line_uses_the_grounded_center_for_both_axes_and_operand_orders() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for (_, major, reverse) in pair_cases()
            .into_iter()
            .filter(|(perp, major, _)| !perp && major.is_some())
        {
            check_collinear_constructor(owner, major.unwrap(), reverse);
        }
    }
}
fn check_collinear_constructor(owner: DocumentType, major: bool, reverse: bool) {
    let (mut files, _, mut source, ellipse, line) = fixture(owner);
    fix_center(&mut files, &mut source, ellipse);
    let request = collinear_request(ellipse, line, major, reverse);
    let result = add_pair(&mut files, &mut source, &request).unwrap();
    assert_eq!(result.kind, "collinear");
    assert_eq!(result.dof, 4);
    check_collinear_rows(&files, line, major);
    check_pair_row(
        &files,
        "collinear",
        request.entities.unwrap().try_into().unwrap(),
    );
}
fn check_collinear_rows(files: &DiscoveryWorkspace, line: u64, major: bool) {
    let points = row(files, line).points.unwrap();
    let index = usize::from(major);
    let coordinate = if major { 2.0 } else { 1.0 };
    for position in points.into_iter().flatten() {
        assert!((position[index] - coordinate).abs() < 1e-8);
    }
    check_line_direction(files, line, !major);
}
#[test]
fn collinear_axis_selection_and_anchors_survive_history_reopen_and_copy() {
    let (mut files, document, mut source, ellipse, line) = fixture(DOCUMENT_PART);
    fix_center(&mut files, &mut source, ellipse);
    let request = collinear_request(ellipse, line, false, true);
    let original = rows(&files);
    add_pair(&mut files, &mut source, &request).unwrap();
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), original);
    files.commands(document).unwrap().redo().unwrap();
    check_collinear_rows(&files, line, false);
    check_saved_collinear(&mut files, document, &mut source, line, &request);
}
fn check_saved_collinear(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    line: u64,
    request: &AddConstraintArgs,
) {
    let document = reopen_with_observer(files, document, source, Default::default());
    check_collinear_rows(files, line, false);
    activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/ellipse-collinear-copy.obk"),
    );
    check_collinear_rows(files, line, false);
    let ids = request
        .entities
        .as_ref()
        .unwrap()
        .as_slice()
        .try_into()
        .unwrap();
    check_pair_row(files, "collinear", ids);
}
