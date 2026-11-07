// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn criterion(property: &'static str, value: &str, join: FindJoin) -> FindCriterion {
    FindCriterion {
        property,
        condition: first_condition_of(property),
        value: value.into(),
        join,
    }
}
fn extrusion() -> FindRow<'static> {
    FindRow {
        name: "Extrusion1",
        kind: "Extrusion",
        ..Default::default()
    }
}
#[test]
fn a_property_starts_on_the_condition_and_value_it_most_often_means() {
    assert_eq!(first_condition_of(NAME), BEGINS_WITH);
    assert_eq!(first_condition_of(TYPE), EQUALS);
    assert_eq!(first_condition_of(SKETCH_PARAMETER), CONTAINS);
    assert_eq!(first_value_of(SUPPRESS_STATUS), SUPPRESSED);
    assert_eq!(first_value_of(NAME), "");
    assert_eq!(conditions_of(NAME).len(), 6);
    assert_eq!(conditions_of(SKETCH_PARAMETER), PARAMETER_CONDITIONS);
}
#[test]
fn or_groups_with_the_row_above_and_each_group_must_hold() {
    let grouped = [
        criterion(TYPE, "Fillet", FindJoin::And),
        criterion(TYPE, "Extrusion", FindJoin::Or),
    ];
    assert!(row_meets(&grouped, &extrusion(), false));
    let and_suppressed = [
        criterion(TYPE, "Extrusion", FindJoin::And),
        criterion(TYPE, "Fillet", FindJoin::Or),
        criterion(SUPPRESS_STATUS, SUPPRESSED, FindJoin::And),
    ];
    assert!(!row_meets(&and_suppressed, &extrusion(), false));
    assert!(!row_meets(&[], &extrusion(), false));
}
#[test]
fn text_conditions_read_the_name_with_or_without_case() {
    let name = |condition, value: &str| FindCriterion {
        property: NAME,
        condition,
        value: value.into(),
        join: FindJoin::And,
    };
    let row = extrusion();
    assert!(name(BEGINS_WITH, "extr").admits(&row, false));
    assert!(!name(BEGINS_WITH, "extr").admits(&row, true));
    assert!(name(ENDS_WITH, "1").admits(&row, false));
    assert!(name(INCLUDES, "rus").admits(&row, false));
    assert!(name(IS_EXACTLY, "Extrusion1").admits(&row, true));
    assert!(name(IS_NOT, "Other").admits(&row, false));
    assert!(name(ANY_VALUE, "").admits(&row, false));
}
#[test]
fn a_rows_note_is_empty_so_only_is_not_holds() {
    let note = |condition| FindCriterion {
        property: NOTE_TITLE,
        condition,
        value: "a".into(),
        join: FindJoin::And,
    };
    assert!(!note(INCLUDES).admits(&extrusion(), false));
    assert!(note(IS_NOT).admits(&extrusion(), false));
}
#[test]
fn a_sketch_parameter_condition_reads_each_name() {
    let row = FindRow {
        parameters: vec!["width", "depth"],
        ..Default::default()
    };
    let parameter = |condition, value: &str| FindCriterion {
        property: SKETCH_PARAMETER,
        condition,
        value: value.into(),
        join: FindJoin::And,
    };
    assert!(parameter(CONTAINS, "WID").admits(&row, false));
    assert!(parameter(DOES_NOT_CONTAIN, "height").admits(&row, false));
}
#[test]
fn a_second_and_on_a_property_needs_grouping_and_lands_under_its_group() {
    let mut list = vec![
        criterion(TYPE, "Extrusion", FindJoin::And),
        criterion(SUPPRESS_STATUS, SUPPRESSED, FindJoin::And),
    ];
    let hole = criterion(TYPE, "Hole", FindJoin::And);
    assert!(needs_grouping(&list, &hole));
    add_grouped(&mut list, hole);
    assert_eq!(list[1].row_text(), "OR Type equals Hole");
    assert_eq!(list[2].property, SUPPRESS_STATUS);
    assert_eq!(list[0].row_text(), "Type equals Extrusion");
}
