// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{distribution::sha256, tree::tests::FixtureTree};

#[test]
fn archives_are_repeatable_and_keep_binary_permissions() {
    let tree = FixtureTree::new();
    tree.put("input/bin/oblikovati-cli", "executable bytes");
    tree.put("input/LICENSE", "GPL");
    for extension in ["zip", "tar.gz"] {
        let first = tree.root().join(format!("first.{extension}"));
        let second = tree.root().join(format!("second.{extension}"));
        create(&tree.root().join("input"), &first).unwrap();
        create(&tree.root().join("input"), &second).unwrap();
        assert_eq!(sha256(&first).unwrap(), sha256(&second).unwrap());
    }
    let mut zip = zip::ZipArchive::new(File::open(tree.root().join("first.zip")).unwrap()).unwrap();
    assert_eq!(
        zip.by_name("bin/oblikovati-cli")
            .unwrap()
            .unix_mode()
            .unwrap()
            & 0o777,
        0o755
    );
    assert_eq!(zip.len(), 2);
}
