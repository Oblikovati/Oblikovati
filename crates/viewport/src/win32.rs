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
    pub window: Handle<AspectNeutralWindow>,
}
const STYLE: u32 = WS_POPUP | WS_THICKFRAME | WS_SYSMENU | WS_MINIMIZEBOX | WS_MAXIMIZEBOX;

impl NativeWindow {
    pub fn create(title: &str, extent: [u32; 2]) -> Result<Self, String> {
        Self::create_with_setup(title, extent, |_| {})
    }
    pub fn create_with_setup(
        title: &str,
        extent: [u32; 2],
        setup: impl FnOnce(isize),
    ) -> Result<Self, String> {
        let [width, height] = valid_extent(extent)?;
        let title = wide_title(title)?;
        let class = windows_sys::w!("OblikovatiViewport");
        let instance = unsafe { GetModuleHandleW(ptr::null()) };
        register_class(instance, class)?;
        let rect = window_rect(width, height)?;
        let hwnd = unsafe {
            CreateWindowExW(
                0,
                class,
                title.as_ptr(),
                STYLE,
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

    pub fn resize(&self, extent: [u32; 2]) -> Result<(), String> {
        let [width, height] = valid_extent(extent)?;
        let rect = window_rect(width, height)?;
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
        if !self.events.queue.borrow().is_empty() {
            let mut ready = wake.wait(Some(Duration::ZERO))?;
            ready.native_events = true;
            return Ok(ready);
        }
        wake.wait(timeout)
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
        WM_GETMINMAXINFO => {
            // The application bar must remain usable at the smallest restored size.
            if let Some(info) = unsafe { (lparam as *mut MINMAXINFO).as_mut() } {
                info.ptMinTrackSize.x = 640;
                info.ptMinTrackSize.y = 480;
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

fn window_rect(width: i32, height: i32) -> Result<RECT, String> {
    let mut rect = RECT {
        left: 0,
        top: 0,
        right: width,
        bottom: height,
    };
    if unsafe { AdjustWindowRectEx(&mut rect, STYLE, 0, 0) } == 0 {
        return Err(win_error("size viewport window frame"));
    }
    Ok(rect)
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
