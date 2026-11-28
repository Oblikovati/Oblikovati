// SPDX-License-Identifier: GPL-2.0-only
// Minimal WASI-free C/C++ runtime and stdio stubs for Dear ImGui on wasm32-unknown-unknown.
#include <stddef.h>
#include <stdint.h>

void *__dso_handle = 0;

int __cxa_atexit(void (*func)(void *), void *arg, void *dso_handle) {
    (void)func;
    (void)arg;
    (void)dso_handle;
    return 0;
}

int __cxa_guard_acquire(uint64_t *guard) {
    uint8_t *flag = (uint8_t *)guard;
    return __atomic_load_n(flag, __ATOMIC_ACQUIRE) == 0 ? 1 : 0;
}

void __cxa_guard_release(uint64_t *guard) {
    uint8_t *flag = (uint8_t *)guard;
    __atomic_store_n(flag, 1, __ATOMIC_RELEASE);
}

void __assert_fail(const char *expr, const char *file, unsigned int line, const char *func) {
    (void)expr;
    (void)file;
    (void)line;
    (void)func;
    __builtin_trap();
}

// Dear ImGui disables ini/log files by passing NULL filenames; stub FILE* symbols so
// linking libc.a for pure string/format routines never pulls in WASI syscalls.
void *stdout = 0;

void *fopen(const char *path, const char *mode) {
    (void)path;
    (void)mode;
    return 0;
}

int fclose(void *stream) {
    (void)stream;
    return 0;
}

size_t fread(void *ptr, size_t size, size_t nmemb, void *stream) {
    (void)ptr;
    (void)size;
    (void)nmemb;
    (void)stream;
    return 0;
}

int fseek(void *stream, long offset, int whence) {
    (void)stream;
    (void)offset;
    (void)whence;
    return -1;
}

long ftell(void *stream) {
    (void)stream;
    return -1;
}

int fflush(void *stream) {
    (void)stream;
    return 0;
}

int printf(const char *fmt, ...) {
    (void)fmt;
    return 0;
}

// Intercept musl's internal FILE lock and low-level I/O hooks so pure memory
// formatting (`vsnprintf`/`snprintf`/`sscanf`) from libc.a never pulls in
// `wasi_snapshot_preview1` imports (`fd_write`, `fd_seek`, `fd_close`, `clock_time_get`).
int __lockfile(void *f) {
    (void)f;
    return 0;
}

void __unlockfile(void *f) {
    (void)f;
}

size_t __stdio_write(void *f, const unsigned char *buf, size_t len) {
    (void)f;
    (void)buf;
    return len;
}

size_t __stdio_read(void *f, unsigned char *buf, size_t len) {
    (void)f;
    (void)buf;
    (void)len;
    return 0;
}

int __stdio_close(void *f) {
    (void)f;
    return 0;
}

long long __stdio_seek(void *f, long long off, int whence) {
    (void)f;
    (void)off;
    (void)whence;
    return -1;
}

int clock_gettime(int clk, void *ts) {
    (void)clk;
    (void)ts;
    return 0;
}

int __clock_gettime(int clk, void *ts) {
    (void)clk;
    (void)ts;
    return 0;
}

int __timedwait(void *addr, int val, int clk, const void *at, int priv) {
    (void)addr;
    (void)val;
    (void)clk;
    (void)at;
    (void)priv;
    return 0;
}
