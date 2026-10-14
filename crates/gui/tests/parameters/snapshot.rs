// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::ParameterInfo;
struct ParameterSnapshotFixture {
    snapshot: GuiParameterSnapshot,
}
impl ParameterSnapshotFixture {
    fn new() -> Self {
        let detail = ParameterDetail {
            parameter_info: ParameterInfo {
                name: "width".into(),
                kind: "user".into(),
                expression: "4 cm".into(),
                value: "40 mm".into(),
                ..Default::default()
            },
            units: "mm".into(),
            model_value: 4.0,
            ..Default::default()
        };
        Self {
            snapshot: GuiParameterSnapshot {
                linked_tables: Vec::new(),
                link_sources: Vec::new(),
                groups: Vec::new(),
                document: 7,
                rows: vec![GuiParameterRow {
                    tolerance_text: String::new(),
                    has_groups: false,
                    parameter: 11,
                    detail,
                }],
            },
        }
    }
}
#[test]
fn canonical_parameter_details_keep_owned_identity_and_unicode_values() {
    let mut fixture = ParameterSnapshotFixture::new();
    fixture.snapshot.rows[0].detail.comment = "é🙂".into();
    fixture.snapshot.validate().unwrap();
    let captured = fixture.snapshot.clone();
    fixture.snapshot.rows[0].detail.parameter_info.name = "span".into();
    assert_eq!(captured.rows[0].parameter, 11);
    assert_eq!(captured.rows[0].detail.parameter_info.name, "width");
    assert_eq!(captured.rows[0].detail.comment, "é🙂");
}
#[test]
fn document_and_row_identity_conflicts_are_named_declines() {
    let mut fixture = ParameterSnapshotFixture::new();
    fixture.snapshot.document = 0;
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("document 0")
    );
    fixture.snapshot.document = 7;
    fixture.snapshot.rows.push(fixture.snapshot.rows[0].clone());
    assert!(fixture.snapshot.validate().unwrap_err().contains("ID 11"));
}
#[test]
fn nul_and_nonfinite_values_are_declined_before_widget_calls() {
    let mut fixture = ParameterSnapshotFixture::new();
    fixture.snapshot.rows[0].detail.parameter_info.expression = "4\0cm".into();
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("equation, NUL at byte 1")
    );
    fixture.snapshot.rows[0].detail.parameter_info.expression = "4 cm".into();
    fixture.snapshot.rows[0].detail.model_value = f64::NAN;
    assert!(fixture.snapshot.validate().unwrap_err().contains("finite"));
}
#[test]
fn empty_views_and_failed_parameters_remain_inspectable() {
    let mut fixture = ParameterSnapshotFixture::new();
    fixture.snapshot.rows[0].detail.parameter_info.health = "Unresolved parameter".into();
    fixture.snapshot.validate().unwrap();
    fixture.snapshot.rows.clear();
    fixture.snapshot.validate().unwrap();
}
#[test]
fn groups_require_current_members_and_safe_presentation_text() {
    let mut fixture = ParameterSnapshotFixture::new();
    fixture.snapshot.groups.push(GuiParameterGroup {
        key: "ratios".into(),
        display_name: "Gear ratios".into(),
        client_id: "addin".into(),
        members: vec![11],
    });
    fixture.snapshot.validate().unwrap();
    fixture.snapshot.groups[0].members.push(12);
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("member 12")
    );
    fixture.snapshot.groups[0].members = vec![11];
    fixture.snapshot.groups[0].display_name = "\0".into();
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("display name")
    );
}
#[test]
fn value_list_choices_are_validated_as_widget_text() {
    use oblikovati_api::wire::ExpressionListInfo;
    let mut fixture = ParameterSnapshotFixture::new();
    fixture.snapshot.rows[0].detail.expression_list = Some(Box::new(ExpressionListInfo {
        expressions: Some(vec!["4 cm".into(), "6 cm".into()]),
        ..Default::default()
    }));
    fixture.snapshot.validate().unwrap();
    fixture.snapshot.rows[0]
        .detail
        .expression_list
        .as_mut()
        .unwrap()
        .expressions = Some(vec!["4\0cm".into()]);
    assert!(fixture.snapshot.validate().unwrap_err().contains("choice"));
}

