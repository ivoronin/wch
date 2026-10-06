# Zig 0.17 migration

Updated on 2026-10-06 with the installed Zig 0.17.0 compiler. The migration is applied to this checkout. All 15 application tests and the native release build pass. The preceding prototype also built every release target and ran in a terminal. Application source changes are small; updated dependency pins do most of the work.

The prototype remains in `/tmp/wch-zig17-explore`. Its migration patch was applied to this checkout. These temporary files can disappear; the dependency pins and source changes are now in the project files.

## Required changes

- In `build.zig`, replace access to the removed `Build.args` field with `run_command.addPassthruArgs()`.
- In `src/diff.zig`, replace `StaticBitSet.initEmpty()` with `std.bit_set.Static` and a typed `.empty` initializer.
- In two viewport tests, replace the removed array repetition operator with typed `@splat` expressions.
- In `build.zig.zon`, require Zig 0.17.0 and update the four direct dependency pins below.
- Use `allocator.print(...)` in place of `std.fmt.allocPrint(allocator, ...)` for the command error, version text, and output test.

These compiler API changes were checked against the installed standard library and the prototype's compiler errors. [Build argument API](https://ziglang.org/documentation/0.17.0/std/#std.Build.Step.Run.addPassthruArgs), [Zig 0.17 release announcement](https://ziglang.org/news/0.17.0-released/)

| Dependency | Tested revision | Source |
| --- | --- | --- |
| Dizzy | `1ab5d293050a6ed9dfa02b44d0820e84a32f20b2` | [Upstream commit](https://github.com/neurocyte/dizzy/tree/1ab5d293050a6ed9dfa02b44d0820e84a32f20b2) |
| Clap | `05faf3905e8548f5cc269a8836e154065e70128d` | [Upstream commit](https://github.com/Hejsil/zig-clap/tree/05faf3905e8548f5cc269a8836e154065e70128d) |
| Zeit | `06bb7b462b4218770fb5438ebe6b0c0c5ef18888`, released as `v0.10.0` | [Upstream release](https://github.com/rockorager/zeit/tree/06bb7b462b4218770fb5438ebe6b0c0c5ef18888) |
| Vaxis | `6fd944a27fb3d6f596e981076381a3131f2448b4`, on `zig-0.17` | [Migration branch commit](https://github.com/rockorager/libvaxis/tree/6fd944a27fb3d6f596e981076381a3131f2448b4) |

## Dependency boundary

The existing pins fail before application compilation. Fixing their build scripts exposes further reflection errors in Vaxis's old `uucode` Unicode-table generator. Updating only the compiler or only the application's build script is insufficient.

Vaxis's migration branch supplies the required transitive updates: `uucode` at `ea62149739404a73c202b48a33bf6dd2af4bd9b0` and `sergot/zigimg` at `c701c9f99779d7ddf594dcc6da8f858fd277d61f`. The image dependency comes from a contributor fork. This is the main maintenance tradeoff: adopting 0.17 now requires a Vaxis branch revision and that fork, rather than a released Vaxis dependency set. [Branch manifest](https://github.com/rockorager/libvaxis/blob/6fd944a27fb3d6f596e981076381a3131f2448b4/build.zig.zon)

Injecting a newer `uucode` through Vaxis's external-module option does not resolve its other build and image dependency errors. The migration branch is the smaller route. No dependency forks or patches maintained by this project were needed in the final prototype.

## Verification

- `zig build test`: all 15 tests pass, including the punctuation highlighting regression.
- `zig build -Doptimize=ReleaseSafe`: native ARM macOS build passes.
- ReleaseSafe builds for `x86_64-macos`, `x86_64-linux`, and `aarch64-linux` pass.
- `zig build run -- --version` prints `wch dev`, confirming argument forwarding.
- A native terminal run executes a repeated `printf` command, renders its output, and exits with status 0 after `q`.
- The final prototype also passes tests with immutable Git URLs and package hashes, replacing the initial local-path dependencies.

Cross-built binaries were compiled but not executed. The terminal check covers startup, command capture, rendering, and exit; it does not cover all history and resize interactions. Remote CI and Homebrew installation were not exercised.

The shared test and release workflows use a pinned setup action that selects `minimum_zig_version` from `build.zig.zon` when no version override is supplied. The updated manifest therefore selects Zig 0.17.0 without a duplicate workflow setting. [Pinned setup action](https://github.com/mlugg/setup-zig/blob/d1434d08867e3ee9daa34448df10607b98908d29/action.yml)

Homebrew still needs a published release that contains these changes. The formula already uses the installed `zig` build dependency; it does not require a compiler pin for the tested Zig 0.17.0 installation.
