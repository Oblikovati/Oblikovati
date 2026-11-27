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
        /// Dispatch a Revolve command. Example: `work.revolve_from_gui(action)?`.
        fn revolve_from_gui(
            &mut self,
            _action: oblikovati_gui::GuiRevolveAction,
        ) -> Result<bool, String> {
            Err("revolve: expected an available desktop owner".into())
        }
        /// Dispatch a Coil command. Example: `work.coil_from_gui(action)?`.
        fn coil_from_gui(
            &mut self,
            _action: oblikovati_gui::GuiCoilAction,
        ) -> Result<bool, String> {
            Err("coil: expected an available desktop owner".into())
        }
        /// Dispatch an Emboss command. Example: `work.emboss_from_gui(action)?`.
        fn emboss_from_gui(
            &mut self,
            _action: oblikovati_gui::GuiEmbossAction,
        ) -> Result<bool, String> {
            Err("emboss: expected an available desktop owner".into())
        }
        /// Dispatch a Derive command. Example: `work.derive_from_gui(action)?`.
        fn derive_from_gui(
            &mut self,
            _action: oblikovati_gui::GuiDeriveAction,
        ) -> Result<bool, String> {
            Err("derive: expected an available desktop owner".into())
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
macro_rules! forward_sweep_work_methods {
    ($owner:ident) => {
        fn loft_from_gui(&mut self, action: oblikovati_gui::GuiLoftAction) -> Result<bool, String> {
            self.$owner.loft_from_gui(action)
        }
        fn revolve_from_gui(
            &mut self,
            action: oblikovati_gui::GuiRevolveAction,
        ) -> Result<bool, String> {
            self.$owner.revolve_from_gui(action)
        }
        fn coil_from_gui(&mut self, action: oblikovati_gui::GuiCoilAction) -> Result<bool, String> {
            self.$owner.coil_from_gui(action)
        }
        fn emboss_from_gui(
            &mut self,
            action: oblikovati_gui::GuiEmbossAction,
        ) -> Result<bool, String> {
            self.$owner.emboss_from_gui(action)
        }
        fn derive_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDeriveAction,
        ) -> Result<bool, String> {
            self.$owner.derive_from_gui(action)
        }
        fn sweep_from_gui(
            &mut self,
            action: oblikovati_gui::GuiSweepAction,
        ) -> Result<bool, String> {
            self.$owner.sweep_from_gui(action)
        }
    };
}
pub(crate) use forward_sweep_work_methods;
pub(crate) use sweep_work_methods;
