// SPDX-License-Identifier: GPL-2.0-only
#define _GNU_SOURCE
#include <X11/Xlib.h>
#include <link.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

typedef Status (*AttributesCall)(Display *, Window, XWindowAttributes *);
static _Atomic(AttributesCall) next_attributes;
static atomic_uint_fast64_t calls;

enum { SAMPLE_CAPACITY = 262144 };
struct call_sample {
    uint64_t entry_ns;
    uint64_t elapsed_ns;
};
static struct call_sample samples[SAMPLE_CAPACITY];

static uint64_t now_ns(void) {
    struct timespec time;
    clock_gettime(CLOCK_MONOTONIC, &time);
    return (uint64_t)time.tv_sec * 1000000000u + (uint64_t)time.tv_nsec;
}

static Status timed_attributes(Display *display, Window window, XWindowAttributes *out) {
    AttributesCall next = atomic_load_explicit(&next_attributes, memory_order_relaxed);
    uint64_t started = now_ns();
    Status result = next(display, window, out);
    uint64_t elapsed = now_ns() - started;
    uint64_t index = atomic_fetch_add_explicit(&calls, 1, memory_order_relaxed);
    if (index < SAMPLE_CAPACITY) {
        samples[index] = (struct call_sample){started, elapsed};
    }
    return result;
}

unsigned int la_version(unsigned int version) {
    return version >= LAV_CURRENT ? LAV_CURRENT : 0;
}

unsigned int la_objopen(struct link_map *map, Lmid_t namespace, uintptr_t *cookie) {
    (void)map;
    (void)namespace;
    (void)cookie;
    return LA_FLG_BINDTO | LA_FLG_BINDFROM;
}

uintptr_t la_symbind64(Elf64_Sym *symbol, unsigned int index, uintptr_t *reference,
                      uintptr_t *definition, unsigned int *flags, const char *name) {
    (void)index;
    (void)reference;
    (void)definition;
    (void)flags;
    if (strcmp(name, "XGetWindowAttributes") != 0) {
        return symbol->st_value;
    }
    atomic_store_explicit(&next_attributes, (AttributesCall)symbol->st_value, memory_order_relaxed);
    return (uintptr_t)timed_attributes;
}

__attribute__((destructor)) static void save_report(void) {
    const char *path = getenv("OBK_X11_AUDIT_LOG");
    if (path == NULL) {
        return;
    }
    FILE *report = fopen(path, "w");
    if (report != NULL) {
        uint64_t count = atomic_load(&calls);
        fputs("entry_ns,elapsed_ns\n", report);
        uint64_t recorded = count < SAMPLE_CAPACITY ? count : SAMPLE_CAPACITY;
        for (uint64_t index = 0; index < recorded; ++index) {
            fprintf(report, "%llu,%llu\n",
                    (unsigned long long)samples[index].entry_ns,
                    (unsigned long long)samples[index].elapsed_ns);
        }
        if (recorded != count) {
            fprintf(stderr, "X11 attribute audit discarded %llu calls\n",
                    (unsigned long long)(count - recorded));
        }
        fclose(report);
    }
}
