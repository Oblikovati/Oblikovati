# ADR-0064 — One FMA policy: a product is rounded where it is made

**Status:** Accepted — on `m48/fma-policy` (Oblikovati#3528). · **Builds on**
[ADR-0061](ADR-0061-csg-fallback-retirement.md) §"Platform stability, measured", which found the
divergence, fixed the four decisions that could see it, and recorded that the arithmetic itself was
still platform-dependent. · **Deletes:** `kernel/predicates`'s local `rounded` helper and the
package-private policy it carried, replaced by one repo-wide rule with one spelling. ·
**Touches:** `math`, `kernel/geom`, `kernel/predicates`, `archguard`, `Makefile`.

## Context

The kernel ground rules require that **output is byte-identical across runs and platforms**. It is
not. The Go specification licenses the compiler to contract floating-point arithmetic:

> An implementation may combine multiple floating-point operations into a single fused operation,
> possibly across statements, and produce a result that differs from the value obtained by executing
> and rounding the instructions individually.

gc takes that licence on arm64 and never on amd64. The macOS CI runners are arm64; the Linux and
Windows ones are amd64. So the same source computes different last bits on one leg of the matrix,
and ADR-0061 recorded twelve rows that failed there and nowhere else. Four of them were decisions
taken at a precision finer than the quantity deciding them, and each was fixed at its root. The
arithmetic was left as it was, and that residual is this ADR.

Everything below was measured on go1.27.0: amd64 natively, arm64 by cross-compiling and running the
binary under `docker run --platform linux/arm64` with `binfmt` (the recipe ADR-0061 records, now
`make arm64`).

### What actually fuses

The witness is `a=0.1, b=0.2, c=-0.02`: separately rounded the sum is `0x3c50000000000000`, fused it
is `0x3c40a3d70a3d70a4`.

| form | amd64 | arm64 |
| --- | --- | --- |
| `a*b + c` | separate | **fused** |
| `p := a * b` … `p + c` | separate | **fused** |
| `box{a * b}.v + c` (through a struct field) | separate | **fused** |
| `ident(a*b) + c` (an inlined identity function) | separate | **fused** |
| `p.Add(v.Scale(s))` — the product is returned in a composite literal and the caller adds it | separate | **fused** |
| `a + b/2` (a power-of-two divide) | separate | **fused** — gc strength-reduces it to `a + b*0.5` |
| `a + b/3` | separate | separate |
| `float64(a*b) + c` | separate | separate |
| `Scalar(a*b) + c` (`Scalar` is an alias for `float64`) | separate | separate |
| the same with `Scale` returning `Vector3{float64(v.X * s), …}` | separate | separate |
| `rnd(a*b) + c` where `rnd(x) = float64(x)`, inlined | separate | separate |
| `complex128(a*b) + c` | separate | **fused** |

Five things follow, and they are the whole design:

1. **The rule cannot be written on the SUM.** Every "no add may sit next to a product" formulation
   is defeated by a local, a struct field or an inlined call — all of which the compiler sees
   through. The rule has to be written on the PRODUCT, where it is made.
2. **An explicit conversion is what blocks it**, and it is the only thing that does. The spec
   guarantee is narrow and exact: "an explicit floating-point type conversion rounds to the
   precision of the target type."
3. **A conversion inside an inlinable helper survives inlining.** `kernel/predicates.rounded` has
   relied on that since #2020 and it still holds. It is nevertheless not the form chosen here: see
   the decision below.
4. **Division is not exempt.** A divide by a constant with an exact reciprocal becomes a multiply
   before the contraction pass runs, so `a + b/2` is a fusion site and `a + b/3` is not. The rule
   covers `/` because deciding which divides are strength-reducible is the compiler's business,
   not the guard's — and gc propagates constants through variables, so no syntactic test is sound.
5. **`complex128` cannot be fixed this way.** A conversion between complex types is not a
   floating-point conversion, and complex division runs inside `runtime.complex128div`, which this
   repo does not compile. Complex arithmetic is therefore excluded from the policy and ratcheted
   instead.

### The standard library fuses too, and that is not all it does

`math.Sin`, `math.Atan2`, `math.Log` and their neighbours are ordinary Go compiled with the same
rules, so their own product-sums contract on arm64. Checksums over 200 000 pseudo-random arguments
per function (`.fmaprobe/libm`), amd64 against arm64:

| | functions that agree |
| --- | --- |
| as shipped | 1 of 20 (`Sqrt` — an IEEE-754 exactly-rounded primitive) |
| arm64 rebuilt with `-gcflags=all=-d=fmahash=<never matches>` | 14 of 20 |

The 13 that come back — `Acos Asin Atan Atan2 Cbrt Cos Hypot Log Log10 Log1p Mod Sin Tan` — were
differing for exactly this reason. The 6 that do not — `Exp Pow Sinh Cosh Tanh Erf` — differ for a
SECOND reason: amd64 has a hand-written `archExp` (`src/math/exp_amd64.s`) whose inner loop branches
at run time on `cpu.X86.HasAVX && cpu.X86.HasFMA`. Measured: `GODEBUG=cpu.fma=off` on amd64
reproduces the arm64 bits of `Exp(1.7)` exactly (`0x4015e552770df8a6` against `0x4015e552770df8a7`).

**So `math.Exp` is not byte-stable across amd64 machines either.** That is a fact about this
project's dependency, not about its code, and no source rule in this repo can change it. It is
recorded here so that "byte-identical across platforms" is read for what it is: a property this
repo can deliver for the arithmetic it OWNS, and not yet for the elementary functions it calls.

## Decision

**A floating-point multiplication or division is explicitly rounded where it is made, unless its
value is consumed immediately by another multiplication, division or comparison.**

The conversion is spelled `Scalar(x)` in `math` — where the package's own scalar alias keeps the
policy correct if `Scalar` is ever flipped — and `float64(x)` in `kernel/geom` and
`kernel/predicates`. One spelling per package, no helper, no import.

Scope, this ADR: `math`, `kernel/geom`, `kernel/predicates` — the arithmetic floor and the exact
predicates, which is where a fused rounding is not merely non-deterministic but WRONG: Shewchuk's
static filter certifies a sign against an a-priori bound derived on the assumption that every
operation rounds separately.

The two exemptions are not conveniences, they are facts about the machine: there is no
fused-multiply-multiply, and arm64 has no fused multiply-compare, so a product consumed by `*`, `/`
or a relational operator cannot be contracted and gains nothing from a conversion.

`_test.go` files are OUT of scope and the guard does not read them. A test's own arithmetic is not
kernel output; where a test computes an expectation that a contraction could move, the fix is to
compare it at a tolerance or to pin the value, not to convert the whole corpus. The two exceptions
are deliberate and live in `math/unfused_test.go`, where the unconverted form IS the subject.

### Enforcement

- `archguard.TestNoFusableProductSums` type-checks the three packages FROM SOURCE and reports every
  product or quotient the rule binds that is not rounded. From source, because the gc export-data
  importer cannot see `oblikovati.org/math` or `oblikovati.org/api` — neither is in GOROOT or
  GOPATH — and it fails SILENTLY, leaving the affected expressions untyped: measured on the
  unconverted tree it finds 1307 of the 1495 sites, and on the converted one it invents 24 that are
  not there.
  The budget is zero. `complexFusionDebt` pins the complex128 sites the policy cannot reach; it may
  fall, never rise.
- `math.TestExplicitConversionBlocksContraction` is the runtime half: it fails if a conversion ever
  stops forcing a rounding, and it fails on the macOS leg, where the contraction actually happens.
  Its companion pins the other side — that the unconverted form still contracts on arm64 and still
  does not on amd64 — so the cost argument below cannot go stale unnoticed.
- `make arm64-fma PKG=./kernel/geom` disassembles the arm64 build and counts the fused instructions
  the compiler actually emitted. That is the completeness oracle: the AST walk checks the source,
  this checks what was done with it.
- `make arm64 PKG=… [RUN=…]` runs one package's tests under emulation, so a developer can see the
  macOS leg before pushing.

### Rejected: `math.FMA` everywhere

Fusing DELIBERATELY on both platforms is the other way to make the platforms agree, and it is
better arithmetic — one rounding instead of two. It was rejected on cost, not on principle. amd64
has never contracted, so every stored fingerprint in this repo (88 mesh pins in
`model/feature/occtparity` alone), every oracle number in ADR-0042..0063, and every recorded
DRAWEXE comparison is an amd64 unfused value. Adopting `math.FMA` moves all of them at once, and a
rebaseline of that size buys a last-bit improvement that no gate in this repo can see. The unfused
form reproduces today's amd64 semantics exactly, which is what makes it a policy that can be
adopted in one wave instead of negotiated across every corpus.

### Rejected: `-gcflags=all=-d=fmahash=<never matches>`

Disabling the contraction with a build flag is one line and covers the standard library too (it is
what produced the 14-of-20 measurement above). It was rejected because it is a compiler DEBUG flag
with no compatibility promise, it does not travel with the source to anyone who builds this module
another way, and a policy that lives in a flag is invisible at the call site. It stays what it is: a
measurement instrument, and the way to prove a residual is the contraction and not something else.

## Consequences

### Conversions

The rule binds 1495 products and quotients across the three packages, and every one of them is
converted:

| package | conversions | files |
| --- | --- | --- |
| `math` | 244 | 18 |
| `kernel/geom` | 1211 | 124 |
| `kernel/predicates` | 40 — 14 new, and 26 `rounded(…)` calls respelled | 3 |

`kernel/predicates.rounded` is deleted. Its docstring was the policy in miniature and its job is now
the guard's; the package doc keeps the reasoning and points here.

### The oracle: fused instructions emitted for arm64

| package | before | after |
| --- | --- | --- |
| `math` | 336 | **0** |
| `kernel/geom` | 2731 | **10** |
| `kernel/predicates` | 7 | **0** |

The remaining 10 are five source sites in `kernel/geom/quartic_real_roots.go` — Ferrari's quartic
factoring, written in `complex128`. Closing them means writing that solve in real arithmetic, which
changes its amd64 results (Go's complex division is Smith's algorithm inside the runtime, not the
naive formula), so it is its own change with its own corpus. `complexFusionDebt` holds the line
meanwhile.

### Cost

amd64 pays nothing, because it never contracted: the conversion is a no-op move that the compiler
elides.

| | amd64 | arm64 |
| --- | --- | --- |
| `math` instructions emitted | 14137 → 14162 (+0.18%) | 13089 → 13394 (**+2.3%**) |
| `kernel/geom` instructions emitted | 204467 → 204461 (−0.003%) | 184842 → 187413 (**+1.4%**) |
| `math` FP arithmetic instructions | — | 860 (336 of them fused) → 1139 (0 fused) |

Wall clock on amd64 is inside the noise over five runs of each benchmark (`Vector3.Dot` 2.44 → 2.48
ns, `Matrix4.Mul` 20.5 → 20.5 ns, `Matrix4.TransformPoint` 3.25 → 3.15 ns). On arm64 the only
honest number here is the static one: the wall-clock runs go through an emulator, whose translation
makes `Matrix4.Mul` read +10% and `Matrix4.TransformPoint` read −26% in the same pair of runs. Read
the instruction counts as the cost and expect a real arm64 core to pay somewhat more in a dependent
chain, where an `FMADD`'s latency replaces an `FMUL` plus an `FADD` rather than one instruction.

That is the price of the ground rule, and it is small: the arithmetic floor is 1–2% denser on one
of three platforms, and in exchange the kernel computes the same numbers on all of them.

### What this does NOT achieve

Bit-identity across platforms is **not** reached by this ADR, and the honest reasons are all above:

- **the standard library still contracts.** `math.Sin`, `math.Atan2`, `math.Log`, `Cbrt`, `Hypot`
  and their neighbours are ordinary Go, compiled with the same licence, and this repo does not
  compile them. Every one of them returns different last bits on arm64 than on amd64, and no rule
  written here can change that.
- **`math.Exp` is worse than platform-dependent, it is CPU-dependent**, and `Pow`, `Sinh`, `Cosh`,
  `Tanh` and `Erf` inherit it.
- **the kernel above the arithmetic floor is out of scope.** `kernel/brep`, `kernel/ops/*`,
  `kernel/topo`, `kernel/mesh` and `model/feature` still contract.

What that costs, measured over the 88 `occtparity` byte-identity pins (arm64 against the amd64
values the pins hold), at this ADR's base and at its HEAD:

| of the 88 pinned bodies | base | HEAD |
| --- | --- | --- |
| bit-identical to the amd64 pin | 23 | **42** |
| hash differs, same triangle count | 49 | 43 |
| triangle COUNT differs | 16 | **3** |

Twenty-one bodies became bit-identical and the structural half of the drift — a refinement decision
landing on the other side, which is what moves a triangle count — went from 16 rows to 3. Two went
the other way (`bfuseblend/A6`, `simple/D8`): a body whose remaining fused arithmetic is fed
differently lands differently, which is what a PARTIAL conversion means. The residual is now a
NAMED, ratcheted set (`crossArchHashDrift`) rather than a skipped test, and it shrinks package by
package: `make arm64-fma` to zero, then re-measure the pins.

Closing the standard-library half needs this project to own a portable elementary-function layer.
That is a separate decision, and it is the real ceiling on the ground rule as written.

### The perturbation row

A policy that removes a 1-ulp difference is worth little if a 1-ulp difference could still flip a
decision. `TestTheFigureEightHoldsUnderAnUlpPerturbation`
(`kernel/ops/boolean/torus_figure_eight_ulp_test.go`) moves the figure-eight fixture's inputs by
whole ulps of the model's own scale and asserts the full decision set — both booleans resolve, the
inventory is unchanged, and the two torus faces still partition the torus at BOTH facetings.

It holds at ±8 ulps on the torus centre, which is not a bifurcation parameter, and on the TANGENCY
OFFSET it holds at 0 and +1 ulp and nowhere below. The measurement is the intersect piece's
torus-face mesh area at the property faceting; its own share is 111.68 mm² and 394.75 is the whole
torus, i.e. the classification has taken the entire tube for the cap:

| k (ulps of 5) | −3 | −2 | −1 | 0 | +1 | +2 |
| --- | --- | --- | --- | --- | --- | --- |
| amd64 | 394.75 | 394.75 | 111.67 | 111.67 | 111.67 | 111.67 |
| arm64 | 394.75 | 111.67 | 394.75 | 111.67 | 111.67 | 394.75 |

amd64 has a boundary at −2; arm64 has none — the verdict alternates. **Within a few ulps of the
tangency this decision is not a function of the geometry, it is a function of the rounding.** The
same piece meshes its own share at the DEFAULT faceting in every one of those cells, so the face's
region is being read differently at two facetings.

That is pre-existing and outside this ADR: amd64 reproduces bit-for-bit at the base `6590a9ba` in a
clean worktree, and the classification lives in `kernel/brep` and `kernel/ops`, which this ADR does
not convert. `TestTheFigureEightMisreadsAnOffsetJustBelowTheTangency` pins it — at least one of −1,
−2, −3 ulps still returns the whole torus, on every platform — so the band cannot stay narrow by
inattention, and when the classification is fixed the pin fails and asks for the band to widen.

It is also the sharpest available statement of what this ADR does NOT buy. Converting the
arithmetic floor made 21 more corpus bodies bit-identical across platforms; it did not make this
decision stop depending on the last bit, because the decision is not taken in the packages it
converts.
