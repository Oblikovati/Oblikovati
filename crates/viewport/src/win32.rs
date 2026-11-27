// SPDX-License-Identifier: GPL-2.0-only
//! Win32 owner-thread window and presentation of the Martensite offscreen target.
use crate::{ViewportEvent, ViewportReady, ViewportWake, types::valid_extent};
use martensite_tkernel::standard::Handle;
use martensite_tkservice::aspect::{AspectNeutralWindow, AspectWindowMembers};
use std::{
    cell::{Cell, RefCell},
    collections::VecDeque,
    ptr,
    time::Duration,
};
use windows_sys::Win32::{
    Foundation::{HWND, LPARAM, LRESULT, RECT, WPARAM},
    Graphics::Gdi::{InvalidateRect, UpdateWindow},
    System::LibraryLoader::GetModuleHandleW,
    UI::HiDpi::{
        DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2, GetDpiForSystem, GetDpiForWindow,
        SetProcessDpiAwarenessContext,
    },
    UI::WindowsAndMessaging::*,
};
mod clipboard;
mod input;
mod pixels;
pub use clipboard::WindowsClipboard;

#[derive(Default)]
struct WindowEvents {
    queue: RefCell<VecDeque<ViewportEvent>>,
    frame: RefCell<Option<pixels::WindowFrame>>,
    mapped: Cell<bool>,
    wheel: Cell<i32>,
    releasing_capture: Cell<bool>,
    surrogate: Cell<Option<u16>>,
}

pub(crate) struct NativeWindow {
    // The pointer installed in GWLP_USERDATA stays valid until DestroyWindow returns.
    events: std::rc::Rc<WindowEvents>,
    hwnd: HWND,
    style: u32,
    pub window: Handle<AspectNeutralWindow>,
}
const STYLE: u32 = WS_POPUP | WS_THICKFRAME | WS_SYSMENU | WS_MINIMIZEBOX | WS_MAXIMIZEBOX;

