// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn fixture(kind: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids, u64, u64) {
    let (mut files, document, mut source) = configured_files(kind, Default::default());
    create_empty_sketch(&mut files, &mut source);
    disable_inference(&mut files, &mut source);
    let ellipse = add_call(&mut files, &mut source, &input())
        .unwrap()
        .entity_id;
    align(&mut files, &mut source, ellipse, false, None);
    let line = add_call(
        &mut files,
        &mut source,
        &literal_input("line", &[[10.0, 10.0], [14.0, 11.0]]),
    )
    .unwrap()
    .entity_id;
    (files, document, source, ellipse, line)
}
fn pair_request(
    ids: [u64; 2],
    perpendicular: bool,
    major: Option<bool>,
    reverse: bool,
) -> AddConstraintArgs {
    let flag = major.map(std::boxed::Box::new);
    AddConstraintArgs {
        kind: if perpendicular {
            "perpendicular"
        } else {
            "parallel"
        }
        .into(),
        entities: Some(ids.to_vec()),
        use_ellipse_one_major_axis: if reverse { flag.clone() } else { None },
        use_ellipse_two_major_axis: if reverse { None } else { flag },
        ..Default::default()
    }
}
fn add_pair(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    request: &AddConstraintArgs,
) -> Result<AddConstraintResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .constrain()
        .add(request)
}
#[test]
fn ellipse_line_pairs_preserve_operand_order_and_default_and_explicit_axis_flags() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for (perpendicular, major, reverse) in pair_cases() {
            check_pair(owner, perpendicular, major, reverse);
        }
    }
}
fn pair_cases() -> Vec<(bool, Option<bool>, bool)> {
    [false, true]
        .into_iter()
        .flat_map(|perpendicular| {
            [None, Some(true), Some(false)]
                .into_iter()
                .flat_map(move |major| [false, true].map(|reverse| (perpendicular, major, reverse)))
        })
        .collect()
}

fn check_pair(owner: DocumentType, perpendicular: bool, major: Option<bool>, reverse: bool) {
    let (mut files, document, mut source, ellipse, line) = fixture(owner);
    let ids = if reverse {
        [ellipse, line]
    } else {
        [line, ellipse]
    };
    let request = pair_request(ids, perpendicular, major, reverse);
    let result = add_pair(&mut files, &mut source, &request).unwrap();
    assert_eq!(result.kind, request.kind);
    assert_eq!(result.dof, 7);
    check_line_direction(&files, line, major.unwrap_or(true) == perpendicular);
    assert_aligned(&files, document, ellipse, false);
    check_pair_row(&files, &request.kind, ids);
}
fn check_line_direction(files: &DiscoveryWorkspace, line: u64, vertical: bool) {
    let points = row(files, line).points.unwrap();
    let a = points[0].as_ref().unwrap();
    let b = points[1].as_ref().unwrap();
    let delta = [b[0] - a[0], b[1] - a[1]];
    let length = delta[0].hypot(delta[1]);
    assert!(length > 0.0);
    assert!(delta[usize::from(!vertical)].abs() / length < 1e-8);
}
fn check_pair_row(files: &DiscoveryWorkspace, kind: &str, ids: [u64; 2]) {
    let rows = Client::new(FileQueryHost::new(files))
        .sketch()
        .constraints(&Default::default())
        .unwrap()
        .constraints
        .unwrap();
    let row = rows.iter().find(|row| row.kind == kind).unwrap();
    assert_eq!(row.entities.as_deref(), Some(ids.as_slice()));
}
#[test]
fn axis_pair_undo_reopen_and_copy_keep_the_minor_selector_and_authored_order() {
    let (mut files, document, mut source, ellipse, line) = fixture(DOCUMENT_PART);
    let request = pair_request([ellipse, line], false, Some(false), true);
    let before = rows(&files);
    add_pair(&mut files, &mut source, &request).unwrap();
    check_line_direction(&files, line, true);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), before);
    files.commands(document).unwrap().redo().unwrap();
    check_line_direction(&files, line, true);
    check_saved_pair(&mut files, document, &mut source, line, &request);
}
fn check_saved_pair(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    line: u64,
    request: &AddConstraintArgs,
) {
    let document = reopen_with_observer(files, document, source, Default::default());
    check_line_direction(files, line, true);
    activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/ellipse-pair-copy.obk"),
    );
    check_line_direction(files, line, true);
    let ids = request
        .entities
        .as_ref()
        .unwrap()
        .as_slice()
        .try_into()
        .unwrap();
    check_pair_row(files, &request.kind, ids);
}
#[test]
fn ellipse_axis_pairs_refuse_unsupported_kinds_and_nonaxis_or_unknown_operands_atomically() {
    let (mut files, document, mut source, ellipse, line) = fixture(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    for refs in [[ellipse, 1], [ellipse, 999]] {
        assert!(
            add_pair(
                &mut files,
                &mut source,
                &pair_request(refs, false, None, true)
            )
            .is_err()
        );
    }
    refuse_collinear(&mut files, &mut source, line, ellipse);
    assert_eq!(owner_revision(&files, document), before);
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
}

fn disable_inference(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .set_inference_options(&oblikovati_api::wire::InferenceOptionsView {
            infer_enabled: false,
            constrain_enabled: false,
            priority: "".into(),
        })
        .unwrap();
}

fn refuse_collinear(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    line: u64,
    ellipse: u64,
) {
    let mut request = pair_request([line, ellipse], false, None, false);
    request.kind = "collinear".into();
    assert!(
        add_pair(files, source, &request)
            .unwrap_err()
            .to_string()
            .contains("not ported")
    );
}
#[test]
fn two_ellipses_use_each_operand_selector_independently() {
    for first_major in [false, true] {
        for (perpendicular, second_major, _) in pair_cases()
            .into_iter()
            .filter(|(_, major, reverse)| major.is_some() && !reverse)
        {
            check_ellipse_pair(first_major, second_major.unwrap(), perpendicular);
        }
    }
}
fn check_ellipse_pair(first_major: bool, second_major: bool, perpendicular: bool) {
    let (mut files, document, mut source, first, _) = fixture(DOCUMENT_PART);
    let second = add_call(&mut files, &mut source, &input())
        .unwrap()
        .entity_id;
    let mut request = pair_request([first, second], perpendicular, Some(second_major), false);
    request.use_ellipse_one_major_axis = Some(std::boxed::Box::new(first_major));
    let result = add_pair(&mut files, &mut source, &request).unwrap();
    assert_eq!(result.dof, 12);
    assert_aligned(
        &files,
        document,
        second,
        (first_major != second_major) ^ perpendicular,
    );
    check_pair_row(&files, &request.kind, [first, second]);
}
