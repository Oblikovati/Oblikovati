// SPDX-License-Identifier: GPL-2.0-only
//! Present the main GUI and any detached document windows on the desktop thread.
use super::*;

impl NativeHeadWindow {
    pub(in crate::native) fn advance_native_gui(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        if self.gui.is_none() && event == Some(&ViewportEvent::CloseRequested) {
            return Ok(Default::default());
        }
        let extent = self.view.drawable_extent()?;
        self.ensure_native_gui()?;
        let now = self.gui_epoch.elapsed();
        let notice = self.native_gui_notice()?;
        let gui = self.gui.as_mut().expect("initialized GUI");
        let mut update = gui.step(now, extent, event, commands, notice.as_deref())?;
        if gui.dirty {
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