impl NativeWindow {
    pub fn create(title: &str, extent: [u32; 2], system_frame: bool) -> Result<Self, String> {
        let style = if system_frame {
            WS_OVERLAPPEDWINDOW
        } else {
            STYLE
        };
        Self::create_with_style(title, extent, style, |_| {})
    }
    pub fn create_with_setup(
        title: &str,
        extent: [u32; 2],
        setup: impl FnOnce(isize),
    ) -> Result<Self, String> {
        Self::create_with_style(title, extent, STYLE, setup)
    }
    fn create_with_style(
        title: &str,
        extent: [u32; 2],
        style: u32,
        setup: impl FnOnce(isize),
    ) -> Result<Self, String> {
        // This must happen before creating the first top-level window. A host may
        // already have selected an awareness context, in which case Windows
        // retains that context and this call returns false.
        static DPI_AWARENESS: std::sync::Once = std::sync::Once::new();
        DPI_AWARENESS.call_once(|| unsafe {
            SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2);
        });
        let dpi = unsafe { GetDpiForSystem() }.max(96);
        let [width, height] = physical_extent(extent, dpi)?;
        let title = wide_title(title)?;
        let class = windows_sys::w!("OblikovatiViewport");
        let instance = unsafe { GetModuleHandleW(ptr::null()) };
        register_class(instance, class)?;
        let rect = window_rect(width, height, style)?;
        let hwnd = unsafe {
            CreateWindowExW(
                0,
                class,
                title.as_ptr(),
                style,
                CW_USEDEFAULT,
                CW_USEDEFAULT,
                rect.right - rect.left,
                rect.bottom - rect.top,
                ptr::null_mut(),
                ptr::null_mut(),
                instance,
                ptr::null(),
            )
        };
        if hwnd.is_null() {
            return Err(win_error("create viewport window"));
        }
        let events = std::rc::Rc::<WindowEvents>::default();
        let window = Handle::new(AspectNeutralWindow::default());
        window.set_virtual(true);
        window.set_size(width, height);
        unsafe { SetWindowLongPtrW(hwnd, GWLP_USERDATA, ptr::from_ref(events.as_ref()) as isize) };
        let native = Self {
            events,
            hwnd,
            style,
            window,
        };
        setup(hwnd as isize);
        native.mapped(true)?;
        native.refresh_extent();
        Ok(native)
    }

    pub fn window_action(&self, action: crate::ViewportWindowAction) -> Result<(), String> {
        match action {
            crate::ViewportWindowAction::Minimize => unsafe {
                ShowWindow(self.hwnd, SW_MINIMIZE);
            },
            crate::ViewportWindowAction::ToggleMaximize => unsafe {
                ShowWindow(
                    self.hwnd,
                    if IsZoomed(self.hwnd) != 0 {
                        SW_RESTORE
                    } else {
                        SW_MAXIMIZE
                    },
                );
            },
            crate::ViewportWindowAction::Move => unsafe {
                windows_sys::Win32::UI::Input::KeyboardAndMouse::ReleaseCapture();
                SendMessageW(self.hwnd, WM_NCLBUTTONDOWN, HTCAPTION as usize, 0);
            },
        }
        Ok(())
    }
    pub fn request_accessible_focus(&self) -> Result<(), String> {
        if unsafe { SetForegroundWindow(self.hwnd) } == 0 {
            return Err(win_error("focus accessible viewport window"));
        }
        Ok(())
    }
    pub fn window_maximized(&self) -> Result<bool, String> {
        Ok(unsafe { IsZoomed(self.hwnd) != 0 })
    }

    pub fn refresh_extent(&self) {
        let mut rect = RECT::default();
        if unsafe { GetClientRect(self.hwnd, &mut rect) } != 0 && rect.right > 0 && rect.bottom > 0
        {
            self.window.set_size(rect.right, rect.bottom);
        }
    }

    pub fn backing_scale(&self) -> f32 {
        let dpi = unsafe { GetDpiForWindow(self.hwnd) };
        if dpi == 0 { 1.0 } else { dpi as f32 / 96.0 }
    }

    pub fn resize(&self, extent: [u32; 2]) -> Result<(), String> {
        let [width, height] = valid_extent(extent)?;
        let rect = window_rect(width, height, self.style)?;
        if unsafe {
            SetWindowPos(
                self.hwnd,
                ptr::null_mut(),
                0,
                0,
                rect.right - rect.left,
                rect.bottom - rect.top,
                SWP_NOMOVE | SWP_NOZORDER | SWP_NOACTIVATE,
            )
        } == 0
        {
            return Err(win_error("resize viewport window"));
        }
        Ok(())
    }

    pub fn is_mapped(&self) -> bool {
        unsafe { IsWindowVisible(self.hwnd) != 0 && IsIconic(self.hwnd) == 0 }
    }

    pub fn mapped(&self, mapped: bool) -> Result<(), String> {
        unsafe { ShowWindow(self.hwnd, if mapped { SW_RESTORE } else { SW_HIDE }) };
        self.events.observe_mapping(self.hwnd);
        Ok(())
    }

    pub fn poll(&self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        for _ in 0..limit {
            if self.events.queue.borrow().len() >= limit {
                break;
            }
            let mut msg = MSG::default();
            // Pump thread input as well as window messages, since IME uses helper windows.
            if unsafe { PeekMessageW(&mut msg, ptr::null_mut(), 0, 0, PM_REMOVE) } == 0 {
                break;
            }
            if msg.message == WM_QUIT {
                self.events.push(ViewportEvent::CloseRequested);
                break;
            }
            unsafe {
                TranslateMessage(&msg);
                DispatchMessageW(&msg);
            }
        }
        let mut queue = self.events.queue.borrow_mut();
        let count = queue.len().min(limit);
        Ok(queue.drain(..count).collect())
    }

    pub fn wait_events(
        &self,
        wake: &ViewportWake,
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        if self.has_pending_input() {
            let mut ready = wake.wait(Some(Duration::ZERO))?;
            ready.native_events = true;
            return Ok(ready);
        }
        wake.wait(timeout)
    }
    pub(crate) fn has_pending_input(&self) -> bool {
        !self.events.queue.borrow().is_empty()
    }

    pub fn set_title(&self, title: &str) -> Result<(), String> {
        let title = wide_title(title)?;
        if unsafe { SetWindowTextW(self.hwnd, title.as_ptr()) } == 0 {
            return Err(win_error("set viewport title"));
        }
        Ok(())
    }

    pub fn title(&self) -> Result<String, String> {
        let len = unsafe { GetWindowTextLengthW(self.hwnd) };
        let mut title = vec![0; len as usize + 1];
        let copied = unsafe { GetWindowTextW(self.hwnd, title.as_mut_ptr(), len + 1) };
        String::from_utf16(&title[..copied as usize])
            .map_err(|err| format!("viewport title: {err}"))
    }

    pub fn text_input_issue(&self) -> Option<String> {
        None
    }

    pub(crate) fn clipboard(&self) -> WindowsClipboard {
        WindowsClipboard::new(self.hwnd, std::rc::Rc::downgrade(&self.events))
    }

    pub fn present(&self, extent: [u32; 2], rgba: &[u8]) -> Result<(), String> {
        *self.events.frame.borrow_mut() = Some(pixels::WindowFrame::from_rgba(extent, rgba)?);
        unsafe {
            InvalidateRect(self.hwnd, ptr::null(), 0);
        }
        if unsafe { UpdateWindow(self.hwnd) } == 0 {
            return Err(win_error("present viewport window"));
        }
        Ok(())
    }
}

