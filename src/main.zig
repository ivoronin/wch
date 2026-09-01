const std = @import("std");
const vaxis = @import("vaxis");
const cli = @import("args.zig");
const Model = @import("model.zig").Model;
const Run = @import("run.zig").Run;

/// Events from Vaxis and the command watcher.
const Event = union(enum) {
    key_press: vaxis.Key,
    winsize: vaxis.Winsize,
    run_started,
    run_finished: Run,
};

const EventLoop = vaxis.Loop(Event);

/// Let the terminal translate wheel input into arrow keys without capturing clicks.
fn setAlternateScroll(writer: *std.Io.Writer, enabled: bool) !void {
    try writer.writeAll(if (enabled) "\x1b[?1007h" else "\x1b[?1007l");
    try writer.flush();
}

/// Run the command on schedule and post lifecycle events until cancellation.
fn watchCommand(
    allocator: std.mem.Allocator,
    event_loop: *EventLoop,
    command_arguments: []const []const u8,
    run_interval: std.Io.Duration,
) void {
    while (true) {
        event_loop.postEvent(.run_started) catch return;

        // Cancellation may surface from capture before sleep sees it.
        if (Run.capture(allocator, event_loop.io, command_arguments)) |captured_run| {
            var run = captured_run;
            event_loop.postEvent(.{ .run_finished = run }) catch {
                run.deinit(allocator);
                return;
            };
        } else |capture_error| if (capture_error == error.Canceled) return;

        event_loop.io.sleep(run_interval, .awake) catch return;
    }
}

/// Run the terminal event loop until the user quits or an operation fails.
pub fn main(process: std.process.Init) !void {
    const io = process.io;
    const allocator = process.gpa;

    var terminal_buffer: [1024]u8 = undefined;

    // Parse before entering alternate screen so help and errors remain visible.
    const watch_options = try cli.parse(process.arena.allocator(), io, process.minimal.args);

    // Reverse defer order stops the watcher before terminal teardown.
    var terminal: vaxis.Tty = try .init(io, &terminal_buffer);
    defer terminal.deinit();

    var tui: vaxis.Vaxis = try .init(io, allocator, process.environ_map, .{});
    defer tui.deinit(allocator, terminal.writer());

    var event_loop: EventLoop = .init(io, &terminal, &tui);
    try event_loop.start();
    defer event_loop.stop();

    try tui.enterAltScreen(terminal.writer());
    try tui.queryTerminal(terminal.writer(), .fromSeconds(1));

    // Avoid a lock-taking signal handler when in-band resize works.
    if (!tui.state.in_band_resize) try event_loop.installResizeHandler();
    defer event_loop.uninstallResizeHandler();

    try setAlternateScroll(terminal.writer(), true);
    defer setAlternateScroll(terminal.writer(), false) catch {};

    // The status bar needs a display string; execution keeps argument boundaries.
    const command_label = try std.mem.join(
        process.arena.allocator(),
        " ",
        watch_options.command_arguments,
    );

    var model: Model = .init(allocator, io, watch_options.history_limit, command_label);
    defer model.deinit();

    // Cancel first to wake a blocked producer, then free runs still in the queue.
    var command_watcher = try io.concurrent(
        watchCommand,
        .{
            allocator,
            &event_loop,
            watch_options.command_arguments,
            watch_options.run_interval,
        },
    );
    defer {
        command_watcher.cancel(io);
        while (event_loop.tryEvent() catch null) |event| switch (event) {
            .run_finished => |queued_run| {
                var run = queued_run;
                run.deinit(allocator);
            },
            else => {},
        };
    }

    while (true) {
        // Draw one frame per queued burst, not per input event.
        try event_loop.pollEvent();
        while (try event_loop.tryEvent()) |event| switch (event) {
            .key_press => |key| {
                if (key.matches('q', .{}) or key.matches('c', .{ .ctrl = true })) return;
                model.handleKeyPress(key, tui.window());
            },
            .winsize => |window_size| try tui.resize(allocator, terminal.writer(), window_size),
            .run_started => model.run_in_progress = true,
            .run_finished => |run| model.finishRun(run),
        };

        const window = tui.window();
        window.clear();
        try model.drawFrame(window);
        try tui.render(terminal.writer());
    }
}

test {
    std.testing.refAllDecls(@This());
}
