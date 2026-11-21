// SPDX-License-Identifier: GPL-2.0-only
// Explicit category/chunk order preserves the fixed Go catalog display order.
use super::CatalogSource;
pub(super) const SOURCES: &[CatalogSource<'static>] = &[
    CatalogSource {
        name: "00-default.yaml",
        text: include_str!("../../assets/material-catalog/00-default.yaml"),
    },
    CatalogSource {
        name: "01-metals-appearances-001.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-appearances-001.yaml"),
    },
    CatalogSource {
        name: "01-metals-appearances-002.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-appearances-002.yaml"),
    },
    CatalogSource {
        name: "01-metals-appearances-003.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-appearances-003.yaml"),
    },
    CatalogSource {
        name: "01-metals-appearances-004.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-appearances-004.yaml"),
    },
    CatalogSource {
        name: "01-metals-appearances-005.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-appearances-005.yaml"),
    },
    CatalogSource {
        name: "01-metals-appearances-006.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-appearances-006.yaml"),
    },
    CatalogSource {
        name: "01-metals-materials-001.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-materials-001.yaml"),
    },
    CatalogSource {
        name: "01-metals-materials-002.yaml",
        text: include_str!("../../assets/material-catalog/01-metals-materials-002.yaml"),
    },
    CatalogSource {
        name: "02-steels-appearances-001.yaml",
        text: include_str!("../../assets/material-catalog/02-steels-appearances-001.yaml"),
    },
    CatalogSource {
        name: "02-steels-materials-001.yaml",
        text: include_str!("../../assets/material-catalog/02-steels-materials-001.yaml"),
    },
    CatalogSource {
        name: "03-aluminum-appearances-001.yaml",
        text: include_str!("../../assets/material-catalog/03-aluminum-appearances-001.yaml"),
    },
    CatalogSource {
        name: "03-aluminum-materials-001.yaml",
        text: include_str!("../../assets/material-catalog/03-aluminum-materials-001.yaml"),
    },
    CatalogSource {
        name: "04-plastics-appearances-001.yaml",
        text: include_str!("../../assets/material-catalog/04-plastics-appearances-001.yaml"),
    },
    CatalogSource {
        name: "04-plastics-appearances-002.yaml",
        text: include_str!("../../assets/material-catalog/04-plastics-appearances-002.yaml"),
    },
    CatalogSource {
        name: "04-plastics-appearances-003.yaml",
        text: include_str!("../../assets/material-catalog/04-plastics-appearances-003.yaml"),
    },
    CatalogSource {
        name: "04-plastics-appearances-004.yaml",
        text: include_str!("../../assets/material-catalog/04-plastics-appearances-004.yaml"),
    },
    CatalogSource {
        name: "04-plastics-materials-001.yaml",
        text: include_str!("../../assets/material-catalog/04-plastics-materials-001.yaml"),
    },
    CatalogSource {
        name: "05-woods-appearances-001.yaml",
        text: include_str!("../../assets/material-catalog/05-woods-appearances-001.yaml"),
    },
    CatalogSource {
        name: "05-woods-appearances-002.yaml",
        text: include_str!("../../assets/material-catalog/05-woods-appearances-002.yaml"),
    },
    CatalogSource {
        name: "05-woods-appearances-003.yaml",
        text: include_str!("../../assets/material-catalog/05-woods-appearances-003.yaml"),
    },
    CatalogSource {
        name: "05-woods-materials-001.yaml",
        text: include_str!("../../assets/material-catalog/05-woods-materials-001.yaml"),
    },
    CatalogSource {
        name: "05-woods-materials-002.yaml",
        text: include_str!("../../assets/material-catalog/05-woods-materials-002.yaml"),
    },
    CatalogSource {
        name: "06-composites-appearances-001.yaml",
        text: include_str!("../../assets/material-catalog/06-composites-appearances-001.yaml"),
    },
    CatalogSource {
        name: "06-composites-appearances-002.yaml",
        text: include_str!("../../assets/material-catalog/06-composites-appearances-002.yaml"),
    },
    CatalogSource {
        name: "06-composites-materials-001.yaml",
        text: include_str!("../../assets/material-catalog/06-composites-materials-001.yaml"),
    },
    CatalogSource {
        name: "07-magnetic-appearances-001.yaml",
        text: include_str!("../../assets/material-catalog/07-magnetic-appearances-001.yaml"),
    },
    CatalogSource {
        name: "07-magnetic-materials-001.yaml",
        text: include_str!("../../assets/material-catalog/07-magnetic-materials-001.yaml"),
    },
];

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn every_shipped_catalog_file_is_embedded_and_remains_small() {
        let directory =
            std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("assets/material-catalog");
        let mut files: Vec<String> = std::fs::read_dir(directory)
            .unwrap()
            .map(|entry| entry.unwrap().file_name().into_string().unwrap())
            .collect();
        files.sort();
        let mut embedded: Vec<String> = SOURCES.iter().map(|source| source.name.into()).collect();
        embedded.sort();
        assert_eq!(files, embedded);
        for source in SOURCES {
            assert!(source.text.lines().count() < 500, "{}", source.name);
            assert!(
                source
                    .text
                    .starts_with("# SPDX-License-Identifier: GPL-2.0-only"),
                "{}",
                source.name
            );
        }
    }
}
