// SPDX-License-Identifier: GPL-2.0-only
//! Keep the published preview target separate from the already active part.
use super::{ApplicationWorld, holds_text, published_entry::fixture};
use cucumber::given;
use std::path::Path;

const PART_ONE: &str = "quick-access-toolbar/PartOne.opd";
const EXTRUSION: &str = "extrusion/extrusion-rect-8mm.opd";

#[given("`fixtures/cad/quick-access-toolbar/PartOne.opd` is open and active in the MCAD Software")]
fn part_one_active(world: &mut ApplicationWorld) {
    let path = fixture(PART_ONE);
    world.open_fixture(&path);
    let id = world.active_id();
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        path.to_str()
    );
    world.previous_document = Some(id);
}

#[given("a folder contains a saved part with a preview and metadata")]
fn saved_preview_part(world: &mut ApplicationWorld) {
    world.saved_path = Some(fixture(EXTRUSION));
}

#[given(
    "the folder contains `fixtures/cad/extrusion/extrusion-rect-8mm.opd` with a saved preview and metadata"
)]
fn published_preview_part(world: &mut ApplicationWorld) {
    assert_eq!(world.saved_path.as_ref(), Some(&fixture(EXTRUSION)));
}

pub(super) fn inspection_path(world: &ApplicationWorld) -> Option<&Path> {
    world
        .saved_path
        .as_deref()
        .filter(|path| path.ends_with(EXTRUSION))
}

pub(super) fn published_part_details(world: &ApplicationWorld) {
    let path = inspection_path(world).unwrap();
    let size = path.metadata().unwrap().len();
    for row in ["Author:", "Part Number:", &format!("Size: {size} B")] {
        assert!(
            holds_text(world, row),
            "expected {row} in the published part details"
        );
    }
}
