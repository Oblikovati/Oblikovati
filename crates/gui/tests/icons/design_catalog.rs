// SPDX-License-Identifier: GPL-2.0-only
use std::{collections::BTreeSet, path::PathBuf};

#[test]
#[ignore = "requires the external design handoff"]
fn rasterizes_every_production_icon_from_the_design_catalog() {
    let root = PathBuf::from(std::env::var_os("OBK_DESIGN_ROOT").expect("set OBK_DESIGN_ROOT"));
    let catalog = std::fs::read_to_string(root.join("ICON-CATALOG.md")).unwrap();
    let paths: BTreeSet<_> = catalog
        .split("](assets/icons/")
        .skip(1)
        .filter_map(|link| link.split_once(')').map(|(path, _)| path))
        .filter(|path| path.ends_with("-32.svg"))
        .collect();
    assert!(
        !paths.is_empty(),
        "expected production SVG links in the design catalog"
    );
    let asset_root = root.join("assets/icons").canonicalize().unwrap();
    for entry in paths {
        let path = asset_root.join(entry).canonicalize().unwrap();
        assert!(path.starts_with(&asset_root));
        let svg = std::fs::read_to_string(&path).unwrap();
        for mode in 0..3 {
            let icon = super::raster::rasterize_identity(&svg, 32, mode)
                .unwrap_or_else(|cause| panic!("{entry}, theme {mode}: {cause}"));
            assert!(
                icon.rgba.iter().any(|pixel| pixel[3] > 0),
                "{entry}, theme {mode}: expected visible artwork"
            );
        }
    }
}
