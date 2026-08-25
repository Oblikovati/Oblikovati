// RTPathPayload (#2155) is the location-0 ray payload shared by pathtrace_realistic.rgen,
// pathtrace_realistic.rchit and pathtrace_realistic.rmiss: it carries a traced ray's
// result back to whoever fired it (radiance, and how far it traveled — hitDistance, used
// by the caller for Beer's-law extinction over the real distance spent inside a solid
// transmissive medium) and, in the other direction, the caller's own recursion state that
// the callee needs to bound/continue the chain (depth, and the nested-dielectric IOR
// stack). A single struct at one location, reused for BOTH directions and for the
// recursive continuation-ray call pathtrace_realistic.rchit fires on itself: GLSL ray
// payloads are read/write storage for the shader instance that declares them
// (rayPayloadInEXT), so the SAME variable doubles as the "in" contract (set by the
// caller before traceRayEXT) and the "out" result (read by the caller after it returns)
// — see pathtrace_realistic.rchit's own doc comment for why this needs no second payload
// location the way the existing shadow-ray bool (location 1) does.
//
// Distinct from (and independent of) pathtrace.rchit's plain `vec3 payload` — that other,
// single-bounce PBI-345 test-harness pipeline is untouched by #2155.

#ifndef OBK_RT_PAYLOAD_GLSL
#define OBK_RT_PAYLOAD_GLSL

#include "openpbr/extended_lobes.glsl"

struct RTPathPayload {
    vec3 radiance;
    float hitDistance;
    int depth;
    float iorStack[OPENPBR_MAX_TRANSMISSION_BOUNCES + 2];
    int iorStackDepth;
    // channelIOR/channelMask implement dispersion (#2155): pathtrace_realistic.rgen sets
    // these once before firing a chain (the material's own IOR / mask=1 when
    // dispersion_scale<=0, else one of 3 Cauchy-dispersed per-channel IORs / a one-hot
    // mask, looping 3 times) and every recursive continuation in
    // pathtrace_realistic.rchit propagates them unchanged — see that file's doc comment
    // for why dispersion is decided once per chain rather than re-branching at every
    // bounce (which would triple the ray count PER LEVEL instead of per chain).
    float channelIOR;
    vec3 channelMask;
};

#endif // OBK_RT_PAYLOAD_GLSL
