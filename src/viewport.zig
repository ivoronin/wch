//! Scrolls display-ready terminal lines without retaining them.

const std = @import("std");
const vaxis = @import("vaxis");

const DisplayLine = []const vaxis.Segment;

// Matching heavy glyphs keep rail-free thumbs visible over content.
const vertical_thumb_cell: vaxis.Cell = .{
    .char = .{ .grapheme = "┃", .width = 1 },
    .style = .{ .fg = .{ .index = 8 } },
};
const horizontal_thumb_cell: vaxis.Cell = .{
    .char = .{ .grapheme = "━", .width = 1 },
    .style = .{ .fg = .{ .index = 8 } },
};

const ContentSize = struct { width: u16, height: u16 };

pub const Viewport = struct {
    pub const Position = union(enum) {
        top,
        bottom,
        line: u32,
    };

    top_line: u32 = 0,
    left_column: u32 = 0,
    line_count: usize = 0,
    content_width: u32 = 0,

    /// Resolve the text area after both scrollbars account for each other.
    fn contentSize(self: Viewport, window: vaxis.Window) ContentSize {
        var content_size: ContentSize = .{ .width = window.width, .height = window.height };
        // Each scrollbar can only enable the other, so two passes reach a fixed point.
        for (0..2) |_| {
            content_size.width = window.width -|
                @intFromBool(self.line_count > content_size.height);
            content_size.height = window.height -|
                @intFromBool(self.content_width > content_size.width);
        }
        return content_size;
    }

    /// Report top, bottom, or the first visible line, preferring top when content fits.
    pub fn position(self: Viewport, window: vaxis.Window) Position {
        if (self.top_line == 0) return .top;
        if (self.top_line >= self.line_count -| self.contentSize(window).height) return .bottom;
        return .{ .line = self.top_line };
    }

    /// Measure replacement lines and move vertically while preserving horizontal scroll.
    pub fn replaceLines(
        self: *Viewport,
        window: vaxis.Window,
        display_lines: []const DisplayLine,
        target_position: Position,
    ) void {
        self.line_count = display_lines.len;
        self.content_width = measureContentWidth(window, display_lines);

        self.top_line = switch (target_position) {
            .top => 0,
            .bottom => @as(u32, @intCast(self.line_count)) -| self.contentSize(window).height,
            .line => |line_index| line_index,
        };
    }

    /// Pull both offsets inside the drawable range.
    fn clampPosition(self: *Viewport, window: vaxis.Window) void {
        const content_size = self.contentSize(window);
        self.top_line = @min(
            self.top_line,
            @as(u32, @intCast(self.line_count)) -| content_size.height,
        );
        // A Vaxis child cannot reach past maxInt(u16), even when content is wider.
        const maximum_left_column = std.math.maxInt(u16) - content_size.width;
        self.left_column = @min(
            self.left_column,
            @min(self.content_width -| content_size.width, maximum_left_column),
        );
    }

    /// Apply one keyboard scroll command without clamping the result.
    pub fn scrollWithKey(self: *Viewport, key: vaxis.Key, window: vaxis.Window) void {
        if (key.matches('j', .{}) or key.matches(vaxis.Key.down, .{})) {
            self.top_line +|= 1;
        } else if (key.matches('k', .{}) or key.matches(vaxis.Key.up, .{})) {
            self.top_line -|= 1;
        } else if (key.matches('l', .{}) or key.matches(vaxis.Key.right, .{})) {
            self.left_column +|= 1;
        } else if (key.matches('h', .{}) or key.matches(vaxis.Key.left, .{})) {
            self.left_column -|= 1;
        } else if (key.matches(vaxis.Key.down, .{ .shift = true }) or
            key.matches(vaxis.Key.page_down, .{}))
        {
            self.top_line +|= self.contentSize(window).height;
        } else if (key.matches(vaxis.Key.up, .{ .shift = true }) or
            key.matches(vaxis.Key.page_up, .{}))
        {
            self.top_line -|= self.contentSize(window).height;
        } else if (key.matches(vaxis.Key.right, .{ .shift = true })) {
            self.left_column +|= self.contentSize(window).width;
        } else if (key.matches(vaxis.Key.left, .{ .shift = true })) {
            self.left_column -|= self.contentSize(window).width;
        } else if (key.matches(vaxis.Key.escape, .{})) {
            // Model consumes Escape in history; live mode uses it to return to top.
            self.top_line = 0;
        }
    }

    /// Clamp offsets, draw visible lines, and overlay scrollbar thumbs.
    pub fn draw(self: *Viewport, window: vaxis.Window, display_lines: []const DisplayLine) void {
        std.debug.assert(display_lines.len == self.line_count);
        self.clampPosition(window);
        std.debug.assert(self.left_column <= std.math.maxInt(u16) -| window.width);
        const content_size = self.contentSize(window);

        // A wider child shifted left lets Vaxis clip horizontal scroll for us.
        const content_window = window.child(.{
            .x_off = -@as(i17, @intCast(self.left_column)),
            .width = content_size.width + @as(u16, @intCast(self.left_column)),
            .height = content_size.height,
        });

        for (0..content_size.height) |viewport_row| {
            const line_index = self.top_line + viewport_row;
            if (line_index >= display_lines.len) break;
            // The child extends off-screen, so wrapping would create false rows.
            _ = content_window.print(
                display_lines[line_index],
                .{ .row_offset = @intCast(viewport_row), .wrap = .none },
            );
        }

        if (content_size.width < window.width) {
            const vertical_thumb = scrollbarThumb(
                content_size.height,
                @intCast(self.line_count),
                self.top_line,
            );
            for (vertical_thumb.offset..vertical_thumb.offset + vertical_thumb.length) |thumb_row|
                window.writeCell(window.width -| 1, @intCast(thumb_row), vertical_thumb_cell);
        }
        if (content_size.height < window.height) {
            const horizontal_thumb = scrollbarThumb(
                content_size.width,
                self.content_width,
                self.left_column,
            );
            for (horizontal_thumb.offset..horizontal_thumb.offset + horizontal_thumb.length) |thumb_column|
                window.writeCell(
                    @intCast(thumb_column),
                    window.height -| 1,
                    horizontal_thumb_cell,
                );
        }
    }
};

