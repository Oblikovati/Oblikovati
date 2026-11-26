// SPDX-License-Identifier: GPL-2.0-only
macro_rules! sweep_work_methods {
    () => {
        /// Dispatch a Loft draft action. Example: `work.loft_from_gui(action)?`.
        fn loft_from_gui(
            &mut self,
            _action: oblikovati_gui::GuiLoftAction,
        ) -> Result<bool, String> {
            Err("loft: expected an available desktop owner".into())
        }
        /// Dispatch a Sweep command. Example: `work.sweep_from_gui(action)?`.
        fn sweep_from_gui(
            &mut self,
            _action: oblikovati_gui::GuiSweepAction,
        ) -> Result<bool, String> {
            Err("sweep: expected an available desktop owner".into())
        }
    };
}
pub(crate) use sweep_work_methods;
