// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch_settings::{SketchInferenceError, default_sketch_settings};
fn input(priority: &str) -> InferenceOptionsView {
    InferenceOptionsView {
        infer_enabled: false,
        constrain_enabled: true,
        priority: priority.into(),
    }
}
#[test]
fn failed_response_projection_discards_prepared_settings() {
    let mut session = session(None);
    let generation = session.generation();
    let result = session.set_sketch_inference_options_projected(&input("none"), |candidate| {
        assert_eq!(candidate.unwrap().constraint_priority, PRIORITY_NONE);
        Err::<(), _>("response refused")
    });
    assert!(result.unwrap_err().to_string().contains("response refused"));
    assert_eq!(session.generation(), generation);
    assert_eq!(session.snapshot().sketch_settings, None);
    assert!(session.undo_labels().is_empty());
}
#[test]
fn ignored_decline_cannot_adopt_invalid_preferences() {
    let mut session = session(None);
    session
        .set_sketch_inference_options_projected(&input("bad"), |candidate| {
            assert_eq!(candidate.unwrap_err().0, "bad");
            Ok::<(), SketchInferenceError>(())
        })
        .unwrap();
    assert_eq!(session.snapshot().sketch_settings, None);
    assert!(session.undo_labels().is_empty());
}
#[test]
fn metadata_command_preserves_current_sketch_verification() {
    let mut session = session(None);
    let sketch = session
        .create_sketch(&oblikovati_model::history::DocumentSketchCreation {
            name: "Owner".into(),
            placement: oblikovati_model::document_sketch::SketchPlacement::Fixed(
                oblikovati_model::sketch::SketchPlane::xy(),
            ),
            orientation: None,
            project_origin: true,
        })
        .unwrap();
    let before = session.snapshot().sketches.clone();
    session
        .set_sketch_inference_options_projected(&input("none"), |candidate| candidate.cloned())
        .unwrap();
    assert!(session.snapshot().sketches.same_evaluated_state(&before));
    session.snapshot().require_resolved_sketch(sketch).unwrap();
}
#[test]
fn commit_veto_restores_absence_without_extra_steps() {
    let mut denied = session(Some(EVENT_TRANSACTION_COMMITTED));
    assert!(
        denied
            .set_sketch_inference_options_projected(&input("none"), |candidate| candidate.cloned())
            .is_err()
    );
    assert_eq!(denied.snapshot().sketch_settings, None);
    assert!(denied.undo_labels().is_empty());
}
#[test]
fn group_abort_restores_absence_and_effective_defaults() {
    let mut session = session(None);
    session.begin("Preferences").unwrap();
    session
        .set_sketch_inference_options_projected(&input("none"), |candidate| candidate.cloned())
        .unwrap();
    session.abort().unwrap();
    assert_eq!(session.snapshot().sketch_settings, None);
    assert_eq!(
        session.snapshot().effective_sketch_settings(),
        default_sketch_settings()
    );
}
