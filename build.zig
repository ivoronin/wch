const std = @import("std");

/// Configure the executable, dependencies, and developer commands.
pub fn build(build_system: *std.Build) void {
    const target = build_system.standardTargetOptions(.{});
    const optimization = build_system.standardOptimizeOption(.{});
    const version = build_system.option([]const u8, "version", "Application version") orelse "dev";

    // One module. The program is the only thing that reads this source, so
    // there is nobody to hand a library module to.
    const executable = build_system.addExecutable(.{
        .name = "wch",
        .root_module = build_system.createModule(.{
            .root_source_file = build_system.path("src/main.zig"),
            .target = target,
            .optimize = optimization,
        }),
    });

    const build_metadata = build_system.addOptions();
    build_metadata.addOption([]const u8, "version", version);
    executable.root_module.addOptions("build_metadata", build_metadata);

    const dizzy = build_system.dependency("dizzy", .{});
    const clap = build_system.dependency("clap", .{});
    const zeit = build_system.dependency("zeit", .{});
    const vaxis = build_system.dependency("vaxis", .{
        .target = target,
        .optimize = optimization,
    });

    // diff.zig aligns text with dizzy.
    executable.root_module.addImport("dizzy", dizzy.module("dizzy"));

    // args.zig reads the command line with clap.
    executable.root_module.addImport("clap", clap.module("clap"));

    // bar.zig shows a clock, which means a local time zone.
    executable.root_module.addImport("zeit", zeit.module("zeit"));

    // output.zig prepares vaxis segments, and viewport.zig paints them.
    executable.root_module.addImport("vaxis", vaxis.module("vaxis"));

    build_system.installArtifact(executable);

    // Depends on the install step, so it runs from the install directory
    // rather than from within the cache.
    const run_command = build_system.addRunArtifact(executable);
    run_command.step.dependOn(build_system.getInstallStep());
    // `zig build run -- arg1 arg2`.
    if (build_system.args) |application_arguments| run_command.addArgs(application_arguments);

    build_system.step("run", "Run the app").dependOn(&run_command.step);

    const test_executable = build_system.addTest(.{ .root_module = executable.root_module });

    build_system.step("test", "Run tests").dependOn(
        &build_system.addRunArtifact(test_executable).step,
    );
}
