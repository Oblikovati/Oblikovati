# API release candidate review progress

This is a partial review. It does not approve a merge or release.

- Released base: `f0943c724a6e07ec49533769c9e680137bb3d55c` (`v0.159.0`).
- Candidate: `a87a7fc945232c1d73c67c6dcfd08967e4738aa3` on `work/api-release-3965`.
- The comparison contains 247 changed files. The candidate worktree was clean.
- `git diff --check` found no whitespace errors. The Go and Rust dependency manifests and lockfiles have no changes in this range.
- The candidate version tool reports `next=0.160.0` and `bump=breaking` for the release range. The changelog describes the driving-dimension migration.
- The local API gate passed. Go aggregate statement coverage was 86.5%; Rust line coverage was 97.77%. See the adjacent gate and coverage records for scope and limits.
- The API wiki generator warned about unresolved references to `Material.AppearanceID`, `Parameter.NominalValue`, `Parameter.UnitName`, and `Arg.Kind`. The source references are unchanged from the base and the methods exist, but the generated wiki shows the references as plain text.
- The candidate changes CI and release scripts. Their behavior has been inspected in part; the full changed tree and release interactions still need review. Do not treat the local gate as a security audit.
- API issues 187 and 301 through 304 remain open. Their release readiness must be established under the user's instruction to resolve API issues before release.
- The [hosted candidate run](https://github.com/Oblikovati/Oblikovati.API/actions/runs/44312785242) passed Linux and Windows. macOS was cancelled without starting on attempt 1 and passed on attempt 2. The release script requires its own exact candidate run on attempt 1, so this is preflight evidence only.

The GitHub workflow dispatch response shape used by the candidate was checked against the [GitHub REST documentation](https://docs.github.com/en/rest/actions/workflows#create-a-workflow-dispatch-event). The candidate has not been promoted or released.
