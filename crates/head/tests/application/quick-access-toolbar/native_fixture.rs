// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[given("`fixtures/cad/blank-metric-part.opd` is open and active in the MCAD Software")]
fn open_blank_metric_part(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/blank-metric-part.opd");
    world.open_fixture(&path);
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    world.previous_document = Some(world.active_id());
}

#[given(
    "\"fixtures/cad/quick-access-toolbar/PartOne.opd\", \"fixtures/cad/quick-access-toolbar/PartTwo.opd\" and \"fixtures/cad/quick-access-toolbar/Assembly.oad\" are open in the MCAD Software"
)]
fn open_native_part_pair_and_assembly(world: &mut ApplicationWorld) {
    let folder = super::super::scenario_root().join("fixtures/cad/quick-access-toolbar");
    for name in ["PartOne.opd", "PartTwo.opd"] {
        world.open_fixture(&folder.join(name));
        assert_eq!(
            world.state().document_type,
            Some(oblikovati_api::types::DOCUMENT_PART)
        );
        let id = world.active_id();
        world.parts.push(id);
    }
    world.open_fixture(&folder.join("Assembly.oad"));
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
    );
    world.other_document = Some(world.active_id());
}

#[given("PartOne.opd is active")]
fn first_part_is_active(world: &mut ApplicationWorld) {
    reactivate_first(world);
    assert_eq!(world.active_id(), world.parts[0]);
}
