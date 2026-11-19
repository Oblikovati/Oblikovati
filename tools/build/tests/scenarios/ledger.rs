// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::{collections::BTreeMap, path::PathBuf};
#[test]
fn capture_only_is_inherited_from_feature_rule_and_scenario() {
    let source = "@capture_only\nFeature: captured\n Scenario: a\n  Given a step\n";
    let captured = inventory(Path::new("root"), REVISION, &mut fixture(source)).unwrap();
    assert!(captured.scenarios[0].capture_only);
    let source = "Feature: mixed\n @capture_only\n Scenario: a\n  Given a step\n Scenario: b\n  Given a step\n @capture_only\n Rule: archived\n  Scenario: c\n   Given a step\n";
    let mixed = inventory(Path::new("root"), REVISION, &mut fixture(source)).unwrap();
    assert_eq!(
        mixed
            .scenarios
            .iter()
            .map(|row| row.capture_only)
            .collect::<Vec<_>>(),
        [true, false, true]
    );
}
#[test]
fn capture_host_only_is_inherited_without_claiming_portability() {
    let source = "@capture_host_only\nFeature: host\n @capture_only\n Scenario: unavailable\n  Given a step\n Scenario: host fixture\n  Given a step\n";
    let ledger = inventory(Path::new("root"), REVISION, &mut fixture(source)).unwrap();
    assert!(ledger.scenarios.iter().all(|row| row.capture_host_only));
    assert!(ledger.scenarios[0].capture_only);
    assert!(!ledger.scenarios[1].capture_only);
}
#[derive(Default)]
struct FakeScenarioFiles {
    files: BTreeMap<PathBuf, Vec<u8>>,
}
impl ScenarioSource for FakeScenarioFiles {
    fn feature_paths(&mut self, _: &Path) -> BuildResult<Vec<PathBuf>> {
        Ok(self.files.keys().cloned().collect())
    }
    fn read(&mut self, path: &Path) -> BuildResult<Vec<u8>> {
        self.files
            .get(path)
            .cloned()
            .ok_or_else(|| format!("missing source {path:?}"))
    }
}
fn fixture(text: &str) -> FakeScenarioFiles {
    FakeScenarioFiles {
        files: BTreeMap::from([(
            PathBuf::from("root/features/application/folder/test.feature"),
            text.as_bytes().to_vec(),
        )]),
    }
}
const REVISION: &str = "35f4a77b1519e2f29005be009c35fb7c5bdf4e89";
#[test]
fn local_inventory_is_labeled_unversioned() {
    let source = "Feature: local\n Scenario: test\n  Given value\n";
    let ledger = local_inventory(Path::new("root"), &mut fixture(source)).unwrap();
    assert_eq!(ledger.source_revision, "unversioned");
    assert!(ledger.scenarios[0].arguments_recorded);
}
#[test]
fn records_step_tables_and_docstrings_in_background_and_scenario() {
    let text = "Feature: arguments\n Background:\n  Given settings\n   | key | value |\n   | a | b |\n Scenario: example\n  When submitting\n   \"\"\"\n   body\n   \"\"\"\n";
    let ledger = inventory(Path::new("root"), REVISION, &mut fixture(text)).unwrap();
    let row = &ledger.scenarios[0];
    assert!(row.arguments_recorded);
    assert_eq!(
        row.background_steps[0].table.as_ref().unwrap()[1],
        ["a", "b"]
    );
    assert_eq!(row.scenario_steps[0].docstring.as_deref(), Some("\nbody\n"));
}
#[test]
fn records_outline_once_and_keeps_identity_when_lines_move() {
    let text = "Feature: example\n Scenario: normal\n  Given a value\n Scenario Outline: template\n  When value <x>\n  Examples:\n   | x |\n   | one |\n   | two |\n";
    let first = inventory(Path::new("root"), REVISION, &mut fixture(text)).unwrap();
    let second = inventory(
        Path::new("root"),
        REVISION,
        &mut fixture(&format!("# heading\n{text}")),
    )
    .unwrap();
    assert_eq!(first.scenarios.len(), 2);
    assert!(!first.scenarios[0].outline && first.scenarios[1].outline);
    assert_eq!(first.scenarios[1].examples.len(), 2);
    assert_eq!(first.scenarios[1].examples[0].line, 8);
    assert_eq!(first.scenarios[1].examples[0].columns, ["x"]);
    assert_eq!(first.scenarios[1].examples[0].values, ["one"]);
    assert_eq!(first.scenarios[1].examples[1].line, 9);
    assert_eq!(first.scenarios[0].id, second.scenarios[0].id);
    assert_ne!(
        first.scenarios[0].source_sha256,
        second.scenarios[0].source_sha256
    );
    assert!(
        first
            .scenarios
            .iter()
            .all(|row| row.state == "inventoried" && row.last_receipt.is_none())
    );
}
#[test]
fn records_multiple_example_tables_and_unchecked_step_arguments() {
    let text = "Feature: cases\n Scenario Outline: <x>\n  Given choose <x>\n   | detail |\n   | value |\n  Examples: first\n   | x |\n   | one |\n  Examples: second\n   | x |\n   | two |\n";
    let ledger = inventory(Path::new("root"), REVISION, &mut fixture(text)).unwrap();
    let row = &ledger.scenarios[0];
    assert!(row.outline_step_arguments);
    assert_eq!(row.examples.len(), 2);
    assert_eq!(row.examples[0].line, 8);
    assert_eq!(row.examples[1].line, 11);
    assert_eq!(row.examples[1].values, ["two"]);
}
#[test]
fn an_outline_without_examples_is_not_misclassified_as_a_normal_scenario() {
    let text = "Feature: empty\n Scenario Outline: no examples\n  Given a step\n";
    let ledger = inventory(Path::new("root"), REVISION, &mut fixture(text)).unwrap();
    assert!(ledger.scenarios[0].outline);
    assert!(ledger.scenarios[0].examples.is_empty());
}
#[test]
fn refuses_duplicate_names_invalid_gherkin_and_invalid_revision() {
    let duplicate =
        "Feature: duplicate\n Scenario: same\n  Given first\n Scenario: same\n  Given second\n";
    assert!(
        inventory(Path::new("root"), REVISION, &mut fixture(duplicate))
            .unwrap_err()
            .contains("unique names")
    );
    assert!(inventory(Path::new("root"), REVISION, &mut fixture("not Gherkin")).is_err());
    assert!(inventory(Path::new("root"), "invalid", &mut fixture(duplicate)).is_err());
}

