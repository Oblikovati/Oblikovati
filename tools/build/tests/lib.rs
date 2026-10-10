// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn command_interface_rejects_ambiguous_input_and_prints_versions() {
    let tree = FixtureTree::new();
    let runner = FakeRunner::default();
    assert!(dispatch(&[], tree.root(), &runner).is_err());
    assert!(dispatch(&["preflight", "skip"], tree.root(), &runner).is_err());
    dispatch(&["matrix"], tree.root(), &runner).unwrap();
    tree.put("release.toml", include_str!("../../../release.toml"));
    dispatch(&["version", "stable"], tree.root(), &runner).unwrap();
    assert!(dispatch(&["version", "invalid"], tree.root(), &runner).is_err());
    assert!(dispatch(&["preflight", "publish"], tree.root(), &runner).is_err());
}
