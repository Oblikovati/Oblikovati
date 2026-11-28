// SPDX-License-Identifier: GPL-2.0-only
//! Rust-backed C/C++ allocator and libm bridge for Dear ImGui on `wasm32-unknown-unknown`.
use std::alloc::{Layout, alloc, alloc_zeroed, dealloc, realloc as std_realloc};

const HEADER_BYTES: usize = 16;

fn allocation_layout(size: usize) -> Option<Layout> {
    let total = size.checked_add(HEADER_BYTES)?;
    Layout::from_size_align(total, HEADER_BYTES).ok()
}

/// Allocate `size` bytes from Rust's global allocator with a 16-byte size header.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn malloc(size: usize) -> *mut u8 {
    let Some(layout) = allocation_layout(size) else {
        return std::ptr::null_mut();
    };
    // SAFETY: `layout` has non-zero size (`size + 16 >= 16`) and valid power-of-two alignment 16.
    let base = unsafe { alloc(layout) };
    if base.is_null() {
        return std::ptr::null_mut();
    }
    // SAFETY: `base` is 16-byte aligned with at least `HEADER_BYTES` writable bytes.
    unsafe {
        base.cast::<usize>().write(size);
        base.add(HEADER_BYTES)
    }
}

/// Allocate zero-initialized memory from Rust's global allocator.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn calloc(count: usize, elem_size: usize) -> *mut u8 {
    let Some(size) = count.checked_mul(elem_size) else {
        return std::ptr::null_mut();
    };
    let Some(layout) = allocation_layout(size) else {
        return std::ptr::null_mut();
    };
    // SAFETY: `layout` has non-zero size (`size + 16 >= 16`) and alignment 16.
    let base = unsafe { alloc_zeroed(layout) };
    if base.is_null() {
        return std::ptr::null_mut();
    }
    // SAFETY: `base` is 16-byte aligned with at least `HEADER_BYTES` writable bytes.
    unsafe {
        base.cast::<usize>().write(size);
        base.add(HEADER_BYTES)
    }
}

/// Free a pointer previously allocated by `malloc`, `calloc`, or `realloc`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn free(ptr: *mut u8) {
    if ptr.is_null() {
        return;
    }
    // SAFETY: `ptr` was returned by `malloc`/`calloc`/`realloc` at `base + HEADER_BYTES`.
    unsafe {
        let base = ptr.sub(HEADER_BYTES);
        let size = base.cast::<usize>().read();
        if let Some(layout) = allocation_layout(size) {
            dealloc(base, layout);
        }
    }
}

/// Reallocate a pointer previously allocated by `malloc`, `calloc`, or `realloc`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn realloc(ptr: *mut u8, new_size: usize) -> *mut u8 {
    if ptr.is_null() {
        // SAFETY: delegates to our `malloc` entrypoint.
        return unsafe { malloc(new_size) };
    }
    // SAFETY: `ptr` was returned by `malloc`/`calloc`/`realloc` at `base + HEADER_BYTES`.
    unsafe {
        let base = ptr.sub(HEADER_BYTES);
        let old_size = base.cast::<usize>().read();
        let Some(old_layout) = allocation_layout(old_size) else {
            return std::ptr::null_mut();
        };
        let Some(new_total) = new_size.checked_add(HEADER_BYTES) else {
            return std::ptr::null_mut();
        };
        let next = std_realloc(base, old_layout, new_total);
        if next.is_null() {
            return std::ptr::null_mut();
        }
        next.cast::<usize>().write(new_size);
        next.add(HEADER_BYTES)
    }
}

/// C++ `operator new(size_t)` routed to the shared Rust global allocator.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn _Znwm(size: usize) -> *mut u8 {
    // SAFETY: delegates to our `malloc` entrypoint.
    unsafe { malloc(size) }
}

/// C++ `operator new[](size_t)` routed to the shared Rust global allocator.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn _Znam(size: usize) -> *mut u8 {
    // SAFETY: delegates to our `malloc` entrypoint.
    unsafe { malloc(size) }
}

/// C++ `operator delete(void*)` routed to the shared Rust global allocator.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn _ZdlPv(ptr: *mut u8) {
    // SAFETY: delegates to our `free` entrypoint.
    unsafe { free(ptr) }
}

/// C++ `operator delete[](void*)` routed to the shared Rust global allocator.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn _ZdaPv(ptr: *mut u8) {
    // SAFETY: delegates to our `free` entrypoint.
    unsafe { free(ptr) }
}
