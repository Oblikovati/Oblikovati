// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEntityId;

fn sketch_selection_fixture(sketches: &[u64], entities: &[(u64, u64)]) -> ViewportSketchSelection {
    ViewportSketchSelection {
        sketches: sketches.iter().copied().map(DocumentSketchId).collect(),
        entities: entities
            .iter()
            .map(|(sketch, entity)| (DocumentSketchId(*sketch), SketchEntityId(*entity)))
            .collect(),
    }
}

#[test]
fn normalization_preserves_source_and_document_scopes_while_sorting_and_deduplicating() {
    let source = sketch_selection_fixture(&[3, 1, 3], &[(3, 9), (1, 9), (1, 4), (1, 9)]);
    let original = source.clone();
    let normalized = source.normalized();
    let expected = sketch_selection_fixture(&[1, 3], &[(1, 4), (1, 9), (3, 9)]);
    assert_eq!(normalized, expected);
    assert_eq!(source, original);
    assert_eq!(normalized.normalized(), normalized);
}

#[test]
fn empty_selection_normalization_stays_empty() {
    let selection = sketch_selection_fixture(&[], &[]);
    assert_eq!(selection, ViewportSketchSelection::default());
    assert_eq!(selection.normalized(), selection);
}
