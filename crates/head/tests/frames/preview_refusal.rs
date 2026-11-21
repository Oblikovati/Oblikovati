// SPDX-License-Identifier: GPL-2.0-only
//! Preview refusal clears stale geometry.
use super::*;

#[test]
fn preview_refusal_clears_stale_geometry_and_reports_without_stopping() {
    let packet = oblikovati_viewport::ViewportSketchPreview::Recipe {
        solve_options: Default::default(),
        plane: oblikovati_model::sketch::SketchPlane::xy(),
        recipe: oblikovati_model::sketch::SketchPlacementRecipe::straight_slot(
            [[0., 0.], [10., 0.]],
            2.,
        )
        .unwrap(),
    };
    assert!(
        FakeHeadWindow::default()
            .set_sketch_preview(Some(&packet))
            .is_err()
    );
    let owner = FakeHeadWork {
        preview: Some(packet),
        ..Default::default()
    };
    let mut window = RefusingPreviewWindow::default();
    assert!(publish_sketch_preview(&mut window, &owner).unwrap());
    assert_eq!(window.cleared, 1);
    let message = window.notice.unwrap().message();
    assert!(message.contains("Preview could not be drawn"));
    assert!(message.contains("sample budget"));
}
#[derive(Default)]
struct RefusingPreviewWindow {
    cleared: usize,
    notice: Option<crate::DesktopCommandNotice>,
}
impl HeadWindow for RefusingPreviewWindow {
    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        if preview.is_some() {
            return Err("sample budget exceeded".into());
        }
        self.cleared += 1;
        Ok(true)
    }
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.notice = Some(notice);
        Ok(())
    }
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(vec![])
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