impl Drop for NativeWindow {
    fn drop(&mut self) {
        unsafe { DestroyWindow(self.hwnd) };
    }
}

impl WindowEvents {
    fn push(&self, event: ViewportEvent) {
        self.queue.borrow_mut().push_back(event);
    }

    fn observe_mapping(&self, hwnd: HWND) {
        let mapped = unsafe { IsWindowVisible(hwnd) != 0 && IsIconic(hwnd) == 0 };
        if self.mapped.replace(mapped) != mapped {
            self.push(ViewportEvent::Mapped(mapped));
        }
    }
}

unsafe extern "system" fn window_proc(
    hwnd: HWND,
    msg: u32,
    wparam: WPARAM,
    lparam: LPARAM,
) -> LRESULT {
    let state = unsafe { GetWindowLongPtrW(hwnd, GWLP_USERDATA) } as *const WindowEvents;
    if let Some(events) = unsafe { state.as_ref() }
        && let Some(result) = dispatch(events, hwnd, msg, wparam, lparam)
    {
        return result;
    }
    unsafe { DefWindowProcW(hwnd, msg, wparam, lparam) }
}

fn dispatch(
    events: &WindowEvents,
    hwnd: HWND,
    msg: u32,
    wparam: WPARAM,
    lparam: LPARAM,
) -> Option<LRESULT> {
    match msg {
        WM_CLOSE => events.push(ViewportEvent::CloseRequested),
        WM_DPICHANGED => {
            // Windows supplies a physical-pixel rectangle preserving the window's
            // logical size on the new monitor. WM_SIZE triggers a fresh GUI frame.
            if let Some(rect) = unsafe { (lparam as *const RECT).as_ref() } {
                unsafe {
                    SetWindowPos(
                        hwnd,
                        ptr::null_mut(),
                        rect.left,
                        rect.top,
                        rect.right - rect.left,
                        rect.bottom - rect.top,
                        SWP_NOZORDER | SWP_NOACTIVATE,
                    )
                };
            }
            let mut client = RECT::default();
            if unsafe { GetClientRect(hwnd, &mut client) } != 0
                && client.right > 0
                && client.bottom > 0
            {
                events.push(ViewportEvent::Resized([
                    client.right as u32,
                    client.bottom as u32,
                ]));
            }
        }
        WM_GETMINMAXINFO => {
            // The application bar must remain usable at the smallest restored size.
            if let Some(info) = unsafe { (lparam as *mut MINMAXINFO).as_mut() } {
                let dpi = unsafe { GetDpiForWindow(hwnd) }.max(96);
                info.ptMinTrackSize.x = ((640 * dpi + 48) / 96) as i32;
                info.ptMinTrackSize.y = ((480 * dpi + 48) / 96) as i32;
            }
            return Some(0);
        }

        WM_SETFOCUS => events.push(ViewportEvent::Focus(true)),
        WM_KILLFOCUS => {
            events.wheel.set(0);
            events.surrogate.set(None);
            events.push(ViewportEvent::Focus(false));
        }
        WM_PAINT => pixels::paint(hwnd, events.frame.borrow().as_ref()),
        WM_ERASEBKGND => return Some(1),
        WM_SHOWWINDOW => {
            events.observe_mapping(hwnd);
            return None;
        }
        WM_SIZE => {
            events.observe_mapping(hwnd);
            let (width, height) = (lparam as u16 as u32, (lparam >> 16) as u16 as u32);
            if width != 0 && height != 0 && wparam != SIZE_MINIMIZED as usize {
                events.push(ViewportEvent::Resized([width, height]));
            }
        }
        _ => return input::dispatch(events, hwnd, msg, wparam, lparam),
    }
    Some(0)
}

