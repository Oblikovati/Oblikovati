// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, release_config::tests::config, tree::tests::FixtureTree};

#[test]
fn renderer_pin_rejects_local_and_changed_sources() {
    let config: crate::release_config::ReleaseConfig =
        toml::from_str(include_str!("../../../release.toml")).unwrap();
    let mut package = LockedPackage {
        name: "martensite-vulkan".into(),
        version: "0.7.0".into(),
        source: None,
    };
    assert!(verify_locked(&package, &config.kernel).is_err());
    package.source = Some("git+https://github.com/Oblikovati/Martensite#changed".into());
    assert!(verify_locked(&package, &config.kernel).is_err());
    let tree = FixtureTree::new();
    tree.put("Cargo.lock", include_str!("../../../Cargo.lock"));
    require_renderer_toolkits(tree.root()).unwrap();
    tree.put("Cargo.lock", "package = []");
    assert!(require_renderer_toolkits(tree.root()).is_err());
}

#[test]
fn application_lock_consumes_the_pinned_kernel_modeling_release() {
    let tree = FixtureTree::new();
    tree.put("release.toml", include_str!("../../../release.toml"));
    tree.put("Cargo.lock", include_str!("../../../Cargo.lock"));
    let pin = crate::release_config::load(tree.root()).unwrap().kernel;
    let names = locked_crates(&tree.root().join("Cargo.lock"), &pin).unwrap();
    assert_eq!(pin.version, "0.7.0");
    assert_eq!(pin.revision, "01a9d38268fd937e36a083520080c0f13a8a94eb");
    for name in [
        "martensite-tkprim",
        "martensite-tkbrep",
        "martensite-tkgeombase",
        "martensite-tktopalgo",
    ] {
        assert!(names.iter().any(|locked| locked == name), "missing {name}");
    }
}

#[test]
fn mixed_kernel_versions_and_test_oracle_have_distinct_pins() {
    let tree = FixtureTree::new();
    tree.put("release.toml", include_str!("../../../release.toml"));
    tree.put("Cargo.lock", include_str!("../../../Cargo.lock"));
    check_pins(tree.root()).unwrap();
    let config = crate::release_config::load(tree.root()).unwrap();
    let oracle = config.kernel_oracle.unwrap();
    assert_eq!(oracle.repository, "Oblikovati/RustyCADKernel");
    assert!(!owned_package("rustycad-tkbo", &config.kernel));
    assert!(owned_package("rustycad-tkbo", &oracle));
}

#[test]
fn kernel_lockfile_uses_each_toolkit_version() {
    let config: crate::release_config::ReleaseConfig =
        toml::from_str(include_str!("../../../release.toml")).unwrap();
    let lock: CargoLock = toml::from_str(include_str!("../../../Cargo.lock")).unwrap();
    let mut locked = lock
        .package
        .into_iter()
        .find(|package| package.name == "martensite-tkprim")
        .unwrap();
    verify_locked(&locked, &config.kernel).unwrap();
    locked.version = "0.7.0".into();
    assert!(verify_locked(&locked, &config.kernel).is_err());
}

#[test]
fn release_assets_use_each_toolkit_version() {
    let dependency = heterogeneous_kernel_pin();
    validate_pin(&dependency).unwrap();
    let release = UpstreamRelease {
        draft: false,
        prerelease: false,
        assets: vec![ReleaseAsset {
            name: "martensite-tkprim-0.3.7.crate".into(),
        }],
    };
    require_assets(&release, &dependency, &["martensite-tkprim".into()]).unwrap();
    assert!(require_assets(&release, &dependency, &["martensite-tkmath".into()]).is_err());
}

fn heterogeneous_kernel_pin() -> ReleasedDependency {
    let mut dependency = config().kernel;
    dependency.repository = "Oblikovati/Martensite".into();
    dependency.revision = "a".repeat(40);
    dependency.version = "0.7.0".into();
    dependency
        .packages
        .insert("martensite-tkprim".into(), "0.3.7".into());
    dependency
}

