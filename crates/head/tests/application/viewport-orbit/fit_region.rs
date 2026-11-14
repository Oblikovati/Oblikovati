// SPDX-License-Identifier: GPL-2.0-only
//! Verify that the fixture's analytic corners are not hidden by docked controls.
use super::ApplicationWorld;

pub(super) fn assert_fixture_in_document_area(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let native = &mut world.app().window.native;
    let region = native.gui.as_ref().unwrap().document_area(extent).unwrap();
    // selected_sketch_prism.rs defines this rectangle and browser_feature.rs extrudes it by 2 cm.
    for z in [0., 2.] {
        for y in [0., 3.] {
            for x in [0., 4.] {
                let point = [x, y, z];
                let [px, py] = native.view.model_pixel(point).unwrap();
                assert!(
                    px >= region[0] as i32
                        && px < region[2] as i32
                        && py >= region[1] as i32
                        && py < region[3] as i32,
                    "fixture corner {point:?} projects to [{px}, {py}], outside document area {region:?}"
                );
            }
        }
    }
}