#[test]
fn parameter_row_editability_includes_tables_and_refuses_reference_derived_and_unknown_kinds() {
    for (kind, expected) in [
        ("user", true),
        ("model", true),
        ("table", true),
        ("reference", false),
        ("derived", false),
        ("unknown", false),
    ] {
        let mut row = GuiParameterRow {
            tolerance_text: String::new(),
            has_groups: false,
            parameter: 0,
            detail: Default::default(),
        };
        row.detail.parameter_info.kind = kind.into();
        assert_eq!(row.is_editable(), expected, "parameter kind {kind}");
    }
}

#[test]
fn parameter_view_changes_detect_signed_zero_in_model_values_and_tolerance_bounds() {
    let original = ParameterSnapshotFixture::new().snapshot;
    let mut positive = original.clone();
    positive.rows[0].detail.model_value = 0.;
    let mut negative = positive.clone();
    negative.rows[0].detail.model_value = -0.;
    assert_ne!(positive, negative);
    positive.rows[0].detail.tolerance =
        Some(Box::new(oblikovati_api::wire::ToleranceInfo::default()));
    negative = positive.clone();
    negative.rows[0].detail.tolerance.as_mut().unwrap().lower = -0.;
    assert_ne!(positive, negative);
    negative = positive.clone();
    negative.rows[0].detail.tolerance.as_mut().unwrap().upper = -0.;
    assert_ne!(positive, negative);
    assert_eq!(positive, positive.clone());
}

#[test]
fn tolerance_bit_comparison_preserves_absence_and_both_bound_signs() {
    let positive = ToleranceInfo::default();
    let mut negative = positive.clone();
    negative.lower = -0.;
    assert!(matching_tolerance_bits(None, None));
    assert!(!matching_tolerance_bits(None, Some(&positive)));
    assert!(!matching_tolerance_bits(Some(&positive), None));
    assert!(matching_tolerance_bits(Some(&positive), Some(&positive)));
    assert!(!matching_tolerance_bits(Some(&positive), Some(&negative)));
}
#[test]
fn tolerance_presentation_changes_request_a_frame_and_nul_is_refused() {
    let original = ParameterSnapshotFixture::new().snapshot;
    let mut changed = original.clone();
    changed.rows[0].tolerance_text = "2 mm / -1 mm".into();
    assert_ne!(changed, original);
    changed.validate().unwrap();
    changed.rows[0].tolerance_text = "bad\0band".into();
    assert!(changed.validate().unwrap_err().contains("field tolerance"));
}
#[test]
fn linked_tables_use_canonical_dtos_and_refuse_invalid_ids_and_unsafe_labels() {
    let mut fixture = ParameterSnapshotFixture::new();
    let table = oblikovati_api::wire::DerivedParameterTableInfo {
        id: 1,
        source_document: "source.opd".into(),
        linked: Some(vec!["width".into()]),
        ..Default::default()
    };
    fixture.snapshot.linked_tables = vec![table.clone()];
    fixture.snapshot.validate().unwrap();
    fixture.snapshot.linked_tables.push(table);
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("unique positive")
    );
    fixture.snapshot.linked_tables.pop();
    fixture.snapshot.linked_tables[0].id = 0;
    assert!(fixture.snapshot.validate().is_err());
    fixture.snapshot.linked_tables[0].id = 1;
    fixture.snapshot.linked_tables[0].health = "bad\0health".into();
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("NUL-free")
    );
}
#[test]
fn link_sources_keep_stable_identities_and_refuse_target_or_unsafe_candidates() {
    let mut fixture = ParameterSnapshotFixture::new();
    fixture.snapshot.link_sources = vec![GuiParameterLinkSource {
        document: 8,
        source_document: "source part".into(),
        display_name: "Source part".into(),
        parameters: vec![GuiParameterLinkParameter {
            parameter: 0,
            name: "width".into(),
        }],
    }];
    fixture.snapshot.validate().unwrap();
    fixture.snapshot.link_sources[0].document = 7;
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("non-target")
    );
    fixture.snapshot.link_sources[0].document = 8;
    fixture.snapshot.link_sources[0].parameters[0].name = "bad\0name".into();
    assert!(
        fixture
            .snapshot
            .validate()
            .unwrap_err()
            .contains("NUL-free")
    );
}
