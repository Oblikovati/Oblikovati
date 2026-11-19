// SPDX-License-Identifier: GPL-2.0-only
//! Name scenario captures from stable source identity.
use super::scenario_root;
use std::fs;

pub(super) fn scenario_capture_namespace(
    feature: &cucumber::gherkin::Feature,
    scenario: &cucumber::gherkin::Scenario,
) -> String {
    use sha2::{Digest, Sha256};
    let path = fs::canonicalize(feature.path.as_ref().expect("scenario feature path"))
        .expect("scenario feature path must exist");
    let root = fs::canonicalize(scenario_root()).expect("scenario root must exist");
    let relative = path
        .strip_prefix(root)
        .expect("scenario feature must be inside scenario root");
    let identity = format!(
        "{}:{}:{}",
        relative.display(),
        scenario.position.line,
        scenario.name
    );
    format!("{:x}", Sha256::digest(identity.as_bytes()))
}
pub(super) fn capture_name(
    feature: &cucumber::gherkin::Feature,
    scenario: &cucumber::gherkin::Scenario,
) -> String {
    if scenario.examples.is_empty() {
        return scenario.name.clone();
    }
    use sha2::{Digest, Sha256};
    let path = fs::canonicalize(feature.path.as_ref().expect("outline feature path"))
        .expect("outline feature path must exist");
    let root = fs::canonicalize(scenario_root()).expect("scenario root must exist");
    let relative = path
        .strip_prefix(root)
        .expect("outline feature must be inside scenario root")
        .components()
        .map(|part| part.as_os_str().to_str().expect("UTF-8 feature path"))
        .collect::<Vec<_>>()
        .join("/");
    let digest = Sha256::digest(relative.as_bytes());
    format!("example-{digest:x}-{}", scenario.position.line)
}
