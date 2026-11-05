// SPDX-License-Identifier: GPL-2.0-only
use crate::*;
#[path = "mod.rs"]
mod parameter_journey;
#[test]
fn parameter_journey_uses_the_configured_widget_shortcuts() {
    for mac_osx in [false, true] {
        let mut engine = GuiFrameEngine::new().unwrap();
        engine.context.io_mut().config_mac_os_behaviors = mac_osx;
        parameter_journey::exercise_parameter_journey(engine, mac_osx);
    }
}
