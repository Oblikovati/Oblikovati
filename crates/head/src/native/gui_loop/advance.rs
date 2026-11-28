// SPDX-License-Identifier: GPL-2.0-only
//! Present the main GUI and any detached document windows on the desktop thread.
use super::*;

impl NativeGui {
    pub(in crate::native) fn clear_document_area(&mut self) {
        self.document_area = None;
    }
    pub(in crate::native) fn home_in_front(&self) -> bool {
        self.engine.home_in_front()
    }
    pub(in crate::native) fn request_revolve(&mut self) {
        self.engine.request_revolve();
        self.deadline = Some(Duration::ZERO);
    }
}

#[cfg(target_os = "linux")]
impl NativeGui {
    pub(in crate::native) fn note_window_preview_pixels(
        &mut self,
        extent: [u32; 2],
        generations: [u64; 4],
        pixels: oblikovati_viewport::ViewportPreviewPixels<'_>,
    ) {
        self.engine
            .note_presented_window_preview(extent, generations, pixels);
    }
}

impl NativeHeadWindow {
    pub(in crate::native) fn finish_native_window(&mut self) -> Result<(), String> {
        let profile_error = match (&mut self.gui, &self.ribbon_settings_directory) {
            (Some(gui), Some(directory)) => gui.save_profile(directory).err(),
            _ => None,
        };
        self.drawing_windows.close();
        self.gui.take();
        #[cfg(target_os = "windows")]
        self.accessibility.take();
        self.capture_requests.clear();
        #[cfg(target_os = "linux")]
        let clipboard = self.clipboard.handoff_on_shutdown();
        self.view.close();
        let mut errors = if self.validate_vulkan {
            self.runtime.validation_errors()?
        } else {
            Vec::new()
        };
        #[cfg(target_os = "linux")]
        errors.extend(clipboard.err());
        errors.extend(profile_error);
        if errors.is_empty() {
            return Ok(());
        }
        Err(format!(
            "head shutdown {errors:?}: expected saved workspace settings, completed clipboard retirement, and no renderer errors"
        ))
    }
    pub(in crate::native) fn advance_native_gui(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &mut GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        if self.gui.is_none() && event == Some(&ViewportEvent::CloseRequested) {
            return Ok(Default::default());
        }
        let extent = self.view.drawable_extent()?;
        self.publish_status_telemetry(event, extent, commands);
        let backing_scale = desktop_gui_scale(self.view.backing_scale(), commands.ui_scale);
        self.ensure_native_gui()?;
        let now = self.gui_epoch.elapsed();
        let notice = self.native_gui_notice()?;
        let gui = self.gui.as_mut().expect("initialized GUI");
        let previous_area = gui.document_area;
        let mut update = gui.step_scaled(
            now,
            extent,
            backing_scale,
            event,
            commands,
            notice.as_deref(),
        )?;
        if previous_area != gui.document_area
            && let Some((layout_extent, Some(area))) = gui.document_area
            && self.view.set_document_projection(layout_extent, area)?
        {
            // Annotation coordinates were prepared from the previous projection.
            gui.dirty = true;
            gui.deadline = Some(now);
            update.changed = true;
            update.deferred = true;
            update.wake_after = Some(Duration::ZERO);
        } else if gui.dirty {
            publish_gui(&mut self.view, gui, now, &mut update)?;
        }
        let drawing = self.drawing_windows.advance(
            &self.runtime,
            gui.engine.detached_drawings(),
            commands,
            now,
        );
        if let Some((document, cause)) = drawing.failure {
            self.command_notice =
                Some(crate::DesktopCommandNotice::DrawingWindowFailed { document, cause });
        }
        for id in drawing.restored {
            gui.engine.restore_detached_drawing(id);
            gui.parameter_changed = true;
            update.wake_after = Some(Duration::ZERO);
        }
        update.actions.extend(drawing.actions);
        update.wake_after = update
            .wake_after
            .into_iter()
            .chain(drawing.wake_after)
            .min();
        Ok(update)
    }
}

pub(super) fn desktop_gui_scale(
    backing_scale: f32,
    options: Option<oblikovati_model::ui_scale::UiScaleOptions>,
) -> f32 {
    if cfg!(target_os = "linux")
        && backing_scale > 1.
        && options.is_some_and(|options| options.saved_font().is_none())
    {
        return 1. + (backing_scale - 1.) * 0.5;
    }
    backing_scale
}

#[cfg(all(test, target_os = "linux"))]
mod tests {
    use super::desktop_gui_scale;
    use oblikovati_model::ui_scale::UiScaleOptions;

    #[test]
    fn automatic_layout_softens_only_x11_upscaling() {
        let automatic = Some(UiScaleOptions::default());
        let explicit = Some(UiScaleOptions::new(1., 1.).unwrap());
        for (backing, layout) in [(0.75, 0.75), (1., 1.), (1.5, 1.25), (2., 1.5)] {
            assert_eq!(desktop_gui_scale(backing, automatic), layout);
            assert_eq!(desktop_gui_scale(backing, explicit), backing);
            assert_eq!(desktop_gui_scale(backing, None), backing);
        }
    }
}
