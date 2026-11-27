// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchTextFieldSpan;

fn adjacent_fields() -> SketchTextFieldProjection {
    SketchTextFieldProjection {
        content: "界10.0010.00z".into(),
        fields: vec![
            SketchTextFieldSpan {
                authored: 3..6,
                display: 3..8,
            },
            SketchTextFieldSpan {
                authored: 6..9,
                display: 8..13,
            },
        ],
    }
}

#[test]
fn selections_expand_to_whole_fields_and_keep_adjacent_edges_distinct() {
    let view = adjacent_fields();
    assert_eq!(authored_selection(&view, 4..9, false), 3..9);
    assert_eq!(authored_selection(&view, 8..8, true), 6..6);
    assert_eq!(authored_selection(&view, 9..9, true), 9..9);
    assert_eq!(authored_selection(&view, 9..9, false), 6..6);
    assert_eq!(project_position(&view, 9), 13);
}

#[test]
fn backspace_between_equal_fields_removes_only_the_first_field() {
    let view = adjacent_fields();
    let edit = projected_field_edit(&view, "界10.010.00z", 8..8, 7).unwrap();
    assert_eq!(edit.authored, 3..6);
    assert_eq!(edit.input, 3..7);
    assert_eq!(edit.replacement, "");
}

#[test]
fn typing_after_a_field_leaves_both_markers_in_the_source() {
    let view = adjacent_fields();
    let edit = projected_field_edit(&view, "界10.0010.00応z", 13..13, 16).unwrap();
    assert_eq!(edit.authored, 9..9);
    assert_eq!(edit.input, 13..16);
    assert_eq!(edit.replacement, "応");
}

#[test]
fn replacement_over_two_partial_fields_removes_both_markers() {
    let view = adjacent_fields();
    let edit = projected_field_edit(&view, "界1応.00z", 4..10, 7).unwrap();
    assert_eq!(edit.authored, 3..9);
    assert_eq!(edit.input, 3..10);
    assert_eq!(edit.replacement, "応");
}