#[test]
fn invalid_source_keeps_declarations_blocked_and_ignores_docstrings() {
    let text = "Feature: broken\n Scenario: real\n  Given a step\n   \"\"\"\n   Scenario: not a scenario\n   \"\"\"\n  and invalid lowercase\n Scenario Outline: template\n  Given <x>\n  Examples:\n   | x |\n   | one |\n";
    let ledger = inventory(Path::new("root"), REVISION, &mut fixture(text)).unwrap();
    assert_eq!(ledger.scenarios.len(), 2);
    assert!(
        ledger
            .scenarios
            .iter()
            .all(|row| row.state == "blocked" && row.source_error.is_some())
    );
    assert_eq!(ledger.scenarios[0].scenario, "real");
    assert!(ledger.scenarios[1].outline);
}

#[test]
fn rejects_wrong_or_dirty_snapshot_revision() {
    let folder = tempfile::tempdir().unwrap();
    std::fs::create_dir(folder.path().join(".git")).unwrap();
    let runner = crate::process::tests::FakeRunner::default();
    runner
        .replies
        .borrow_mut()
        .push_back(Ok(format!("{REVISION}\n")));
    runner.replies.borrow_mut().push_back(Ok(String::new()));
    runner.replies.borrow_mut().push_back(Ok(String::new()));
    verify_checkout(folder.path(), REVISION, &runner).unwrap();
    runner.replies.borrow_mut().push_back(Ok("wrong\n".into()));
    assert!(
        verify_checkout(folder.path(), REVISION, &runner)
            .unwrap_err()
            .contains("expected checkout commit")
    );
    runner.replies.borrow_mut().push_back(Ok(REVISION.into()));
    runner
        .replies
        .borrow_mut()
        .push_back(Ok(" M features/a.feature\n".into()));
    runner.replies.borrow_mut().push_back(Ok(String::new()));
    assert!(
        verify_checkout(folder.path(), REVISION, &runner)
            .unwrap_err()
            .contains("expected clean source")
    );
}

struct ChangingScenarioFiles {
    reads: usize,
}
impl ScenarioSource for ChangingScenarioFiles {
    fn feature_paths(&mut self, root: &Path) -> BuildResult<Vec<PathBuf>> {
        Ok(vec![root.join("features/application/folder/test.feature")])
    }
    fn read(&mut self, _: &Path) -> BuildResult<Vec<u8>> {
        self.reads += 1;
        Ok(format!(
            "Feature: test\n Scenario: name\n  Given read {}\n",
            self.reads
        )
        .into_bytes())
    }
}
#[test]
fn rejects_a_source_that_changes_between_reads() {
    let error = inventory(
        Path::new("root"),
        REVISION,
        &mut ChangingScenarioFiles { reads: 0 },
    )
    .unwrap_err();
    assert!(error.contains("expected unchanged bytes"));
}

#[test]
fn records_feature_and_rule_backgrounds_with_the_scenario_steps() {
    let text = "Feature: test\n Background: shared\n  Given global setup\n Rule: constraints\n  Background: local\n   Given rule setup\n  Scenario: named\n   When action\n   Then outcome\n";
    let ledger = inventory(Path::new("root"), REVISION, &mut fixture(text)).unwrap();
    let row = &ledger.scenarios[0];
    assert_eq!(row.rule.as_deref(), Some("constraints"));
    assert_eq!(
        row.background_steps
            .iter()
            .map(|step| step.name.as_str())
            .collect::<Vec<_>>(),
        ["global setup", "rule setup"]
    );
    assert_eq!(
        row.scenario_steps
            .iter()
            .map(|step| step.name.as_str())
            .collect::<Vec<_>>(),
        ["action", "outcome"]
    );
}
