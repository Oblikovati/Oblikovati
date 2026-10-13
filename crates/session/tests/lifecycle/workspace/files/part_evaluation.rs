// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[allow(dead_code)]
#[path = "../../../../../../test-utilities/rust/selected_sketch_prism.rs"]
mod prism_fixture;
use oblikovati_model::feature::{
    FeatureId, part::SketchPrismEvaluator, sketch_prism::ResolvedSketchPrism,
};
use oblikovati_session::part_evaluation::PartEvaluationSlot;
struct FilePrismEvaluator;
impl SketchPrismEvaluator for FilePrismEvaluator {
    type Body = f64;
    type Failure = &'static str;
    fn verify_body_reference(&self, _body: &Self::Body, key: &[u8]) -> Result<(), String> {
        Err(format!(
            "unnamed fake reference {key:?}: expected a named test body"
        ))
    }
    fn evaluate_prism(
        &mut self,
        _feature: oblikovati_model::feature::FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<f64, &'static str> {
        Ok(input.displacement[2])
    }
}
fn stored_prism_package() -> Vec<u8> {
    let snapshot = prism_fixture::selected_prism_document();
    let package = DocumentPackage::decode(&fixture(1)).unwrap();
    let parameters = package
        .with_parameters(&snapshot.parameters, UnitsOfMeasure::default())
        .unwrap();
    let sketches = parameters
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap();
    sketches
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
        .encode()
        .unwrap()
}
fn prism_file_host() -> Host {
    let host = host(RecordingWorkspace::default());
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), stored_prism_package());
    host
}
#[test]
fn file_owned_commands_adopt_restored_prism_results_without_dirtying_the_document() {
    let mut host = prism_file_host();
    let mut guids = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut guids, OpenActivation::Activate);
    let request = host.commands(id).unwrap().prepare_part_evaluation();
    let completed = request.evaluate(&mut FilePrismEvaluator).unwrap();
    let mut slot = PartEvaluationSlot::default();
    host.commands(id)
        .unwrap()
        .adopt_part_evaluation(completed, &mut slot)
        .unwrap();
    let session = host.workspace().session(id).unwrap();
    assert_eq!(
        *slot.current(session).unwrap().body(FeatureId(1)).unwrap(),
        2.
    );
    assert!(session.undo_labels().is_empty());
    assert!(!host.workspace().needs_save(id).unwrap());
}
