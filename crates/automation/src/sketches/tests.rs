// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn wire_counts_refuse_overflow_instead_of_truncating_structural_diagnostics() {
    assert_eq!(wire_count(0).unwrap(), 0);
    if let Ok(count) = usize::try_from(i64::MAX as u64 + 1) {
        let message = wire_count(count).unwrap_err().to_string();
        assert!(message.contains(&count.to_string()));
        assert!(message.contains("nonnegative i64"));
    }
}

#[test]
fn incomplete_or_cancelled_reports_cannot_produce_a_success_response() {
    for cancelled in [false, true] {
        let report = DocumentSketchRecompute {
            reports: vec![],
            cancelled,
        };
        assert!(
            solve_reply(Ok(&report), 7)
                .unwrap_err()
                .to_string()
                .contains("7")
        );
    }
}

#[test]
fn missing_identity_cannot_produce_a_success_response() {
    let missing = oblikovati_model::document_sketch::DocumentSketches::default();
    let cause = missing.get(DocumentSketchId(99)).unwrap_err();
    assert!(
        solve_reply(Err(cause), 7)
            .unwrap_err()
            .to_string()
            .contains("99")
    );
}