/// Measure the widest line in terminal cells.
fn measureContentWidth(window: vaxis.Window, display_lines: []const DisplayLine) u32 {
    var content_width: u32 = 0;
    for (display_lines) |display_line| {
        var line_width: u32 = 0;
        for (display_line) |segment| line_width += displayWidth(window, segment.text);
        content_width = @max(content_width, line_width);
    }
    return content_width;
}

/// Measure printable ASCII directly and leave all other text to Vaxis.
fn displayWidth(window: vaxis.Window, text: []const u8) u32 {
    for (text) |byte| if (!std.ascii.isPrint(byte)) return window.gwidth(text);
    return @intCast(text.len);
}

const ScrollbarThumb = struct { offset: u16, length: u16 };

/// Scale an overflowing axis into a thumb that lands exactly at both ends.
fn scrollbarThumb(
    track_length: u16,
    content_length: u32,
    content_offset: u32,
) ScrollbarThumb {
    const thumb_length: u16 = @min(
        track_length,
        @max(1, @as(u16, @intCast(@as(u32, track_length) * track_length / content_length))),
    );
    const thumb_travel = track_length -| thumb_length;
    const scroll_range = content_length - track_length;
    return .{
        .offset = @intCast(@as(u32, thumb_travel) * content_offset / scroll_range),
        .length = thumb_length,
    };
}

test "positions and clamps stay inside the content" {
    const line_segments = [_]vaxis.Segment{.{ .text = "x" }};
    const display_lines = [_]DisplayLine{&line_segments} ** 30;

    var view = try vaxis.widgets.View.init(std.testing.allocator, .{ .width = 10, .height = 8 });
    defer view.deinit();
    const window = view.window();

    var viewport: Viewport = .{};
    viewport.replaceLines(window, &display_lines, .top);
    try std.testing.expectEqual(@as(u32, 0), viewport.top_line);

    viewport.top_line = 1;
    try std.testing.expectEqualDeep(
        @as(Viewport.Position, .{ .line = 1 }),
        viewport.position(window),
    );
    viewport.replaceLines(window, &display_lines, .{ .line = 2 });
    try std.testing.expectEqual(@as(u32, 2), viewport.top_line);

    viewport.replaceLines(window, &display_lines, .bottom);
    try std.testing.expectEqual(@as(u32, 22), viewport.top_line);
    try std.testing.expectEqualDeep(
        @as(Viewport.Position, .bottom),
        viewport.position(window),
    );

    viewport.top_line = 100;
    viewport.clampPosition(window);
    try std.testing.expectEqual(@as(u32, 22), viewport.top_line);

    viewport.left_column = 100;
    viewport.clampPosition(window);
    try std.testing.expectEqual(@as(u32, 0), viewport.left_column);

    var tall_view = try vaxis.widgets.View.init(std.testing.allocator, .{ .width = 10, .height = 40 });
    defer tall_view.deinit();
    viewport.clampPosition(tall_view.window());
    try std.testing.expectEqual(@as(u32, 0), viewport.top_line);
}

test "drawing clamps content wider than a Vaxis child" {
    var view = try vaxis.widgets.View.init(std.testing.allocator, .{ .width = 80, .height = 4 });
    defer view.deinit();
    const window = view.window();

    const wide_text = "x" ** 66_000;
    const line_segments = [_]vaxis.Segment{.{ .text = wide_text }};
    const display_lines = [_]DisplayLine{&line_segments};

    var viewport: Viewport = .{};
    viewport.replaceLines(window, &display_lines, .top);

    viewport.left_column = 90_000;
    viewport.draw(window, &display_lines);
    try std.testing.expect(viewport.left_column <= std.math.maxInt(u16) - window.width);
    try std.testing.expectEqual(@as(u8, 'x'), view.readCell(0, 0).?.char.grapheme[0]);
}

test "display width preserves Vaxis handling outside printable ASCII" {
    var view = try vaxis.widgets.View.init(std.testing.allocator, .{ .width = 10, .height = 2 });
    defer view.deinit();
    const window = view.window();

    try std.testing.expectEqual(@as(u32, 5), displayWidth(window, "plain"));
    try std.testing.expectEqual(@as(u32, window.gwidth("漢")), displayWidth(window, "漢"));
    try std.testing.expectEqual(@as(u32, window.gwidth("\x1b")), displayWidth(window, "\x1b"));
}
