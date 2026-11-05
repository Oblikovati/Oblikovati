// SPDX-License-Identifier: GPL-2.0-only
use super::super::spline_requests::{fit_request, offset_request};
use super::*;
use oblikovati_model::sketch::SketchEntityId;

fn create_spline_family(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
) -> [AddSketchEntityResult; 3] {
    let fit = add_call(files, source, &fit_request("chord", true)).unwrap();
    let mut control = fit_request("", false);
    control.kind = "controlPointSpline".into();
    control.construction = true;
    let control = add_call(files, source, &control).unwrap();
    let offset = offset_request(fit.entity_id, "-5 mm");
    let offset = add_call(files, source, &offset).unwrap();
    [fit, control, offset]
}

#[test]
fn typed_spline_family_survives_both_owners_reopen_and_copy() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_spline_family(owner);
    }
}

fn verify_spline_family(owner: DocumentType) {
    let (mut files, document, mut source) = configured_files(owner, Default::default());
    create_empty_sketch(&mut files, &mut source);
    let [fit, control, offset] = create_spline_family(&mut files, &mut source);
    assert_family(&files, document, [&fit, &control, &offset]);
    let before = rows(&files);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), before);
    activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/spline-api-copy.obk"),
    );
    let copied = rows(&files);
    assert_eq!(copied.len(), before.len());
    assert_copy_namespaces(&before, &copied);
    verify_keys(&files, &copied);
}

fn assert_family(
    files: &DiscoveryWorkspace,
    document: u64,
    [fit, control, offset]: [&AddSketchEntityResult; 3],
) {
    assert_eq!(fit.kind, "spline");
    assert_eq!(fit.point_i_ds.as_ref().unwrap().len(), 5);
    assert_eq!(row(files, fit.entity_id).fit_method, "chord");
    assert_eq!(control.kind, "controlPointSpline");
    assert!(row(files, control.entity_id).construction);
    assert_eq!(offset.kind, "offsetSpline");
    assert_eq!(offset.point_i_ds, None);
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let owned = geometry
        .offset_spline(SketchEntityId(offset.entity_id))
        .unwrap();
    assert_eq!(owned.parent(), SketchEntityId(fit.entity_id));
    assert_eq!(owned.distance(), -0.5);
}

#[test]
fn malformed_spline_requests_are_atomic_and_leave_first_ids_available() {
    let (mut files, document, mut source) = fresh();
    let mut invalid = fit_request("bogus", false);
    assert_refused(&mut files, document, &mut source, &invalid, "fit method");
    invalid = literal_input("spline", &[[0., 0.]]);
    assert_refused(
        &mut files,
        document,
        &mut source,
        &invalid,
        "two defining points",
    );
    invalid = offset_request(99, "5 mm");
    assert_refused(&mut files, document, &mut source, &invalid, "spline");
    invalid = offset_request(2, "5 mm");
    assert_refused(&mut files, document, &mut source, &invalid, "spline");
    invalid = offset_request(99, "1 rad");
    assert_refused(&mut files, document, &mut source, &invalid, "length");
    let created = add_call(&mut files, &mut source, &fit_request("smooth", false)).unwrap();
    assert_eq!(created.point_i_ds, Some(vec![2, 3, 4, 5, 6]));
    assert_eq!(created.entity_id, 7);
}
