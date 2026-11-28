// SPDX-License-Identifier: GPL-2.0-only
//! Scalar ImGui table, multiline input, and sized selectable bridges (#3803).
use imgui::{ImString, SelectableFlags, TableFlags, Ui, sys};
use std::ffi::{c_char, c_int, c_void};
use std::ops::Range;

unsafe extern "C" {
    fn oblikovati_gui_begin_table(
        id: *const c_char,
        columns: c_int,
        flags: c_int,
        outer_width: f32,
        outer_height: f32,
        inner_width: f32,
    ) -> bool;
    fn oblikovati_gui_input_text_multiline(
        label: *const c_char,
        buf: *mut c_char,
        buf_size: usize,
        width: f32,
        height: f32,
        flags: c_int,
        callback: sys::ImGuiInputTextCallback,
        user_data: *mut c_void,
    ) -> bool;
    fn oblikovati_gui_selectable_sized(
        label: *const c_char,
        selected: bool,
        flags: c_int,
        width: f32,
        height: f32,
    ) -> bool;
}

/// Active ImGui table scope; `Drop` ends the table when `BeginTable` returned `true`.
pub(crate) struct TableToken<'ui> {
    _ui: &'ui Ui,
}

impl TableToken<'_> {
    pub(crate) fn end(self) {}
}

impl Drop for TableToken<'_> {
    fn drop(&mut self) {
        // SAFETY: this token is created only when BeginTable returned true for this live Ui frame.
        unsafe { sys::igEndTable() };
    }
}

/// Begin an ImGui table through scalar dimensions; example: `begin_table(ui, "##rows", 4)`.
pub(crate) fn begin_table<'ui>(ui: &'ui Ui, id: &str, columns: usize) -> Option<TableToken<'ui>> {
    begin_table_with_sizing(ui, id, columns, TableFlags::empty(), [0., 0.], 0.)
}

/// Begin a sized ImGui table without passing `ImVec2` across the FFI ABI (#3803).
pub(crate) fn begin_table_with_sizing<'ui>(
    ui: &'ui Ui,
    id: &str,
    columns: usize,
    flags: TableFlags,
    outer_size: [f32; 2],
    inner_width: f32,
) -> Option<TableToken<'ui>> {
    let columns = c_int::try_from(columns).ok()?;
    let id = ImString::new(id);
    // SAFETY: the Ui frame owns the current ImGui context and C++ consumes the NUL-terminated ID synchronously.
    let opened = unsafe {
        oblikovati_gui_begin_table(
            id.as_ptr(),
            columns,
            flags.bits() as c_int,
            outer_size[0],
            outer_size[1],
            inner_width,
        )
    };
    opened.then_some(TableToken { _ui: ui })
}

/// Draw a selectable with an explicit scalar size; example: `selectable_sized(ui, "##tile", true, flags, [96., 112.])`.
pub(crate) fn selectable_sized(
    _: &Ui,
    label: &str,
    selected: bool,
    flags: SelectableFlags,
    size: [f32; 2],
) -> bool {
    let label = ImString::new(label);
    // SAFETY: the Ui frame owns the context and C++ consumes the NUL-terminated label synchronously.
    unsafe {
        oblikovati_gui_selectable_sized(
            label.as_ptr(),
            selected,
            flags.bits() as c_int,
            size[0],
            size[1],
        )
    }
}

/// Safe view of ImGui's active multiline text-edit callback payload.
pub(crate) struct MultilineCallbackData(*mut sys::ImGuiInputTextCallbackData);

impl MultilineCallbackData {
    pub(crate) fn str(&self) -> &str {
        // SAFETY: ImGui provides a valid UTF-8 buffer of length `BufTextLen` during the callback.
        unsafe {
            std::str::from_utf8(std::slice::from_raw_parts(
                (*self.0).Buf.cast::<u8>(),
                (*self.0).BufTextLen as usize,
            ))
            .expect("ImGui multiline buffer must remain valid UTF-8")
        }
    }

    pub(crate) fn selection(&self) -> Range<usize> {
        // SAFETY: ImGui keeps the callback data live for the callback invocation.
        let (start, end) = unsafe {
            (
                (*self.0).SelectionStart as usize,
                (*self.0).SelectionEnd as usize,
            )
        };
        if start < end { start..end } else { end..start }
    }

    pub(crate) fn cursor_pos(&self) -> usize {
        // SAFETY: ImGui keeps the callback data live for the callback invocation.
        unsafe { (*self.0).CursorPos as usize }
    }

    pub(crate) fn set_cursor_pos(&mut self, cursor_pos: usize) {
        // SAFETY: ImGui keeps the callback data live for the callback invocation.
        unsafe { (*self.0).CursorPos = cursor_pos as i32 };
    }

