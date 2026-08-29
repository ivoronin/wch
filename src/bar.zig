//! Draws live or history status in the bottom row.

const std = @import("std");
const vaxis = @import("vaxis");
const zeit = @import("zeit");

const clock_width = 8;
const edge_padding: u16 = 1;
const history_spacing: u16 = 2;

const bar_style: vaxis.Style = .{ .bg = .{ .index = 8 }, .fg = .{ .index = 15 } };
const shortcut_style: vaxis.Style = .{ .bg = bar_style.bg, .fg = bar_style.fg, .bold = true };

const live_help: []const vaxis.Segment = &.{
    .{ .text = "b", .style = shortcut_style },
    .{ .text = " hist", .style = bar_style },
    .{ .text = " · ", .style = bar_style },
    .{ .text = "q", .style = shortcut_style },
    .{ .text = " quit", .style = bar_style },
};

const history_help: []const vaxis.Segment = &.{
    .{ .text = "j", .style = shortcut_style },
    .{ .text = " prev", .style = bar_style },
    .{ .text = " · ", .style = bar_style },
    .{ .text = "k", .style = shortcut_style },
    .{ .text = " next", .style = bar_style },
    .{ .text = " · ", .style = bar_style },
    .{ .text = "b", .style = shortcut_style },
    .{ .text = " live", .style = bar_style },
    .{ .text = " · ", .style = bar_style },
    .{ .text = "q", .style = shortcut_style },
    .{ .text = " quit", .style = bar_style },
};

const StatusIndicator = enum {
    idle,
    running,
    history,

    /// Return the glyph for this state.
    fn glyph(self: StatusIndicator) []const u8 {
        return switch (self) {
            .idle => "·",
            .running => "*",
            .history => "←",
        };
    }
};

pub const StatusBar = struct {
    pub const height = 1;

    /// Borrowed until the status bar is released.
    command_label: []const u8,
    timezone: zeit.TimeZone,
    // Vaxis cells borrow formatted text until the frame is rendered.
    clock_buffer: [clock_width]u8 = undefined,
    run_position_buffer: [64]u8 = undefined,

    /// Create a status bar using local time when available.
    pub fn init(
        allocator: std.mem.Allocator,
        io: std.Io,
        command_label: []const u8,
    ) StatusBar {
        return .{
            .command_label = command_label,
            .timezone = zeit.local(allocator, io, .{}) catch zeit.utc,
        };
    }

    /// Release timezone storage.
    pub fn deinit(self: *StatusBar) void {
        self.timezone.deinit();
        self.* = undefined;
    }

    /// Draw the command, latest time, activity, and live-mode help.
    pub fn drawLive(
        self: *StatusBar,
        window: vaxis.Window,
        last_run_at: ?std.Io.Timestamp,
        run_in_progress: bool,
    ) void {
        window.fill(.{ .style = bar_style });
        const help_column = window.width -|
            window.print(live_help, .{ .wrap = .none, .commit = false }).col -|
            edge_padding;
        _ = window.print(live_help, .{ .col_offset = help_column, .wrap = .none });

        const clock_text = if (last_run_at) |completed_at|
            formatClockTime(self, completed_at)
        else
            "";
        const clock_column = clockColumn(window.width);
        const indicator_column = clock_column +| clock_width +| 1;
        const indicator_text = (if (run_in_progress) StatusIndicator.running else StatusIndicator.idle).glyph();

        const middle_status_visible = indicator_column +| window.gwidth(indicator_text) <= help_column;
        if (middle_status_visible) {
            _ = window.printSegment(
                .{ .text = clock_text, .style = bar_style },
                .{ .col_offset = clock_column, .wrap = .none },
            );
            _ = window.printSegment(
                .{ .text = indicator_text, .style = bar_style },
                .{ .col_offset = indicator_column, .wrap = .none },
            );
        }

        const command_end_column = if (middle_status_visible) clock_column else help_column;
        self.drawCommand(window, command_end_column);
    }

    /// Draw history position, time, and navigation help.
    pub fn drawHistory(
        self: *StatusBar,
        window: vaxis.Window,
        run_number: usize,
        run_count: usize,
        completed_at: std.Io.Timestamp,
    ) void {
        window.fill(.{ .style = bar_style });
        const clock_column = clockColumn(window.width);
        _ = window.printSegment(
            .{ .text = formatClockTime(self, completed_at), .style = bar_style },
            .{ .col_offset = clock_column, .wrap = .none },
        );
        _ = window.printSegment(
            .{ .text = StatusIndicator.history.glyph(), .style = bar_style },
            .{ .col_offset = clock_column +| clock_width +| 1, .wrap = .none },
        );

        const run_position_text = std.fmt.bufPrint(
            &self.run_position_buffer,
            "Run {d}/{d}",
            .{ run_number, run_count },
        ) catch unreachable;
        if (edge_padding +| window.gwidth(run_position_text) +| history_spacing <= clock_column)
            _ = window.printSegment(
                .{ .text = run_position_text, .style = bar_style },
                .{ .col_offset = edge_padding, .wrap = .none },
            );

        const help_column = window.width -|
            window.print(history_help, .{ .wrap = .none, .commit = false }).col -|
            edge_padding;
        if (clock_column +| clock_width +| history_spacing <= help_column)
            _ = window.print(history_help, .{ .col_offset = help_column, .wrap = .none });
    }

    /// Draw the command before an end column and mark truncation.
    fn drawCommand(self: *const StatusBar, window: vaxis.Window, end_column: u16) void {
        const command_window = window.child(.{
            .x_off = edge_padding,
            .width = end_column -| edge_padding -| edge_padding,
        });
        if (command_window.printSegment(
            .{ .text = self.command_label, .style = bar_style },
            .{ .wrap = .none },
        ).overflow)
            command_window.writeCell(command_window.width -| 1, 0, .{
                .char = .{ .grapheme = "…", .width = 1 },
                .style = bar_style,
            });
    }
};

/// Format a timestamp as local `HH:MM:SS` in the status bar buffer.
fn formatClockTime(
    self: *StatusBar,
    timestamp: std.Io.Timestamp,
) []const u8 {
    const local_time = zeit.instant(.{ .unix_nano = timestamp.nanoseconds }, &self.timezone).time();
    return std.fmt.bufPrint(
        &self.clock_buffer,
        "{d:0>2}:{d:0>2}:{d:0>2}",
        .{ local_time.hour, local_time.minute, local_time.second },
    ) catch unreachable;
}

/// Center the fixed-width clock.
fn clockColumn(bar_width: u16) u16 {
    return (bar_width -| clock_width) / 2;
}
