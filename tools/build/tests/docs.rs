// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::tree::tests::FixtureTree;

#[test]
fn checks_headers_sizes_and_relative_links_with_anchors() {
    let tree = FixtureTree::new();
    tree.put("LICENSE", "GPL");
    tree.put("lib.rs", "// SPDX-License-Identifier: GPL-2.0-only\n");
    tree.put(
        "doc.md",
        "# Build & test\n[local](#build--test) [web](https://example.test)\n",
    );
    check(tree.root()).unwrap();
    tree.put("doc.md", "[broken](missing.md)");
    assert!(check(tree.root()).unwrap_err().contains("missing"));
    tree.put("doc.md", "[broken](#missing)");
    assert!(check(tree.root()).unwrap_err().contains("heading"));
    tree.put("doc.md", "# Valid");
    tree.put("lib.rs", "missing header");
    assert!(check(tree.root()).unwrap_err().contains("SPDX"));
    assert_eq!(
        anchors("# Same\n## Same\n### Code `name`"),
        ["same", "same-1", "code-name"]
    );
}

#[test]
fn rust_source_limit_rejects_500_lines() {
    let tree = FixtureTree::new();
    let header = "// SPDX-License-Identifier: GPL-2.0-only\n";
    tree.put(
        "src/example.rs",
        &format!("{header}{}", "// line\n".repeat(498)),
    );
    check_rust_sources(tree.root()).unwrap();
    tree.put(
        "src/example.rs",
        &format!("{header}{}", "// line\n".repeat(499)),
    );
    let error = check_rust_sources(tree.root()).unwrap_err();
    assert!(error.contains("example.rs") && error.contains("fewer than 500"));
}

#[test]
fn docs_validate_nested_sources_but_not_other_worktrees() {
    let tree = FixtureTree::new();
    tree.put("LICENSE", "GPL");
    tree.put(
        "fixtures/nested/src/lib.rs",
        "// SPDX-License-Identifier: GPL-2.0-only\n",
    );
    tree.put(".slim/worktrees/other/src/broken.rs", "missing header");
    tree.put(".slim/worktrees/other/broken.md", "[broken](absent.md)");
    check(tree.root()).unwrap();
    tree.put("fixtures/nested/src/lib.rs", "missing header");
    let error = check(tree.root()).unwrap_err();
    assert!(error.contains("fixtures/nested") && error.contains("SPDX"));
}
