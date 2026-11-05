// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

struct NativeRuntimeFixture {
    files: FixtureTree,
    metadata: serde_json::Value,
}
impl NativeRuntimeFixture {
    fn new() -> Self {
        let files = FixtureTree::new();
        files.put("release.toml", include_str!("../../../release.toml"));
        files.put("Cargo.lock", include_str!("../../../Cargo.lock"));
        files.put("sdk/scripts/install/vulkan-validation.sh", "");
        let lock: toml::Value = toml::from_str(include_str!("../../../Cargo.lock")).unwrap();
        let pin = lock["package"]
            .as_array()
            .unwrap()
            .iter()
            .find(|package| package["name"].as_str() == Some("martensite-vulkan"))
            .unwrap();
        let metadata = serde_json::json!({"packages": [{"name": "martensite-vulkan", "version": pin["version"].as_str().unwrap(), "source": pin["source"].as_str().unwrap(), "manifest_path": files.root().join("sdk/crates/martensite-vulkan/Cargo.toml")}]});
        Self { files, metadata }
    }
    fn runner(&self) -> FakeRunner {
        let runner = FakeRunner::default();
        runner.replies.borrow_mut().extend([
            Ok(self.metadata.to_string()),
            Ok("runtime installed\n".into()),
        ]);
        runner
    }
    fn assert_declined_before_installation(&self) {
        let runner = self.runner();
        let message = prepare(
            self.files.root(),
            &runner,
            Path::new("cache"),
            Path::new("environment"),
        )
        .unwrap_err();
        assert!(message.contains("expected locked"));
        assert_eq!(runner.calls.borrow().len(), 1);
    }
}

#[test]
fn native_runtime_uses_only_the_locked_sdk_installer_and_explicit_destinations() {
    let fixture = NativeRuntimeFixture::new();
    let runner = fixture.runner();
    crate::dispatch(
        &["native-validation", "cache directory", "github environment"],
        fixture.files.root(),
        &runner,
    )
    .unwrap();
    let calls = runner.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[1][0], "bash");
    assert!(Path::new(&calls[1][1]).ends_with("sdk/scripts/install/vulkan-validation.sh"));
    assert_eq!(
        &calls[1][2..],
        ["cache directory", "--github-env", "github environment"]
    );
}

#[test]
fn a_changed_or_local_sdk_cannot_execute_its_installer() {
    for (field, value) in [
        ("source", serde_json::Value::Null),
        (
            "source",
            serde_json::json!("git+https://other.test/sdk#changed"),
        ),
        ("version", serde_json::json!("99.0.0")),
    ] {
        let mut fixture = NativeRuntimeFixture::new();
        fixture.metadata["packages"][0][field] = value;
        fixture.assert_declined_before_installation();
    }
}

#[test]
fn missing_installer_and_invalid_manifest_layout_are_named_declines() {
    let fixture = FixtureTree::new();
    let manifest = fixture.root().join("crates/martensite-vulkan/Cargo.toml");
    let message = installer_from_manifest(&manifest).unwrap_err();
    assert!(message.contains("expected") && message.contains("vulkan-validation.sh"));
    assert!(installer_from_manifest(Path::new("Cargo.toml")).is_err());
}

#[test]
fn a_fetched_package_must_exist_in_the_lockfile() {
    let fixture = NativeRuntimeFixture::new();
    let package = crate::fetched_source::FetchedPackage {
        name: "missing-runtime".into(),
        version: "1.0.0".into(),
        source: None,
        manifest_path: PathBuf::from("unknown/Cargo.toml"),
    };
    assert!(
        dependencies::require_locked_source(fixture.files.root(), &package)
            .unwrap_err()
            .contains("expected a lockfile package")
    );
}
