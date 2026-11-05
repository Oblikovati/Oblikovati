// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    packaging::tests::resources_fixture,
    platform::{Platform, platform},
    process::{BuildResult, Runner},
    release_config::{ReleaseConfig, tests::config},
    tree::{tests::FixtureTree, write_text},
};
use std::{cell::RefCell, path::Path};

pub(crate) const VERSION: &str = "0.0015400.1.0";

/// Named external-command fake; its binary and image bytes are test fixtures only.
pub(crate) struct ReleaseFixture {
    pub(crate) tree: FixtureTree,
    pub(crate) config: ReleaseConfig,
    pub(crate) platform: Platform,
    pub(crate) calls: RefCell<Vec<String>>,
}

impl ReleaseFixture {
    pub(crate) fn new(target: &str) -> Self {
        let fixture = Self {
            tree: FixtureTree::new(),
            config: released_config(),
            platform: platform(target).unwrap(),
            calls: RefCell::new(Vec::new()),
        };
        fixture.prepare_tree();
        fixture
    }

    fn prepare_tree(&self) {
        self.tree
            .put("release.toml", &toml::to_string(&self.config).unwrap());
        self.tree.put("Cargo.toml", "[workspace]\nmembers = []");
        self.tree.put("LICENSE", "GPL fixture");
        self.tree.put(
            &format!("packaging/smoke/{}.sh", self.platform.target),
            "fixture smoke program",
        );
        resources_fixture(&self.tree);
        let locks = [(&self.config.api, "oblikovati-api"), (&self.config.kernel, "rustycad-kernel")].iter().map(|(pin, name)| {
            format!("[[package]]\nname = '{name}'\nversion = '{}'\nsource = 'git+https://github.com/{}?tag=v{}#{}'\n", pin.version, pin.repository, pin.version, pin.revision)
        }).collect::<String>();
        self.tree.put("Cargo.lock", &locks);
        self.prepare_native_libraries();
    }

    fn prepare_native_libraries(&self) {
        for (formula, name) in [
            ("vulkan-loader", "libvulkan.1.dylib"),
            ("molten-vk", "libMoltenVK.dylib"),
            ("glfw", "libglfw.3.dylib"),
        ] {
            self.tree.put(
                &format!("native/{formula}/lib/{name}"),
                "fixture native dylib",
            );
        }
    }

    fn metadata(&self) -> String {
        let packages = ["oblikovati-cli", "oblikovati-head"].map(|name| serde_json::json!({
            "id": name, "name": name, "manifest_path": self.tree.root().join("Cargo.toml"),
            "license": "GPL-2.0-only", "dependencies": [], "targets":[{"name":name,"kind":["bin"]}]
        }));
        serde_json::json!({"workspace_root":self.tree.root(),"workspace_members":["oblikovati-cli","oblikovati-head"],"packages":packages}).to_string()
    }

    fn git_output(&self, arguments: &[&str]) -> String {
        match arguments {
            ["rev-parse", "--is-shallow-repository"] => "false".into(),
            ["rev-parse", "HEAD"] => "c".repeat(40),
            ["show", ..] => "1791633600".into(),
            ["log", ..] => "feat: fixture application".into(),
            ["ls-remote", repository, tag, ..] => {
                let pin = if repository.contains("Oblikovati.API") {
                    &self.config.api
                } else {
                    &self.config.kernel
                };
                format!("{}\t{tag}\n", pin.revision)
            }
            _ => String::new(),
        }
    }

    fn release_output(&self, endpoint: &str) -> String {
        let name = if endpoint.contains("Oblikovati.API") {
            "oblikovati-api-0.154.0.crate"
        } else {
            "rustycad-kernel-0.1.0.crate"
        };
        serde_json::json!({"draft":false,"prerelease":false,"assets":[{"name":name}]}).to_string()
    }

    fn build_binaries(&self) {
        let extension = if self.platform.target.contains("windows") {
            ".exe"
        } else {
            ""
        };
        for name in ["oblikovati-cli", "oblikovati-head"] {
            self.tree.put(
                &format!("target/{}/release/{name}{extension}", self.platform.target),
                "fixture executable bytes",
            );
        }
    }
}

fn released_config() -> ReleaseConfig {
    let mut config = config();
    config.api.revision = "a".repeat(40);
    config.kernel.version = "0.1.0".into();
    config.kernel.revision = "b".repeat(40);
    config.product.publication_ready = true;
    config.product.reference_key = "fixture-reference-key".into();
    config.product.c_abi = "fixture-abi".into();
    config.linuxdeploy.url = "https://example.test/fixture-linuxdeploy".into();
    config.linuxdeploy.sha256 = format!(
        "{:x}",
        <sha2::Sha256 as sha2::Digest>::digest(b"fixture tool")
    );
    config
}

impl Runner for ReleaseFixture {
    fn output(&self, root: &Path, program: &str, arguments: &[&str]) -> BuildResult<String> {
        self.calls
            .borrow_mut()
            .push(format!("{program} {}", arguments.join(" ")));
        match (program, arguments) {
            ("cargo", ["metadata", ..]) => Ok(self.metadata()),
            ("cargo", ["build", ..]) => {
                self.build_binaries();
                Ok(String::new())
            }
            ("git", arguments) => Ok(self.git_output(arguments)),
            ("gh", ["api", endpoint]) => Ok(self.release_output(endpoint)),
            ("rustc", ["-vV"]) => Ok(format!("host: {}", self.platform.target)),
            ("rustc", _) => Ok("rustc 1.99.0 (fixture)".into()),
            ("brew", ["--prefix", formula]) => Ok(self
                .tree
                .root()
                .join("native")
                .join(formula)
                .to_string_lossy()
                .into_owned()),
            ("curl", _) => {
                let index = arguments
                    .iter()
                    .position(|value| *value == "--output")
                    .unwrap();
                write_text(Path::new(arguments[index + 1]), "fixture tool")?;
                Ok(String::new())
            }
            (_, ["--version"]) => Ok(VERSION.into()),
            _ => {
                let _ = root;
                Ok(String::new())
            }
        }
    }

    fn output_env(
        &self,
        root: &Path,
        program: &str,
        arguments: &[&str],
        environment: &[(&str, &str)],
    ) -> BuildResult<String> {
        if let Some((_, output)) = environment.iter().find(|(name, _)| *name == "OUTPUT") {
            write_text(Path::new(output), "fixture AppImage")?;
        }
        self.output(root, program, arguments)
    }
}
