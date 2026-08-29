//! Draws live or history status in the bottom row.

const std = @import("std");
const vaxis = @import("vaxis");
const zeit = @import("zeit");

const clock_width = 8;
const edge_padding: u16 = 1;
const section_spacing: u16 = 2;

const bar_style: vaxis.Style = .{ .bg = .{ .index = 8 }, .fg = .{ .index = 15 } };
const shortcut_style: vaxis.Style = .{ .bg = bar_style.bg, .fg = bar_style.fg, .bold = true };
const ellipsis_cell: vaxis.Cell = .{ .char = .{ .grapheme = "…" }, .style = bar_style };

const live_help: []const vaxis.Segment = &.{
    .{ .text = "b", .style = shortcut_style },
    .{ .text = " hist · ", .style = bar_style },
    .{ .text = "q", .style = shortcut_style },
    .{ .text = " quit", .style = bar_style },
};

const history_help: []const vaxis.Segment = &.{
    .{ .text = "j", .style = shortcut_style },
    .{ .text = " prev · ", .style = bar_style },
    .{ .text = "k", .style = shortcut_style },
    .{ .text = " next · ", .style = bar_style },
    .{ .text = "b", .style = shortcut_style },
    .{ .text = " live · ", .style = bar_style },
    .{ .text = "q", .style = shortcut_style },
    .{ .text = " quit", .style = bar_style },
};

pub const StatusBar = struct {
    pub const height = 1;

    /// Borrowed until the status bar is released.
    command_label: []const u8,
    timezone: zeit.TimeZone,
    // Vaxis cells borrow formatted text until the frame is rendered.
    clock_buffer: [clock_width]u8 = undefined,
    status_buffer: [64]u8 = undefined,

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
        const clock_text = if (last_run_at) |completed_at|
            formatClockTime(self, completed_at)
        else
            "";
        self.draw(window, clock_text, if (run_in_progress) "*" else "·", live_help);
    }

    /// Draw the command, history position, time, and navigation help.
    pub fn drawHistory(
        self: *StatusBar,
        window: vaxis.Window,
        run_number: usize,
        run_count: usize,
        completed_at: std.Io.Timestamp,
    ) void {
        const status_text = std.fmt.bufPrint(
            &self.status_buffer,
            "← {d}/{d}",
            .{ run_number, run_count },
        ) catch unreachable;
        self.draw(window, formatClockTime(self, completed_at), status_text, history_help);
    }

    /// Draw one bar using the same geometry in both modes.
    fn draw(
        self: *const StatusBar,
        window: vaxis.Window,
        clock_text: []const u8,
        status_text: []const u8,
        help: []const vaxis.Segment,
    ) void {
        window.fill(.{ .style = bar_style });
        const clock_column = clockColumn(window.width);
        const status_column = clock_column + clock_width + 1;
        const status_end = status_column + window.gwidth(status_text);

        _ = window.printSegment(
            .{ .text = clock_text, .style = bar_style },
            .{ .col_offset = clock_column, .wrap = .none },
        );
        if (status_end > window.width) return;
        _ = window.printSegment(
            .{ .text = status_text, .style = bar_style },
            .{ .col_offset = status_column, .wrap = .none },
        );
        const command_window = window.child(.{
            .x_off = edge_padding,
            .width = clock_column -| edge_padding -| section_spacing,
        });
        if (command_window.printSegment(
            .{ .text = self.command_label, .style = bar_style },
            .{ .wrap = .none },
        ).overflow)
            command_window.writeCell(command_window.width -| 1, 0, ellipsis_cell);

        const help_start = status_end + section_spacing;
        const help_available = window.width -| help_start -| edge_padding;
        if (help_available > 0) {
            var help_width: u16 = 0;
            for (help) |segment| help_width +|= window.gwidth(segment.text);
            const help_offset: i17 = @as(i17, @intCast(help_available)) - @as(i17, @intCast(help_width));
            const help_window = window.child(.{ .x_off = help_start, .width = help_available });
            _ = help_window.child(.{
                .x_off = help_offset,
                .width = help_width,
            }).print(help, .{ .wrap = .none });
            if (help_offset < 0)
                help_window.writeCell(0, 0, ellipsis_cell);
        }
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
