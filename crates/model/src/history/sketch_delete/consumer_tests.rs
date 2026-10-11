// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::feature::{FeatureId, FeatureSequenceError};
use std::error::Error;

#[test]
fn key_namespaces_are_distinct_and_removing_a_reader_keeps_other_consumers() {
    let mut index = DeletionIncidences::default();
    let target = FeatureReadKey::Feature(FeatureId(1));
    let first = FeatureReadKey::Sketch(1);
    let second = FeatureReadKey::Sketch(2);
    let footprint = FeatureReadFootprint::new(BTreeSet::from([target]), false);
    index.record(first, &footprint);
    index.record(second, &footprint);
    assert!(index.require_unused(DocumentSketchId(1)).is_ok());
    index.remove_consumer(first).unwrap();
    assert_eq!(index.consumer_keys(target), [second]);
    index.remove_consumer(second).unwrap();
    assert!(!index.has_consumers(target));
}

#[test]
fn inconsistent_source_target_or_reader_edges_have_named_declines() {
    let mut index = DeletionIncidences::default();
    let reader = FeatureReadKey::Sketch(2);
    let target = FeatureReadKey::Feature(FeatureId(3));
    assert_eq!(
        index.remove_consumer(reader),
        Err(DocumentSketchDeleteError::MissingIncidence(reader))
    );
    assert_eq!(
        index.remove_edge(reader, target),
        Err(DocumentSketchDeleteError::MissingIncidence(target))
    );
    index.readers.insert(target, BTreeSet::new());
    let error = index.remove_edge(reader, target).unwrap_err();
    assert_eq!(error, DocumentSketchDeleteError::MissingIncidence(reader));
    assert!(error.to_string().contains("Sketch(2)"));
    assert!(error.source().is_none());
}

#[test]
fn work_plane_declines_keep_their_cause_and_diagnostic() {
    let cause = FeatureSequenceError::Index {
        index: 8,
        length: 0,
    };
    let error = DocumentSketchDeleteError::Plane(cause.clone());
    assert_eq!(error.to_string(), cause.to_string());
    assert_eq!(error.source().unwrap().to_string(), cause.to_string());
}
