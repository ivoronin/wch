//! Captures one execution of the watched command.

const std = @import("std");

pub const Run = struct {
    completed_at: std.Io.Timestamp,
    /// Owned stderr followed by stdout, with tabs left unchanged.
    output: []const u8,

    /// Run once, returning launch failures as output and cancellation as an error.
    pub fn capture(
        allocator: std.mem.Allocator,
        io: std.Io,
        command_arguments: []const []const u8,
    ) !Run {
        // A fixed shell interprets one command string independently of `$SHELL`.
        const process_arguments: []const []const u8 = if (command_arguments.len == 1)
            &.{ "/bin/sh", "-c", command_arguments[0] }
        else
            command_arguments;

        if (std.process.run(allocator, io, .{ .argv = process_arguments })) |process_result| {
            const captured_output = if (process_result.stderr.len == 0) stdout_only: {
                allocator.free(process_result.stderr);
                break :stdout_only process_result.stdout;
            } else if (process_result.stdout.len == 0) stderr_only: {
                allocator.free(process_result.stdout);
                break :stderr_only process_result.stderr;
            } else both_streams: {
                defer allocator.free(process_result.stdout);
                defer allocator.free(process_result.stderr);
                break :both_streams try std.mem.concat(
                    allocator,
                    u8,
                    &.{ process_result.stderr, process_result.stdout },
                );
            };

            return .{
                .completed_at = .now(io, .real),
                .output = captured_output,
            };
        } else |launch_error| {
            if (launch_error == error.Canceled) return error.Canceled;

            return .{
                .completed_at = .now(io, .real),
                .output = try std.fmt.allocPrint(
                    allocator,
                    "Error running command: {any}",
                    .{launch_error},
                ),
            };
        }
    }

    /// Free captured output.
    pub fn deinit(self: *Run, allocator: std.mem.Allocator) void {
        allocator.free(self.output);
        self.* = undefined;
    }
};