fn register_class(
    instance: windows_sys::Win32::Foundation::HINSTANCE,
    class: *const u16,
) -> Result<(), String> {
    let window_class = WNDCLASSW {
        lpfnWndProc: Some(window_proc),
        hInstance: instance,
        hCursor: unsafe { LoadCursorW(ptr::null_mut(), IDC_ARROW) },
        lpszClassName: class,
        ..WNDCLASSW::default()
    };
    if unsafe { RegisterClassW(&window_class) } == 0
        && std::io::Error::last_os_error().raw_os_error() != Some(1410)
    {
        return Err(win_error("register viewport window class"));
    }
    Ok(())
}

fn window_rect(width: i32, height: i32, style: u32) -> Result<RECT, String> {
    let mut rect = RECT {
        left: 0,
        top: 0,
        right: width,
        bottom: height,
    };
    if unsafe { AdjustWindowRectEx(&mut rect, style, 0, 0) } == 0 {
        return Err(win_error("size viewport window frame"));
    }
    Ok(rect)
}

fn physical_extent(extent: [u32; 2], dpi: u32) -> Result<[i32; 2], String> {
    valid_extent(extent)?;
    let physical = extent.map(|logical| (u64::from(logical) * u64::from(dpi) + 48) / 96);
    if physical
        .iter()
        .any(|value| *value == 0 || *value > i32::MAX as u64)
    {
        return Err(format!(
            "viewport extent {extent:?} at {dpi} DPI: expected positive 31-bit physical dimensions"
        ));
    }
    Ok(physical.map(|value| value as i32))
}

fn wide_title(title: &str) -> Result<Vec<u16>, String> {
    if title.contains('\0') {
        return Err("viewport title: expected no embedded NUL characters".into());
    }
    Ok(title.encode_utf16().chain([0]).collect())
}

fn win_error(operation: &str) -> String {
    format!("{operation}: {}", std::io::Error::last_os_error())
}

#[cfg(test)]
mod setup_tests {
    use super::*;

    #[test]
    fn logical_window_extents_scale_without_accepting_overflow() {
        assert_eq!(physical_extent([640, 480], 144).unwrap(), [960, 720]);
        assert_eq!(physical_extent([640, 480], 96).unwrap(), [640, 480]);
        assert!(physical_extent([0, 480], 144).is_err());
        assert!(physical_extent([i32::MAX as u32, 480], 144).is_err());
    }

    #[test]
    fn native_setup_runs_before_the_window_is_visible() {
        let visited = Cell::new(false);
        let native = NativeWindow::create_with_setup("setup visibility", [640, 480], |handle| {
            assert_eq!(unsafe { IsWindowVisible(handle as HWND) }, 0);
            visited.set(true);
        })
        .expect("create a native test window");
        assert!(visited.get());
        assert_ne!(unsafe { IsWindowVisible(native.hwnd) }, 0);
    }
}
