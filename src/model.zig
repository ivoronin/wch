//! Owns run selection and draws complete screen frames.

const std = @import("std");
const vaxis = @import("vaxis");
const History = @import("history.zig").History;
const Output = @import("output.zig").Output;
const Viewport = @import("viewport.zig").Viewport;
const Run = @import("run.zig").Run;
const StatusBar = @import("bar.zig").StatusBar;

/// Return the screen area above the one-line status bar.
fn viewportWindow(window: vaxis.Window) vaxis.Window {
    return window.child(.{
        .width = window.width,
        .height = window.height -| StatusBar.height,
    });
}

pub const Model = struct {
    history: History,
    output: Output,
    viewport: Viewport,
    status_bar: StatusBar,

    /// Null follows the newest run.
    selected_cursor: ?History.Cursor = null,
    /// Null before the first run is shown.
    displayed_cursor: ?History.Cursor = null,
    run_in_progress: bool = false,

    /// Create an empty model and its owned modules.
    pub fn init(
        allocator: std.mem.Allocator,
        io: std.Io,
        history_limit: usize,
        command_label: []const u8,
    ) Model {
        return .{
            .history = .init(allocator, history_limit),
            .output = .init(allocator),
            .viewport = .{},
            .status_bar = .init(allocator, io, command_label),
        };
    }

    /// Release model-owned storage.
    pub fn deinit(self: *Model) void {
        self.history.deinit();
        self.output.deinit();
        self.status_bar.deinit();
        self.* = undefined;
    }

    /// Record a completed run and leave the model idle.
    pub fn finishRun(self: *Model, run: Run) void {
        self.run_in_progress = false;
        self.history.append(run);
    }

    /// Apply model keys, then pass unclaimed input to the viewport.
    pub fn handleKeyPress(self: *Model, key: vaxis.Key, window: vaxis.Window) void {
        if (key.matches('b', .{})) {
            self.selected_cursor = if (self.selected_cursor == null)
                self.displayed_cursor
            else
                null;
            return;
        }

        if (self.selected_cursor) |selected_cursor| {
            if (key.matches(vaxis.Key.escape, .{})) {
                self.selected_cursor = null;
                return;
            }

            if (key.matches('j', .{})) {
                self.selected_cursor = self.history.previous(selected_cursor);
                return;
            }
            if (key.matches('k', .{})) {
                self.selected_cursor = self.history.next(selected_cursor);
                return;
            }
        }

        self.viewport.scrollWithKey(key, viewportWindow(window));
    }

    /// Resolve selection, refresh output, and draw the viewport and status bar.
    pub fn drawFrame(self: *Model, window: vaxis.Window) std.mem.Allocator.Error!void {
        const viewport_window = viewportWindow(window);
        const status_bar_window = window.child(.{
            .y_off = @intCast(viewport_window.height),
            .width = window.width,
            .height = StatusBar.height,
        });

        const run_view = self.history.resolve(self.selected_cursor);

        if (run_view) |resolved_run_view| try self.showRun(viewport_window, resolved_run_view);
        self.viewport.draw(viewport_window, self.output.display_lines);

        if (self.selected_cursor != null) {
            const selected_run_view = run_view orelse return;
            self.status_bar.drawHistory(
                status_bar_window,
                selected_run_view.run_number,
                selected_run_view.run_count,
                selected_run_view.run.completed_at,
            );
        } else {
            self.status_bar.drawLive(
                status_bar_window,
                if (run_view) |latest_run_view| latest_run_view.run.completed_at else null,
                self.run_in_progress,
            );
        }
    }

    /// Replace displayed output when the resolved history run changes.
    fn showRun(
        self: *Model,
        viewport_window: vaxis.Window,
        run_view: History.View,
    ) std.mem.Allocator.Error!void {
        if (self.displayed_cursor == run_view.cursor) return;

        const viewport_position = self.viewport.position(viewport_window);
        const visible_line = switch (viewport_position) {
            .line => |line_index| line_index,
            else => null,
        };
        const mapped_visible_line = try self.output.replaceText(
            viewport_window.screen.width_method,
            run_view.previous_output,
            run_view.run.output,
            visible_line,
        );
        const next_position: Viewport.Position = if (mapped_visible_line) |line_index|
            .{ .line = line_index }
        else
            viewport_position;
        self.viewport.replaceLines(viewport_window, self.output.display_lines, next_position);

        self.displayed_cursor = run_view.cursor;
    }
};
