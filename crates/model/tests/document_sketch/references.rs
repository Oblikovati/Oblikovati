// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn an_ambiguous_key_is_declined_without_replacing_its_first_target() {
    let guid: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut index = DocumentSketchReferenceIndex::new(&DocumentSketches::default(), guid).unwrap();
    let first = SketchReferenceTarget {
        sketch: DocumentSketchId(1),
        entity: None,
    };
    let second = SketchReferenceTarget {
        sketch: DocumentSketchId(2),
        entity: None,
    };
    index.insert("collision".into(), first).unwrap();
    let error = index.insert("collision".into(), second).unwrap_err();
    assert!(error.to_string().contains("ambiguous sketch reference"));
    assert!(error.to_string().contains("collision"));
    assert!(error.to_string().contains("DocumentSketchId(2)"));
    assert_eq!(index.resolve("collision"), Some(first));
}
