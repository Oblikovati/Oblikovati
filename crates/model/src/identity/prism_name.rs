// SPDX-License-Identifier: GPL-2.0-only
//! One semantic naming rule for authored sketch-prism results.
use super::{DocumentGuid, SemanticReferenceName};
use crate::{document_sketch::DocumentSketchId, feature::FeatureId};

/// Construction role used by prism naming, independent of kernel handles.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum SketchPrismReferenceRole {
    Body,
    StartCap,
    EndCap,
    Side(u64),
}

/// Encode the stable prism role using its authored sources.
/// Example: `sketch_prism_reference_name(doc, feature, sketch, SketchPrismReferenceRole::StartCap)?`.
pub fn sketch_prism_reference_name(
    document: DocumentGuid,
    feature: FeatureId,
    sketch: DocumentSketchId,
    role: SketchPrismReferenceRole,
) -> Result<Vec<u8>, String> {
    sketch_prism_output_reference_name(document, feature, sketch, 0, role)
}

/// Encode a role for one output slot, with slot zero retaining the original format.
/// Example: `sketch_prism_output_reference_name(doc, feature, sketch, 7, SketchPrismReferenceRole::Body)?`.
pub fn sketch_prism_output_reference_name(
    document: DocumentGuid,
    feature: FeatureId,
    sketch: DocumentSketchId,
    slot: u32,
    role: SketchPrismReferenceRole,
) -> Result<Vec<u8>, String> {
    let (kind, tag, mut sources) = match role {
        SketchPrismReferenceRole::Body => (4, "body", vec![sketch.0]),
        SketchPrismReferenceRole::StartCap => (1, "start", vec![sketch.0]),
        SketchPrismReferenceRole::EndCap => (1, "end", vec![sketch.0]),
        SketchPrismReferenceRole::Side(edge) => (1, "side", vec![sketch.0, edge]),
    };
    if slot != 0 {
        sources.push(u64::from(slot));
    }
    SemanticReferenceName::new(document, feature.0, kind, "sketch-prism", tag, &sources)?.encode()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn roles_encode_the_existing_semantic_names_and_validate_sources() {
        let document = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        for (role, kind, tag, sources) in [
            (SketchPrismReferenceRole::Body, 4, "body", vec![7]),
            (SketchPrismReferenceRole::StartCap, 1, "start", vec![7]),
            (SketchPrismReferenceRole::EndCap, 1, "end", vec![7]),
            (SketchPrismReferenceRole::Side(9), 1, "side", vec![7, 9]),
        ] {
            let actual =
                sketch_prism_reference_name(document, FeatureId(3), DocumentSketchId(7), role)
                    .unwrap();
            let expected =
                SemanticReferenceName::new(document, 3, kind, "sketch-prism", tag, &sources)
                    .unwrap()
                    .encode()
                    .unwrap();
            assert_eq!(actual, expected);
        }
        assert!(
            sketch_prism_reference_name(
                document,
                FeatureId(0),
                DocumentSketchId(7),
                SketchPrismReferenceRole::StartCap
            )
            .is_err()
        );
        assert!(
            sketch_prism_reference_name(
                document,
                FeatureId(3),
                DocumentSketchId(0),
                SketchPrismReferenceRole::EndCap
            )
            .is_err()
        );
        assert!(
            sketch_prism_reference_name(
                document,
                FeatureId(3),
                DocumentSketchId(7),
                SketchPrismReferenceRole::Side(0)
            )
            .is_err()
        );
    }
}