#[test]
fn incomplete_or_invalid_toolkit_maps_are_refused() {
    let mut dependency = heterogeneous_kernel_pin();
    dependency.packages.clear();
    assert!(validate_pin(&dependency).is_err());
    for (name, version) in [
        ("wrong-package", "0.3.7"),
        ("martensite-tkprim", "wrong-version"),
    ] {
        dependency.packages.clear();
        dependency.packages.insert(name.into(), version.into());
        assert!(validate_pin(&dependency).is_err());
    }
}

#[test]
fn published_archive_and_tag_must_both_match_the_pin() {
    let tree = FixtureTree::new();
    let mut dependency = config().api;
    dependency.revision = "a".repeat(40);
    let runner = FakeRunner::default();
    runner.replies.borrow_mut().extend([Ok(r#"{"draft":false,"prerelease":false,"assets":[{"name":"oblikovati-api-0.154.0.crate"}]}"#.into()), Ok(format!("{}\trefs/tags/v0.154.0\n", dependency.revision))]);
    verify_release(
        tree.root(),
        &runner,
        &dependency,
        &["oblikovati-api".into()],
    )
    .unwrap();
    assert_eq!(
        release_commit(
            "tagobject\trefs/tags/v1\ncommit\trefs/tags/v1^{}",
            "refs/tags/v1"
        )
        .unwrap(),
        "commit"
    );
    assert!(release_commit("", "missing").is_err());
    assert!(
        require_assets(
            &UpstreamRelease {
                draft: false,
                prerelease: false,
                assets: vec![]
            },
            &dependency,
            &["oblikovati-api".into()]
        )
        .is_err()
    );
    assert!(
        require_assets(
            &UpstreamRelease {
                draft: true,
                prerelease: false,
                assets: vec![]
            },
            &dependency,
            &[]
        )
        .is_err()
    );
    assert!(
        verify_release(
            tree.root(),
            &runner,
            &dependency,
            &["oblikovati-api".into()]
        )
        .is_err()
    );
}

#[test]
fn lockfile_cannot_substitute_a_local_or_different_dependency() {
    let tree = FixtureTree::new();
    let mut dependency = config().api;
    dependency.revision = "a".repeat(40);
    let source = format!(
        "git+https://github.com/{}.git?tag=v{}#{}",
        dependency.repository, dependency.version, dependency.revision
    );
    tree.put(
        "Cargo.lock",
        &format!(
            "[[package]]\nname = 'oblikovati-api'\nversion = '0.154.0'\nsource = '{source}'\n"
        ),
    );
    assert_eq!(
        locked_crates(&tree.root().join("Cargo.lock"), &dependency).unwrap(),
        ["oblikovati-api"]
    );
    let package = LockedPackage {
        name: "oblikovati-api".into(),
        version: "0.154.0".into(),
        source: None,
    };
    assert!(verify_locked(&package, &dependency).is_err());
    tree.put("Cargo.lock", "package = []");
    assert!(locked_crates(&tree.root().join("Cargo.lock"), &dependency).is_err());
    tree.put("Cargo.lock", "malformed");
    assert!(locked_crates(&tree.root().join("Cargo.lock"), &dependency).is_err());
    assert!(owned_package("rustycad-tkbrep", &config().kernel));
    assert!(!owned_package("serde", &config().kernel));
}

#[test]
fn application_lock_consumes_the_api_release_named_by_packaging() {
    let tree = FixtureTree::new();
    tree.put("release.toml", include_str!("../../../release.toml"));
    tree.put("Cargo.lock", include_str!("../../../Cargo.lock"));
    let pin = crate::release_config::load(tree.root()).unwrap().api;
    let names = locked_crates(&tree.root().join("Cargo.lock"), &pin).unwrap();
    assert_eq!(names, ["oblikovati-api"]);
    assert_eq!(pin.version, "0.158.0");
    assert_eq!(pin.revision, "32ff38398ab6d6a7df0ca838654d5b8bc8d8b9cd");
}
