// SPDX-License-Identifier: GPL-2.0-only
//! What the Find dialog searches with: criteria, how they join, and whether a row meets them.

const ANY_VALUE: &str = "any value";
const BEGINS_WITH: &str = "begins with";
const ENDS_WITH: &str = "ends with";
const INCLUDES: &str = "includes";
const IS_EXACTLY: &str = "is exactly";
const IS_NOT: &str = "is not";
const EQUALS: &str = "equals";
const CONTAINS: &str = "contains";
const DOES_NOT_CONTAIN: &str = "does not contain";
pub(crate) const NAME: &str = "Name";
const NOTE_CONTENTS: &str = "Note contents";
const NOTE_TITLE: &str = "Note title";
pub(crate) const SUPPRESS_STATUS: &str = "Suppress status";
pub(crate) const TYPE: &str = "Type";
pub(crate) const SKETCH_PARAMETER: &str = "Sketch Parameter Name";
const SUPPRESSED: &str = "Suppressed";
const UNSUPPRESSED: &str = "Unsuppressed";
const FEATURE_PROPERTIES: [&str; 5] = [NAME, NOTE_CONTENTS, NOTE_TITLE, SUPPRESS_STATUS, TYPE];
const SKETCH_PROPERTIES: [&str; 4] = [NAME, NOTE_CONTENTS, NOTE_TITLE, SKETCH_PARAMETER];
const TEXT_CONDITIONS: [&str; 6] = [
    ANY_VALUE,
    BEGINS_WITH,
    ENDS_WITH,
    INCLUDES,
    IS_EXACTLY,
    IS_NOT,
];
const CHOICE_CONDITIONS: [&str; 1] = [EQUALS];
const PARAMETER_CONDITIONS: [&str; 2] = [CONTAINS, DOES_NOT_CONTAIN];
const SUPPRESS_VALUES: [&str; 2] = [SUPPRESSED, UNSUPPRESSED];
const TYPE_VALUES: [&str; 7] = [
    "Extrusion",
    "Fillet",
    "Hole",
    "Chamfer",
    "Revolution",
    "Sweep",
    "Work Feature",
];

/// What kind of row a search looks at.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum FindObject {
    #[default]
    Features,
    Sketches,
}
impl FindObject {
    pub(crate) const ALL: [(Self, &'static str); 2] =
        [(Self::Features, "Features"), (Self::Sketches, "Sketches")];
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::Features => "Features",
            Self::Sketches => "Sketches",
        }
    }
    pub(crate) fn properties(self) -> &'static [&'static str] {
        match self {
            Self::Features => &FEATURE_PROPERTIES,
            Self::Sketches => &SKETCH_PROPERTIES,
        }
    }
}
/// The conditions a property offers; the first of its list the dialog starts on differs.
pub(crate) fn conditions_of(property: &str) -> &'static [&'static str] {
    match property {
        TYPE | SUPPRESS_STATUS => &CHOICE_CONDITIONS,
        SKETCH_PARAMETER => &PARAMETER_CONDITIONS,
        _ => &TEXT_CONDITIONS,
    }
}
/// The condition a property starts with; a text property most often means its start.
pub(crate) fn first_condition_of(property: &str) -> &'static str {
    match property {
        TYPE | SUPPRESS_STATUS => EQUALS,
        SKETCH_PARAMETER => CONTAINS,
        _ => BEGINS_WITH,
    }
}
/// The fixed values a property chooses from; `None` means typed text.
pub(crate) fn value_choices_of(property: &str) -> Option<&'static [&'static str]> {
    match property {
        TYPE => Some(&TYPE_VALUES),
        SUPPRESS_STATUS => Some(&SUPPRESS_VALUES),
        _ => None,
    }
}
/// The value a property starts with; a typed one starts empty.
pub(crate) fn first_value_of(property: &str) -> String {
    value_choices_of(property)
        .map(|choices| choices[0].to_owned())
        .unwrap_or_default()
}
/// How a criterion joins the one before it.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum FindJoin {
    #[default]
    And,
    Or,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct FindCriterion {
    pub property: &'static str,
    pub condition: &'static str,
    pub value: String,
    pub join: FindJoin,
}
impl FindCriterion {
    /// A list row: `OR` marks a criterion grouped with the one above it.
    /// Example: `OR Type equals Fillet`.
    pub(crate) fn row_text(&self) -> String {
        let prefix = if self.join == FindJoin::Or { "OR " } else { "" };
        format!(
            "{prefix}{} {} {}",
            self.property, self.condition, self.value
        )
        .trim_end()
        .to_owned()
    }
    /// True when a row's facts meet this criterion.
    fn admits(&self, row: &FindRow, match_case: bool) -> bool {
        let wanted = fold_case(&self.value, match_case);
        match self.property {
            TYPE => row.kind == self.value,
            SUPPRESS_STATUS => row.suppressed == (self.value == SUPPRESSED),
            SKETCH_PARAMETER => {
                let found = row
                    .parameters
                    .iter()
                    .any(|name| fold_case(name, match_case).contains(&wanted));
                found == (self.condition == CONTAINS)
            }
            NAME => text_admits(self.condition, &fold_case(row.name, match_case), &wanted),
            // Nothing in this build carries a note, so only "is not" can hold.
            _ => text_admits(self.condition, "", &wanted),
        }
    }
}
fn fold_case(text: &str, match_case: bool) -> String {
    if match_case {
        return text.to_owned();
    }
    text.to_lowercase()
}
fn text_admits(condition: &str, text: &str, wanted: &str) -> bool {
    match condition {
        ANY_VALUE => !text.is_empty(),
        BEGINS_WITH => !wanted.is_empty() && text.starts_with(wanted),
        ENDS_WITH => !wanted.is_empty() && text.ends_with(wanted),
        INCLUDES => !wanted.is_empty() && text.contains(wanted),
        IS_EXACTLY => text == wanted,
        _ => text != wanted,
    }
}
/// What a search can read about one browser row.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub(crate) struct FindRow<'a> {
    pub name: &'a str,
    pub kind: &'a str,
    pub suppressed: bool,
    pub parameters: Vec<&'a str>,
}
/// `OR` binds a criterion to the one above it, and each such group must hold.
pub(crate) fn row_meets(criteria: &[FindCriterion], row: &FindRow, match_case: bool) -> bool {
    let mut groups = Vec::new();
    for criterion in criteria {
        let holds = criterion.admits(row, match_case);
        match (criterion.join, groups.last_mut()) {
            (FindJoin::Or, Some(group)) => *group |= holds,
            _ => groups.push(holds),
        }
    }
    !groups.is_empty() && groups.into_iter().all(|group| group)
}
/// An And on a property already listed can never hold beside it: it has to be an Or.
pub(crate) fn needs_grouping(criteria: &[FindCriterion], added: &FindCriterion) -> bool {
    added.join == FindJoin::And && criteria.iter().any(|row| row.property == added.property)
}
/// Add a criterion as Or, directly under the last one on its property.
pub(crate) fn add_grouped(criteria: &mut Vec<FindCriterion>, mut added: FindCriterion) {
    added.join = FindJoin::Or;
    let after = criteria
        .iter()
        .rposition(|row| row.property == added.property)
        .map_or(criteria.len(), |place| place + 1);
    criteria.insert(after, added);
}

#[cfg(test)]
#[path = "../../../tests/application/model-browser/find_criteria.rs"]
mod tests;
