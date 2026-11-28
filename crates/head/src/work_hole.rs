// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for Part Modify Hole preview, creation, and editing.
macro_rules! hole_work_methods {
    () => {
        /// Dispatch a Hole Properties panel action; example: `work.hole_from_gui(action)?`.
        fn hole_from_gui(&mut self, action: oblikovati_gui::GuiHoleAction) -> Result<bool, String> {
            Err(format!("hole {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Slot Properties panel action; example: `work.slot_from_gui(action)?`.
        fn slot_from_gui(&mut self, action: oblikovati_gui::GuiSlotAction) -> Result<bool, String> {
            Err(format!("slot {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Combine Properties panel action; example: `work.combine_from_gui(action)?`.
        fn combine_from_gui(
            &mut self,
            action: oblikovati_gui::GuiCombineAction,
        ) -> Result<bool, String> {
            Err(format!(
                "combine {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Draft Properties panel action; example: `work.draft_from_gui(action)?`.
        fn draft_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDraftAction,
        ) -> Result<bool, String> {
            Err(format!("draft {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Thread Properties panel action; example: `work.thread_from_gui(action)?`.
        fn thread_from_gui(
            &mut self,
            action: oblikovati_gui::GuiThreadAction,
        ) -> Result<bool, String> {
            Err(format!(
                "thread {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Delete Face Properties panel action; example: `work.delete_face_from_gui(action)?`.
        fn delete_face_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDeleteFaceAction,
        ) -> Result<bool, String> {
            Err(format!(
                "delete face {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Thicken/Offset Properties panel action; example: `work.thicken_offset_from_gui(action)?`.
        fn thicken_offset_from_gui(
            &mut self,
            action: oblikovati_gui::GuiThickenOffsetAction,
        ) -> Result<bool, String> {
            Err(format!(
                "thicken/offset {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Direct Edit toolbar action; example: `work.direct_edit_from_gui(action)?`.
        fn direct_edit_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDirectEditAction,
        ) -> Result<bool, String> {
            Err(format!(
                "direct edit {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Split Properties panel action; example: `work.split_from_gui(action)?`.
        fn split_from_gui(
            &mut self,
            action: oblikovati_gui::GuiSplitAction,
        ) -> Result<bool, String> {
            Err(format!("split {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Mark Properties panel action; example: `work.mark_from_gui(action)?`.
        fn mark_from_gui(&mut self, action: oblikovati_gui::GuiMarkAction) -> Result<bool, String> {
            Err(format!("mark {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Finish Properties panel action; example: `work.finish_from_gui(action)?`.
        fn finish_from_gui(
            &mut self,
            action: oblikovati_gui::GuiFinishAction,
        ) -> Result<bool, String> {
            Err(format!(
                "finish {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a User Coordinate System action; example: `work.ucs_from_gui(action)?`.
        fn ucs_from_gui(&mut self, action: oblikovati_gui::GuiUcsAction) -> Result<bool, String> {
            Err(format!("ucs {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Work Plane action; example: `work.work_plane_from_gui(action)?`.
        fn work_plane_from_gui(
            &mut self,
            action: oblikovati_gui::GuiWorkPlaneAction,
        ) -> Result<bool, String> {
            Err(format!(
                "work plane {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Fillet v2 Properties panel action; example: `work.fillet_v2_from_gui(action)?`.
        fn fillet_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiFilletV2Action,
        ) -> Result<bool, String> {
            Err(format!(
                "fillet {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Chamfer v2 Properties panel action; example: `work.chamfer_v2_from_gui(action)?`.
        fn chamfer_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiChamferV2Action,
        ) -> Result<bool, String> {
            Err(format!(
                "chamfer {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Shell v2 Properties panel action; example: `work.shell_v2_from_gui(action)?`.
        fn shell_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiShellV2Action,
        ) -> Result<bool, String> {
            Err(format!("shell {action:?}: expected a certified part owner"))
        }
    };
}
pub(crate) use hole_work_methods;
macro_rules! forward_hole_work_methods {
    ($owner:ident) => {
        fn hole_from_gui(&mut self, action: oblikovati_gui::GuiHoleAction) -> Result<bool, String> {
            self.$owner.hole_from_gui(action)
        }
        fn slot_from_gui(&mut self, action: oblikovati_gui::GuiSlotAction) -> Result<bool, String> {
            self.$owner.slot_from_gui(action)
        }
        fn combine_from_gui(
            &mut self,
            action: oblikovati_gui::GuiCombineAction,
        ) -> Result<bool, String> {
            self.$owner.combine_from_gui(action)
        }
        fn draft_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDraftAction,
        ) -> Result<bool, String> {
            self.$owner.draft_from_gui(action)
        }
        fn thread_from_gui(
            &mut self,
            action: oblikovati_gui::GuiThreadAction,
        ) -> Result<bool, String> {
            self.$owner.thread_from_gui(action)
        }
        fn delete_face_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDeleteFaceAction,
        ) -> Result<bool, String> {
            self.$owner.delete_face_from_gui(action)
        }
        fn thicken_offset_from_gui(
            &mut self,
            action: oblikovati_gui::GuiThickenOffsetAction,
        ) -> Result<bool, String> {
            self.$owner.thicken_offset_from_gui(action)
        }
        fn direct_edit_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDirectEditAction,
        ) -> Result<bool, String> {
            self.$owner.direct_edit_from_gui(action)
        }
        fn split_from_gui(
            &mut self,
            action: oblikovati_gui::GuiSplitAction,
        ) -> Result<bool, String> {
            self.$owner.split_from_gui(action)
        }
        fn mark_from_gui(&mut self, action: oblikovati_gui::GuiMarkAction) -> Result<bool, String> {
            self.$owner.mark_from_gui(action)
        }
        fn finish_from_gui(
            &mut self,
            action: oblikovati_gui::GuiFinishAction,
        ) -> Result<bool, String> {
            self.$owner.finish_from_gui(action)
        }
        fn ucs_from_gui(&mut self, action: oblikovati_gui::GuiUcsAction) -> Result<bool, String> {
            self.$owner.ucs_from_gui(action)
        }
        fn work_plane_from_gui(
            &mut self,
            action: oblikovati_gui::GuiWorkPlaneAction,
        ) -> Result<bool, String> {
            self.$owner.work_plane_from_gui(action)
        }
        fn fillet_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiFilletV2Action,
        ) -> Result<bool, String> {
            self.$owner.fillet_v2_from_gui(action)
        }
        fn chamfer_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiChamferV2Action,
        ) -> Result<bool, String> {
            self.$owner.chamfer_v2_from_gui(action)
        }
        fn shell_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiShellV2Action,
        ) -> Result<bool, String> {
            self.$owner.shell_v2_from_gui(action)
        }
    };
}
pub(crate) use forward_hole_work_methods;
