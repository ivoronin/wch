//! Parses watch options and preserves the remaining command arguments.

const std = @import("std");
const clap = @import("clap");
const build_metadata = @import("build_metadata");

const option_parameters = clap.parseParamsComptime(
    \\-h, --help             Display this help and exit.
    \\-v, --version          Display version and exit.
    \\-i, --interval <i64>   Seconds between runs. One by default.
    \\-l, --limit <usize>    How many runs to keep. 3600 by default.
    \\<str>
);

const usage_text = "usage: wch [options] command [args...]\n\noptions:\n";

/// Validated command-line input.
pub const WatchOptions = struct {
    run_interval: std.Io.Duration,
    history_limit: usize,
    /// Command arguments retain their original shell boundaries.
    command_arguments: []const []const u8,
};

/// Parse arena-backed options, printing help or errors before exiting when needed.
pub fn parse(
    arena: std.mem.Allocator,
    io: std.Io,
    process_arguments: std.process.Args,
) !WatchOptions {
    const raw_arguments = try process_arguments.toSlice(arena);
    const user_arguments = try arena.alloc([]const u8, raw_arguments.len - 1);
    for (raw_arguments[1..], user_arguments) |raw_argument, *user_argument|
        user_argument.* = raw_argument;

    var diagnostic: clap.Diagnostic = .{};
    var argument_parser: clap.args.SliceIterator = .{ .args = user_arguments };
    const parsed_arguments = clap.parseEx(
        clap.Help,
        &option_parameters,
        clap.parsers.default,
        &argument_parser,
        .{
            .diagnostic = &diagnostic,
            .allocator = arena,
            // Leave every argument after the command to the command itself.
            .terminating_positional = 0,
        },
    ) catch |parse_error| {
        // The diagnostic is complete; returning the error would add a useless stack trace.
        try diagnostic.reportToFile(io, .stderr(), parse_error);
        std.process.exit(1);
    };
    if (parsed_arguments.args.version != 0) {
        const version_text = try std.fmt.allocPrint(arena, "wch {s}\n", .{build_metadata.version});
        try std.Io.File.stdout().writeStreamingAll(io, version_text);
        std.process.exit(0);
    }
    if (parsed_arguments.args.help != 0 or parsed_arguments.positionals[0] == null) {
        try std.Io.File.stdout().writeStreamingAll(io, usage_text);
        try clap.helpToFile(io, .stdout(), clap.Help, &option_parameters, .{});
        std.process.exit(0);
    }

    return .{
        .run_interval = .fromSeconds(@max(parsed_arguments.args.interval orelse 1, 1)),
        .history_limit = parsed_arguments.args.limit orelse 3600,
        // parseEx stops after consuming the command name.
        .command_arguments = user_arguments[argument_parser.index - 1 ..],
    };
}
