// SPDX-License-Identifier: GPL-2.0-only
use super::part_evaluation::prism_fixture as fixture;
use super::*;
use oblikovati_model::feature::{FeatureId, part::PartFeatureDefinition};

fn profile_package() -> DocumentPackage {
    let snapshot = fixture::selected_prism_document();
    DocumentPackage::empty(DOCUMENT_PART, "Extrusion")
        .with_parameters(&snapshot.parameters, snapshot.parameters.units().clone())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
}
fn extrusion_depth(host: &Host, document: u64, feature: FeatureId) -> f64 {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let authored = snapshot.work_geometry.get(feature).unwrap();
    let PartFeatureDefinition::SketchPrism(definition) = authored.definition() else {
        panic!("feature {feature:?}: expected an authored sketch extrusion");
    };
    definition.resolve(snapshot).unwrap().displacement[2]
}
fn edit_saved_depth(host: &mut Host, document: u64, feature: FeatureId) {
    host.commands(document)
        .unwrap()
        .run("Depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    assert_eq!(extrusion_depth(host, document, feature), 3.);
    host.commands(document).unwrap().undo().unwrap();
    assert_eq!(extrusion_depth(host, document, feature), 2.);
    host.commands(document).unwrap().redo().unwrap();
    assert_eq!(extrusion_depth(host, document, feature), 3.);
}
#[test]
fn session_created_extrusion_reopens_with_parameter_links_and_reserved_identity() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let document = create_profile(&mut host, &mut source);
    let feature = create_extrusion(&mut host, document);
    edit_saved_depth(&mut host, document, feature);
    host.save(document, "alpha", &mut source).unwrap();
    let sketches = snapshot(&host, document).sketches.recipe();
    let reopened = reopen_profile(&mut host, document, &mut source);
    assert_ne!(document, reopened);
    assert_eq!(extrusion_depth(&host, reopened, feature), 3.);
    assert_eq!(snapshot(&host, reopened).sketches.recipe(), sketches);
    assert!(!host.workspace().needs_save(reopened).unwrap());
    assert!(create_extrusion(&mut host, reopened).0 > feature.0);
}
fn snapshot(host: &Host, document: u64) -> &DocumentSnapshot {
    host.workspace().session(document).unwrap().snapshot()
}
fn reopen_profile(host: &mut Host, document: u64, source: &mut SequenceGuids) -> u64 {
    host.close_discarding_changes(document).unwrap();
    open_file(host, "extrusion.obk", source, OpenActivation::Activate)
}
fn create_profile(host: &mut Host, source: &mut SequenceGuids) -> u64 {
    host.create(
        &path("extrusion.obk"),
        profile_package(),
        RecordingObserver::default(),
        "alpha",
        source,
    )
    .unwrap()
    .session_id
}
fn create_extrusion(host: &mut Host, document: u64) -> FeatureId {
    host.commands(document)
        .unwrap()
        .create_sketch_prism("Extrude", fixture::selected_prism_definition())
        .unwrap()
}
