// SPDX-License-Identifier: GPL-2.0-only
use super::entities::{configured_scaled, install_mixed};
use super::*;
use oblikovati_api::wire::{AddSketchPatternArgs, AddSketchPatternResult};

fn request() -> AddSketchPatternArgs {
    AddSketchPatternArgs {
        kind: "rectangular".into(),
        entities: Some(vec![4]),
        count1: 3,
        count2: 2,
        spacing1: "10 mm".into(),
        spacing2: "5 mm".into(),
        dir1: Some(vec![-2., 0.]),
        dir2: Some(vec![0., 3.]),
        ..Default::default()
    }
}
fn call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    request: &AddSketchPatternArgs,
) -> Result<AddSketchPatternResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .pattern()
        .rectangular(request)
}
fn pattern(
    files: &DiscoveryWorkspace,
    document: u64,
) -> oblikovati_model::sketch::SketchRectangularPattern {
    owned_geometry(files, document).sketches()[0]
        .geometry()
        .rectangular_patterns()[0]
        .clone()
}
fn x_offset(files: &DiscoveryWorkspace, document: u64, id: u64) -> f64 {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let seed = geometry.circle(SketchEntityId(4)).unwrap();
    let member = geometry.circle(SketchEntityId(id)).unwrap();
    geometry.point(member.center).unwrap().position[0]
        - geometry.point(seed.center).unwrap().position[0]
}
#[test]
fn rectangular_pattern_api_drives_spacing_and_survives_history_and_reopen() {
    for scale in [0.1, 2.0] {
        let (mut files, document, mut source) = configured_scaled(scale);
        install_mixed(&mut files, document);
        let mut input = request();
        input.spacing1 = "width".into();
        let created = call(&mut files, &mut source, &input)
            .unwrap()
            .created
            .unwrap();
        assert_eq!(created.len(), 5);
        let record = pattern(&files, document);
        assert_eq!(record.definition.counts, [3, 2]);
        let member = record
            .members
            .iter()
            .find(|item| item.index == [1, 0])
            .unwrap()
            .entities[0]
            .0;
        assert!(created.contains(&member));
        let initial = x_offset(&files, document, member);
        files
            .commands(document)
            .unwrap()
            .run("Edit pattern spacing", |snapshot, _| {
                snapshot.parameters.set_expression(1, "20 mm")
            })
            .unwrap();
        assert!((x_offset(&files, document, member) - initial * 2.).abs() < 1e-7);
        files.commands(document).unwrap().undo().unwrap();
        assert!((x_offset(&files, document, member) - initial).abs() < 1e-7);
        files.commands(document).unwrap().undo().unwrap();
        assert!(
            owned_geometry(&files, document).sketches()[0]
                .geometry()
                .rectangular_patterns()
                .is_empty()
        );
        files.commands(document).unwrap().redo().unwrap();
        files.commands(document).unwrap().redo().unwrap();
        let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
        assert_eq!(pattern(&files, reopened), record);
        assert!((x_offset(&files, reopened, member) - initial * 2.).abs() < 1e-7);
    }
}
#[test]
fn invalid_pattern_requests_are_atomic_and_do_not_publish_parameters() {
    let (mut files, document, mut source) = configured_scaled(1.0);
    install_mixed(&mut files, document);
    let before = owned_geometry(&files, document).clone();
    let parameters = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .clone();
    let revision = owner_revision(&files, document);
    let mut invalid = request();
    invalid.count1 = 0;
    assert!(call(&mut files, &mut source, &invalid).is_err());
    invalid = request();
    invalid.dir1 = Some(vec![0., 0.]);
    assert!(call(&mut files, &mut source, &invalid).is_err());
    invalid = request();
    invalid.spacing2 = "bad_name".into();
    assert!(call(&mut files, &mut source, &invalid).is_err());
    invalid = request();
    invalid.kind = "circular".into();
    assert!(call(&mut files, &mut source, &invalid).is_err());
    let session = files.workspace().session(document).unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(revision.0, session.generation());
    assert!(parameters.same_recipe(&session.snapshot().parameters));
}

#[test]
fn observer_veto_does_not_publish_pattern_or_spacing_parameters() {
    let (mut files, document, mut source) = configured_scaled(1.0);
    install_mixed(&mut files, document);
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let session = files.workspace().session(document).unwrap();
    let before = session.snapshot().sketches.clone();
    let parameters = session.snapshot().parameters.clone();
    expect_api_decline(call(&mut files, &mut source, &request()), "locked");
    let session = files.workspace().session(document).unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(parameters.same_recipe(&session.snapshot().parameters));
}
