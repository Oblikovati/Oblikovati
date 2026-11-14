// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiParameterRow;
struct ParameterDraftFixture {
    drafts: GuiParameterDrafts,
    snapshot: GuiParameterSnapshot,
    target: GuiParameterTextTarget,
}
impl ParameterDraftFixture {
    fn new() -> Self {
        let snapshot = GuiParameterSnapshot {
            linked_tables: Vec::new(),
            link_sources: Vec::new(),
            groups: Vec::new(),
            document: 7,
            rows: vec![GuiParameterRow {
                tolerance_text: String::new(),
                has_groups: false,
                parameter: 11,
                detail: Default::default(),
            }],
        };
        let mut drafts = GuiParameterDrafts::new(128, 8).unwrap();
        drafts.synchronize(Some(&snapshot)).unwrap();
        let target = GuiParameterTextTarget {
            document: 7,
            parameter: 11,
            field: GuiParameterTextField::Equation,
        };
        Self {
            drafts,
            snapshot,
            target,
        }
    }
    fn edit(&mut self, value: &str) -> GuiParameterTextEdit {
        *self.drafts.text_buffer(self.target, "4 cm").unwrap() = value.into();
        self.drafts.submit(self.target).unwrap().unwrap()
    }
}
#[test]
fn changed_text_is_owned_and_success_reseeds_from_the_model() {
    let mut fixture = ParameterDraftFixture::new();
    assert_eq!(
        fixture.drafts.submit(fixture.target).unwrap_err(),
        missing_cell(fixture.target)
    );
    fixture.drafts.text_buffer(fixture.target, "4 cm").unwrap();
    assert_eq!(fixture.drafts.submit(fixture.target).unwrap(), None);
    let edit = fixture.edit("6 cm");
    assert_eq!(edit.value, "6 cm");
    assert_eq!(fixture.drafts.submit(fixture.target).unwrap(), None);
    assert!(fixture.drafts.text_buffer(fixture.target, "4 cm").is_err());
    assert_eq!(
        fixture.drafts.acknowledge(&edit, Ok(())),
        GuiParameterAck::Applied
    );
    assert_eq!(fixture.drafts.text(fixture.target), None);
    assert_eq!(
        fixture.drafts.text_buffer(fixture.target, "60 mm").unwrap(),
        "60 mm"
    );
}
#[test]
fn failed_edit_retains_text_and_issue_without_automatic_resubmission() {
    let mut fixture = ParameterDraftFixture::new();
    let edit = fixture.edit("missing * 2");
    assert_eq!(
        fixture
            .drafts
            .acknowledge(&edit, Err("Unresolved: missing".into())),
        GuiParameterAck::Failed
    );
    assert_eq!(fixture.drafts.text(fixture.target), Some("missing * 2"));
    assert_eq!(
        fixture.drafts.issue(fixture.target),
        Some("Unresolved: missing")
    );
    assert_eq!(fixture.drafts.submit(fixture.target).unwrap(), None);
    let corrected = fixture.edit("8 cm");
    assert_ne!(corrected.request, edit.request);
    assert_eq!(fixture.drafts.issue(fixture.target), None);
    assert_eq!(
        fixture.drafts.acknowledge(&edit, Ok(())),
        GuiParameterAck::Obsolete
    );
    assert_eq!(fixture.drafts.text(fixture.target), Some("8 cm"));
    assert_eq!(
        fixture.drafts.acknowledge(&corrected, Ok(())),
        GuiParameterAck::Applied
    );
}
#[test]
fn external_row_changes_keep_the_authored_draft_on_the_same_identity() {
    let mut fixture = ParameterDraftFixture::new();
    *fixture.drafts.text_buffer(fixture.target, "4 cm").unwrap() = "6 cm".into();
    fixture.snapshot.rows[0].detail.parameter_info.name = "span".into();
    fixture.drafts.synchronize(Some(&fixture.snapshot)).unwrap();
    assert_eq!(
        fixture.drafts.text_buffer(fixture.target, "12 cm").unwrap(),
        "6 cm"
    );
    assert_eq!(
        fixture
            .drafts
            .submit(fixture.target)
            .unwrap()
            .unwrap()
            .target
            .parameter,
        11
    );
}
#[test]
fn untouched_cells_follow_external_values_and_forged_responses_cannot_clear_pending_text() {
    let mut fixture = ParameterDraftFixture::new();
    fixture.drafts.text_buffer(fixture.target, "4 cm").unwrap();
    assert_eq!(
        fixture.drafts.text_buffer(fixture.target, "8 cm").unwrap(),
        "8 cm"
    );
    let edit = fixture.edit("9 cm");
    let mut forged = edit.clone();
    forged.value = "99 cm".into();
    assert_eq!(
        fixture.drafts.acknowledge(&forged, Ok(())),
        GuiParameterAck::Obsolete
    );
    assert_eq!(fixture.drafts.text(fixture.target), Some("9 cm"));
    assert_eq!(
        fixture.drafts.acknowledge(&edit, Ok(())),
        GuiParameterAck::Applied
    );
}
#[test]
fn document_switch_deletion_and_close_retire_only_their_old_cells() {
    let mut fixture = ParameterDraftFixture::new();
    let old = fixture.edit("6 cm");
    fixture.snapshot.document = 8;
    fixture.drafts.synchronize(Some(&fixture.snapshot)).unwrap();
    assert_eq!(fixture.drafts.text(fixture.target), None);
    assert!(fixture.drafts.submit(fixture.target).is_err());
    assert_eq!(
        fixture.drafts.acknowledge(&old, Ok(())),
        GuiParameterAck::Obsolete
    );
    fixture.target.document = 8;
    let new = fixture.edit("9 cm");
    assert_ne!(new.request, old.request);
    fixture.snapshot.rows.clear();
    fixture.drafts.synchronize(Some(&fixture.snapshot)).unwrap();
    assert_eq!(
        fixture.drafts.acknowledge(&new, Ok(())),
        GuiParameterAck::Obsolete
    );
    fixture.drafts.synchronize(None).unwrap();
    assert!(fixture.drafts.text_buffer(fixture.target, "4 cm").is_err());
}
#[test]
fn each_text_field_has_its_own_buffer_and_commit_identity() {
    let mut fixture = ParameterDraftFixture::new();
    let equation = fixture.edit("6 cm");
    fixture.target.field = GuiParameterTextField::Comment;
    let comment = fixture.edit("é🙂");
    assert_ne!(equation.target, comment.target);
    assert_ne!(equation.request, comment.request);
    assert_eq!(fixture.drafts.text(equation.target), Some("6 cm"));
    assert_eq!(fixture.drafts.text(comment.target), Some("é🙂"));
}
#[test]
fn draft_budgets_nul_and_request_exhaustion_fail_before_admission() {
    let mut fixture = ParameterDraftFixture::new();
    assert!(GuiParameterDrafts::new(0, 1).is_err());
    assert!(GuiParameterDrafts::new(1, 0).is_err());
    assert!(fixture.drafts.text_buffer(fixture.target, "a\0b").is_err());
    *fixture.drafts.text_buffer(fixture.target, "4 cm").unwrap() = "x".repeat(129);
    assert!(fixture.drafts.submit(fixture.target).is_err());
    *fixture.drafts.text_buffer(fixture.target, "4 cm").unwrap() = "6 cm".into();
    fixture.drafts.next_request = u64::MAX;
    assert!(
        fixture
            .drafts
            .submit(fixture.target)
            .unwrap_err()
            .contains("request 18446744073709551615")
    );
    assert_eq!(fixture.drafts.text(fixture.target), Some("6 cm"));
}
#[test]
fn full_cell_budget_preserves_existing_text_and_invalid_snapshots_are_atomic() {
    let mut fixture = ParameterDraftFixture::new();
    fixture.drafts.cell_limit = 1;
    *fixture.drafts.text_buffer(fixture.target, "4 cm").unwrap() = "edited".into();
    let mut second = fixture.target;
    second.field = GuiParameterTextField::Name;
    assert!(fixture.drafts.text_buffer(second, "width").is_err());
    fixture.snapshot.document = 0;
    assert!(fixture.drafts.synchronize(Some(&fixture.snapshot)).is_err());
    assert_eq!(fixture.drafts.text(fixture.target), Some("edited"));
    assert_eq!(
        fixture
            .drafts
            .submit(fixture.target)
            .unwrap()
            .unwrap()
            .value,
        "edited"
    );
}

#[test]
fn clean_cells_can_be_reseeded_but_pending_and_failed_cells_cannot_be_evicted() {
    let mut fixture = ParameterDraftFixture::new();
    fixture.drafts.cell_limit = 1;
    fixture.drafts.text_buffer(fixture.target, "4 cm").unwrap();
    let mut second = fixture.target;
    second.field = GuiParameterTextField::Name;
    fixture.drafts.text_buffer(second, "width").unwrap();
    assert!(fixture.drafts.text(fixture.target).is_none());
    let edit = fixture.edit("6 cm");
    assert!(fixture.drafts.text_buffer(second, "width").is_err());
    fixture
        .drafts
        .acknowledge(&edit, Err("formula: expected a valid expression".into()));
    assert!(fixture.drafts.text_buffer(second, "width").is_err());
    assert_eq!(fixture.drafts.text(fixture.target), Some("6 cm"));
    let prior = fixture.drafts.text(fixture.target).map(str::to_owned);
    assert!(fixture.drafts.text_buffer(second, "bad\0seed").is_err());
    assert_eq!(
        fixture.drafts.text(fixture.target).map(str::to_owned),
        prior
    );
}
