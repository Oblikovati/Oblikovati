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