    pub(crate) fn selection_start_mut(&mut self) -> &mut i32 {
        // SAFETY: ImGui keeps the callback data live for the callback invocation.
        unsafe { &mut (*self.0).SelectionStart }
    }

    pub(crate) fn selection_end_mut(&mut self) -> &mut i32 {
        // SAFETY: ImGui keeps the callback data live for the callback invocation.
        unsafe { &mut (*self.0).SelectionEnd }
    }

    pub(crate) fn insert_chars(&mut self, pos: usize, text: &str) {
        assert!(self.str().is_char_boundary(pos));
        let start = text.as_ptr();
        // SAFETY: `start..end` is valid UTF-8 and `pos` is a char boundary in the callback buffer.
        unsafe {
            let end = start.add(text.len());
            sys::ImGuiInputTextCallbackData_InsertChars(
                self.0,
                pos as i32,
                start.cast::<c_char>(),
                end.cast::<c_char>(),
            );
        }
    }

    pub(crate) fn remove_chars(&mut self, pos: usize, char_count: usize) {
        let inner = &self.str()[pos..];
        let byte_count = inner
            .char_indices()
            .nth(char_count)
            .map_or(inner.len(), |(offset, _)| offset);
        // SAFETY: `pos` and `pos + byte_count` are valid char boundaries in the callback buffer.
        unsafe {
            sys::ImGuiInputTextCallbackData_DeleteChars(self.0, pos as i32, byte_count as i32);
        }
    }
}

/// Edit a multiline string through the scalar C++ bridge without passing `ImVec2` across FFI (#3803).
pub(crate) fn input_text_multiline(ui: &Ui, label: &str, buf: &mut String, size: [f32; 2]) -> bool {
    run_multiline(
        ui,
        label,
        buf,
        size,
        sys::ImGuiInputTextFlags_CallbackResize as c_int,
        |_| {},
    )
}

/// Edit a multiline string with an `ALWAYS` callback through the scalar C++ bridge (#3803).
pub(crate) fn input_text_multiline_always<F: FnMut(&mut MultilineCallbackData)>(
    ui: &Ui,
    label: &str,
    buf: &mut String,
    size: [f32; 2],
    on_always: F,
) -> bool {
    let flags = (sys::ImGuiInputTextFlags_CallbackResize | sys::ImGuiInputTextFlags_CallbackAlways)
        as c_int;
    run_multiline(ui, label, buf, size, flags, on_always)
}

struct MultilineUserData<F> {
    buffer: *mut String,
    on_always: F,
}

fn run_multiline<F: FnMut(&mut MultilineCallbackData)>(
    _: &Ui,
    label: &str,
    buf: &mut String,
    size: [f32; 2],
    flags: c_int,
    on_always: F,
) -> bool {
    buf.push('\0');
    let (ptr, capacity) = (buf.as_mut_ptr(), buf.capacity());
    let mut user_data = MultilineUserData {
        buffer: buf as *mut String,
        on_always,
    };
    let label = ImString::new(label);
    // SAFETY: `buf` is NUL-terminated, `user_data` outlives the synchronous call, and no `ImVec2` crosses FFI.
    let changed = unsafe {
        oblikovati_gui_input_text_multiline(
            label.as_ptr(),
            ptr.cast::<c_char>(),
            capacity,
            size[0],
            size[1],
            flags,
            Some(multiline_callback::<F>),
            (&raw mut user_data).cast::<c_void>(),
        )
    };
    let cap = buf.capacity();
    // SAFETY: ImGui maintains a NUL terminator inside the string's allocation and writes valid UTF-8 before it.
    let bytes = unsafe { std::slice::from_raw_parts(buf.as_ptr(), cap) };
    if let Some(len) = bytes.iter().position(|byte| *byte == b'\0') {
        unsafe { buf.as_mut_vec().set_len(len) };
    }
    changed
}

unsafe extern "C" fn multiline_callback<F: FnMut(&mut MultilineCallbackData)>(
    data: *mut sys::ImGuiInputTextCallbackData,
) -> c_int {
    // SAFETY: ImGui passes our `MultilineUserData<F>` pointer and a live callback record.
    let event = unsafe { (*data).EventFlag as u32 };
    let user_data = unsafe { &mut *((*data).UserData.cast::<MultilineUserData<F>>()) };
    if event == sys::ImGuiInputTextFlags_CallbackResize {
        let requested_size = unsafe { (*data).BufSize as usize };
        let buffer = unsafe { &mut *user_data.buffer };
        if requested_size > buffer.capacity() {
            let additional = requested_size - buffer.len();
            buffer.reserve(additional);
            unsafe {
                (*data).Buf = buffer.as_mut_ptr().cast::<c_char>();
                (*data).BufDirty = true;
            }
        }
    } else if event == sys::ImGuiInputTextFlags_CallbackAlways {
        let mut view = MultilineCallbackData(data);
        (user_data.on_always)(&mut view);
    }
    0
}
