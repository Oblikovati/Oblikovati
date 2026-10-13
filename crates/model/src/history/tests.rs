// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::units::UnitsOfMeasure;

fn empty_history() -> DocumentEditHistory {
    DocumentEditHistory::new(
        DocumentSnapshot {
            work_geometry: Default::default(),
            sketches: Default::default(),
            attributes: Default::default(),
            attachments: Default::default(),
            interests: Default::default(),
            resources: Default::default(),
            assets: Default::default(),
            body_names: Default::default(),
            sketch_settings: None,
            parameters: ParameterCollection::new(UnitsOfMeasure::default()),
            properties: DocumentProperties::default(),
        },
        RecipeIdAllocator::default(),
    )
    .unwrap()
}

#[test]
fn exhausted_tokens_do_not_open_another_frame() {
    let mut history = empty_history();
    history.next_token = u64::MAX - 1;
    let last = history.begin("last").unwrap();
    let error = history.begin("overflow").unwrap_err();
    assert_eq!(error, HistoryError::Exhausted("transaction tokens"));
    assert!(error.to_string().contains("u64::MAX"));
    history.commit(&last, false).unwrap();
    assert_eq!(history.undo_labels(), ["last"]);
}

#[test]
fn exhausted_generation_refuses_edit_and_abort_without_calling_mutation() {
    let mut history = empty_history();
    let token = history.begin("open").unwrap();
    history.generation.store(u64::MAX, Ordering::Release);
    let result = history.edit(&token, |_, _| -> Result<(), ()> {
        panic!("exhausted generation must not invoke command")
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::Exhausted(_)))
    ));
    assert!(matches!(
        history.abort(&token),
        Err(HistoryError::Exhausted(_))
    ));
    assert_eq!(history.open.len(), 1);
    assert_eq!(history.high_water(), 0);
    history.commit(&token, false).unwrap();
    assert_eq!(history.generation(), u64::MAX);
}

#[test]
fn exhausted_generation_preserves_both_navigation_stacks() {
    let mut history = empty_history();
    let token = history.begin("step").unwrap();
    history.commit(&token, true).unwrap();
    history.generation.store(u64::MAX, Ordering::Release);
    assert!(matches!(history.undo(), Err(HistoryError::Exhausted(_))));
    assert_eq!(history.undo_labels(), ["step"]);
    history.generation.store(u64::MAX - 1, Ordering::Release);
    history.undo().unwrap();
    assert!(matches!(history.redo(), Err(HistoryError::Exhausted(_))));
    assert_eq!(history.redo_labels(), ["step"]);
    assert!(history.undo_labels().is_empty());
}
