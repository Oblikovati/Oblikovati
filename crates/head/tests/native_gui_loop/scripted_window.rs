// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl HeadWindow for ScriptedGuiWindow {
    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        self.native.set_sketch_preview(preview)
    }
    fn set_view_slot(&mut self, slot: u64) -> Result<bool, String> {
        self.native.set_view_slot(slot)
    }
    fn sketch_pixel(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        point: [f64; 2],
    ) -> Result<[i32; 2], String> {
        HeadWindow::sketch_pixel(&self.native, plane, point)
    }
    fn model_pixel(&self, point: [f64; 3]) -> Result<[i32; 2], String> {
        self.native.model_pixel(point)
    }
    fn view_camera(&self) -> Result<oblikovati_viewport::ViewportCamera, String> {
        self.native.view_camera()
    }
    fn view_line(&self, pixel: [i32; 2]) -> Result<[[f64; 3]; 2], String> {
        self.native.view_line(pixel)
    }
    fn fit_view(&mut self) -> Result<bool, String> {
        self.native.fit_view()
    }
    fn view_projection(&self) -> oblikovati_gui::GuiViewProjection {
        self.native.view_projection()
    }
    fn set_view_projection(&mut self, mode: oblikovati_gui::GuiViewProjection) {
        self.native.set_view_projection(mode);
    }
    fn set_view_camera(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<bool, String> {
        self.native.set_view_camera(camera)
    }
    fn navigation_active(&self) -> bool {
        self.native.navigation_active()
    }
    fn sketch_point(
        &self,
        pixel: [i32; 2],
        plane: oblikovati_model::sketch::SketchPlane,
    ) -> Result<Option<[f64; 2]>, String> {
        self.native.sketch_point(pixel, plane)
    }
    fn set_sketch_edit(&mut self, edit: Option<crate::HeadSketchEdit>) -> Result<bool, String> {
        self.native.set_sketch_edit(edit)
    }
    fn gui_frame(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        self.native.gui_frame(event, commands)
    }
    fn capture_window_preview(&mut self) -> Result<bool, String> {
        self.native.capture_window_preview()
    }
    fn request_close(&mut self) -> Result<bool, String> {
        self.native.request_close()
    }
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = self.native.events(limit)?;
        events.append(&mut self.scripted);
        Ok(events)
    }
    fn process_view_input(&mut self, event: &ViewportEvent) -> Result<bool, String> {
        self.passed.push(*event);
        self.native.process_view_input(event)
    }
    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.native.submit_scene(update)
    }
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.native.submit_sketch_scene(update)
    }
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.native.show_command_notice(notice)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        if !self.presentation_delay.is_zero() {
            std::thread::sleep(self.presentation_delay);
        }
        self.native.present()
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
