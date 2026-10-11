// SPDX-License-Identifier: GPL-2.0-only

use super::*;

pub(crate) fn config() -> ReleaseConfig {
    toml::from_str(include_str!("fixtures/release.toml")).unwrap()
}

#[test]
fn repository_state_cannot_publish_an_empty_port() {
    let config: ReleaseConfig = toml::from_str(include_str!("../../../release.toml")).unwrap();
    assert!(publication_ready(&config).unwrap_err().contains("R9"));
}

#[test]
fn fixture_pins_and_compatibility_fields_require_complete_values() {
    let mut config = config();
    assert!(validate_pin(&config.api).is_err());
    assert!(validate_pin(&config.kernel).is_err());
    config.api.revision = "a".repeat(40);
    assert!(validate_pin(&config.api).is_ok());
    config.api.repository = "other/project".into();
    assert!(validate_pin(&config.api).is_err());
    config.product.publication_ready = true;
    assert!(
        publication_ready(&config)
            .unwrap_err()
            .contains("reference-key")
    );
    config.product.reference_key = "OKRK-verified".into();
    config.product.c_abi = "verified".into();
    assert!(publication_ready(&config).is_ok());
    assert!(!hex_digest(&"A".repeat(40), 40));
}

#[test]
fn application_requires_both_named_binary_targets() {
    use crate::{
        cargo_tree::{Target, tests::package},
        tree::tests::FixtureTree,
    };
    let tree = FixtureTree::new();
    let mut workspace = Metadata {
        workspace_root: tree.root().into(),
        workspace_members: vec![],
        packages: vec![],
    };
    assert!(
        require_application(&[])
            .unwrap_err()
            .contains("oblikovati-cli")
    );
    for name in ["oblikovati-cli", "oblikovati-head"] {
        let mut member = package(&tree);
        member.name = name.into();
        member.targets.push(Target {
            name: name.into(),
            kind: vec!["bin".into()],
        });
        workspace.packages.push(member);
    }
    assert!(require_application(&[workspace]).is_ok());
    tree.put("release.toml", include_str!("../../../release.toml"));
    assert!(load(tree.root()).is_ok());
    tree.put("release.toml", "invalid");
    assert!(load(tree.root()).is_err());
}
