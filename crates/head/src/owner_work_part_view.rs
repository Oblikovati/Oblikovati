// SPDX-License-Identifier: GPL-2.0-only
//! Desktop actions for part view representations.
macro_rules! desktop_part_view_work {
    () => {
        fn part_view_choice_from_gui(
            &mut self,
            document: u64,
            choice: &oblikovati_gui::GuiBrowserChoice,
            window: &mut dyn crate::HeadWindow,
        ) -> Result<bool, String> {
            use oblikovati_automation::PartViewDisplayAccess as _;
            self.require_browser_document(document)?;
            let mut display = crate::part_view_display::WindowPartViewDisplay(window);
            let original = if matches!(
                choice,
                oblikovati_gui::GuiBrowserChoice::SavePartViewCamera(_)
                    | oblikovati_gui::GuiBrowserChoice::RestorePartViewCamera(_)
            ) {
                Some(display.camera()?)
            } else {
                None
            };
            let client = oblikovati_api::client::Client::new(
                self.document_host().with_part_view_display(&mut display),
            );
            match choice {
                oblikovati_gui::GuiBrowserChoice::CopyPartView(id) => {
                    client
                        .part_design_views()
                        .copy(&oblikovati_api::wire::CopyPartDesignViewArgs {
                            source_id: *id,
                            name: String::new(),
                        })
                        .map_err(|cause| cause.to_string())?;
                }
                oblikovati_gui::GuiBrowserChoice::ActivatePartView(id) => {
                    client
                        .part_design_views()
                        .activate(&oblikovati_api::wire::PartDesignViewRef { id: *id })
                        .map_err(|cause| cause.to_string())?;
                }
                oblikovati_gui::GuiBrowserChoice::SavePartViewCamera(id) => {
                    client
                        .part_design_views()
                        .capture_camera(&oblikovati_api::wire::CapturePartDesignViewCameraArgs {
                            id: *id,
                            camera: original.ok_or("camera save: expected the current camera")?,
                        })
                        .map_err(|cause| cause.to_string())?;
                }
                oblikovati_gui::GuiBrowserChoice::RestorePartViewCamera(id) => {
                    let camera = client
                        .part_design_views()
                        .camera(&oblikovati_api::wire::PartDesignViewRef { id: *id })
                        .map_err(|cause| cause.to_string())?;
                    drop(client);
                    if let Err(cause) = display.apply(&camera) {
                        if let Some(original) = original {
                            display.apply(&original).map_err(|restore| {
                                format!("{cause}; camera restore failed: {restore}")
                            })?;
                        }
                        return Err(cause);
                    }
                }
                _ => {
                    return Err(format!(
                        "part view choice {choice:?}: expected a view camera or activation action"
                    ));
                }
            }
            Ok(true)
        }
        fn jump_history_with_window(
            &mut self,
            document: u64,
            position: i64,
            window: &mut dyn crate::HeadWindow,
        ) -> Result<bool, String> {
            self.jump_history_action_with_window(document, position, window)
        }
        fn process_live_window_inputs(
            &mut self,
            inputs: &[oblikovati_viewport::ViewportEvent],
            window: &mut dyn crate::HeadWindow,
        ) -> Result<bool, String> {
            let cue_changed = self.update_model_pick_cue(inputs, window);
            let ucs_changed = self.process_ucs_inputs(inputs);
            Ok(self.process_live_placement_inputs(inputs, window)? || cue_changed || ucs_changed)
        }
        fn poll_window(
            &mut self,
            budget: usize,
            window: &mut dyn crate::HeadWindow,
        ) -> Result<HeadWorkStatus, String> {
            if budget == 0 {
                return Err("desktop drain budget 0: expected a positive bounded batch".into());
            }
            let mut display = crate::part_view_display::WindowPartViewDisplay(window);
            let serviced = self.drain_calls(budget, Some(&mut display))?;
            self.finish_owner_poll(serviced)
        }
    };
}
