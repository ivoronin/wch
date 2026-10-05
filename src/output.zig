//! Owns normalized command output and its display-ready terminal segments.

const std = @import("std");
const vaxis = @import("vaxis");
const diff = @import("diff.zig");
const ansi = @import("ansi.zig");

const added_style: vaxis.Style = .{ .fg = .{ .index = 2 } };
const DisplayLine = []const vaxis.Segment;

test "padding and punctuation edits preserve unchanged text" {
    var output: Output = .init(std.testing.allocator);
    defer output.deinit();

    const Case = struct { before: []const u8, after: []const u8, highlighted: []const u8 };
    for ([_]Case{
        .{ .before = "node (zone)  Ready", .after = "node (zone)   Ready", .highlighted = "   " },
        .{ .before = "node (zone)   Ready", .after = "node (zone)  Ready", .highlighted = "  " },
        .{ .before = "node (zone)] Ready", .after = "node (zone),] Ready", .highlighted = "," },
        .{ .before = "node (zone)) Ready", .after = "node (zone))) Ready", .highlighted = ")" },
        .{ .before = "node (漢字)  old", .after = "node (漢字)   new", .highlighted = "   new" },
    }) |case| {
        _ = try output.replaceText(.unicode, case.before, case.after, null);
        var highlighted: std.ArrayList(u8) = .empty;
        defer highlighted.deinit(std.testing.allocator);
        for (output.display_lines[0]) |segment| {
            if (segment.style.fg.eql(added_style.fg))
                try highlighted.appendSlice(std.testing.allocator, segment.text);
        }
        try std.testing.expectEqualStrings(case.highlighted, highlighted.items);
    }
}

pub const Output = struct {
    normalized_output: ?[]const u8 = null,
    display_lines: []const DisplayLine = &.{},
    storage: std.heap.ArenaAllocator,

    /// Create empty output storage.
    pub fn init(allocator: std.mem.Allocator) Output {
        return .{ .storage = .init(allocator) };
    }

    /// Release all output storage.
    pub fn deinit(self: *Output) void {
        self.storage.deinit();
        self.* = undefined;
    }

    /// Atomically replace output and map an optional line from the displayed output.
    pub fn replaceText(
        self: *Output,
        width_method: vaxis.gwidth.Method,
        previous_output: ?[]const u8,
        current_output: []const u8,
        visible_line: ?u32,
    ) std.mem.Allocator.Error!?u32 {
        const backing_allocator = self.storage.child_allocator;

        var next_storage: std.heap.ArenaAllocator = .init(backing_allocator);
        errdefer next_storage.deinit();

        var line_scratch: std.heap.ArenaAllocator = .init(backing_allocator);
        defer line_scratch.deinit();
        var word_scratch: std.heap.ArenaAllocator = .init(backing_allocator);
        defer word_scratch.deinit();

        const retained_allocator = next_storage.allocator();
        const line_allocator = line_scratch.allocator();

        const normalized_previous_output = if (previous_output) |raw_previous_output|
            (try ansi.normalize(line_allocator, width_method, raw_previous_output)).text
        else
            null;
        const current = try ansi.normalize(
            retained_allocator,
            width_method,
            current_output,
        );
        const line_changes = try diff.compareLines(
            line_allocator,
            normalized_previous_output,
            current.text,
        );

        var mapped_visible_line: ?u32 = null;
        if (visible_line) |line_index| if (self.normalized_output) |displayed_output| {
            const previous_is_displayed = if (normalized_previous_output) |normalized_previous|
                std.mem.eql(u8, displayed_output, normalized_previous)
            else
                false;
            const mapping_changes = if (previous_is_displayed)
                line_changes
            else
                try diff.compareLines(
                    line_allocator,
                    displayed_output,
                    current.text,
                );
            mapped_visible_line = mapping_changes.mapBeforeLine(line_index);
        };

        const rendered_lines = try renderLines(
            retained_allocator,
            &word_scratch,
            line_changes,
            current,
        );

        self.storage.deinit();
        self.* = .{
            .normalized_output = current.text,
            .display_lines = rendered_lines,
            .storage = next_storage,
        };
        return mapped_visible_line;
    }
};

/// Render the new side of line changes with additions highlighted.
fn renderLines(
    retained_allocator: std.mem.Allocator,
    word_scratch: *std.heap.ArenaAllocator,
    line_changes: diff.Changes,
    current: ansi.Text,
) ![]const DisplayLine {
    var rendered_lines: std.ArrayList(DisplayLine) = .empty;
    var after_line_index: usize = 0;
    var source: SegmentSource = .{ .text = current.text, .styles = current.styles };

    for (line_changes.edits) |edit| {
        const line_count: usize = @intCast(edit.range.end - edit.range.start);
        switch (edit.kind) {
            .equal => {
                for (
                    line_changes.before[edit.range.start..edit.range.end],
                    line_changes.after[after_line_index..][0..line_count],
                ) |before_line, after_line| {
                    var segments: std.ArrayList(vaxis.Segment) = .empty;
                    if (std.mem.eql(u8, before_line, after_line))
                        try source.appendSegments(retained_allocator, &segments, after_line, false)
                    else
                        try renderWordChanges(
                            retained_allocator,
                            word_scratch,
                            before_line,
                            after_line,
                            &segments,
                            &source,
                        );
                    try rendered_lines.append(retained_allocator, segments.items);
                }
                after_line_index += line_count;
            },
            .delete => {},
            .insert => {
                for (line_changes.after[after_line_index..][0..line_count]) |inserted_line| {
                    var segments: std.ArrayList(vaxis.Segment) = .empty;
                    try source.appendSegments(retained_allocator, &segments, inserted_line, true);
                    try rendered_lines.append(retained_allocator, segments.items);
                }
                after_line_index += line_count;
            },
        }
    }

    return rendered_lines.items;
}

/// Render one matched line with inserted words highlighted.
fn renderWordChanges(
    retained_allocator: std.mem.Allocator,
    word_scratch: *std.heap.ArenaAllocator,
    before_line: []const u8,
    after_line: []const u8,
    rendered_segments: *std.ArrayList(vaxis.Segment),
    source: *SegmentSource,
) !void {
    _ = word_scratch.reset(.retain_capacity);
    const word_changes = try diff.compareWords(
        word_scratch.allocator(),
        before_line,
        after_line,
    );

    var after_part_index: usize = 0;

    for (word_changes.edits) |edit| {
        const part_count: usize = @intCast(edit.range.end - edit.range.start);
        switch (edit.kind) {
            .delete => {},
            .equal, .insert => {
                const after_parts = word_changes.after[after_part_index..][0..part_count];
                try source.appendSegments(retained_allocator, rendered_segments, partSpan(after_parts), edit.kind == .insert);
                after_part_index += part_count;
            },
        }
    }
    std.debug.assert(after_part_index == word_changes.after.len);
}

const SegmentSource = struct {
    text: []const u8,
    styles: []const ansi.Span,
    style_index: usize = 0,

    /// Visit text ranges in order, preserving command styles except the added foreground.
    fn appendSegments(
        self: *SegmentSource,
        allocator: std.mem.Allocator,
        segments: *std.ArrayList(vaxis.Segment),
        text: []const u8,
        added: bool,
    ) !void {
        if (text.len == 0) return;
        var start = @intFromPtr(text.ptr) - @intFromPtr(self.text.ptr);
        const end = start + text.len;
        while (start < end) {
            while (self.style_index + 1 < self.styles.len and self.styles[self.style_index + 1].start <= start)
                self.style_index += 1;
            const segment_end = if (self.style_index + 1 < self.styles.len)
                @min(end, self.styles[self.style_index + 1].start)
            else
                end;
            var style = self.styles[self.style_index].style;
            if (added) style.fg = added_style.fg;
            try segments.append(allocator, .{ .text = self.text[start..segment_end], .style = style });
            start = segment_end;
        }
    }
};

/// Return the text covered by adjacent parts without copying it.
fn partSpan(parts: []const []const u8) []const u8 {
    std.debug.assert(parts.len > 0);

    const first_part = parts[0];
    const last_part = parts[parts.len - 1];
    const span_length = @intFromPtr(last_part.ptr) + last_part.len - @intFromPtr(first_part.ptr);
    return first_part.ptr[0..span_length];
}

test "tabs follow terminal stops" {
    var output: Output = .init(std.testing.allocator);
    defer output.deinit();

    _ = try output.replaceText(.unicode, null, "漢\tx\nabcdefgh\ty", null);
    try std.testing.expectEqualStrings("漢      x\nabcdefgh        y", output.normalized_output.?);
}

test "maps owned output after the caller changes" {
    var output: Output = .init(std.testing.allocator);
    defer output.deinit();

    var initial_output = [_]u8{ 'a', '\n', 'b', '\n', 'c', '\n', 'd' };
    _ = try output.replaceText(.unicode, null, &initial_output, null);
    @memset(&initial_output, 'x');

    try std.testing.expectEqual(
        @as(?u32, 2),
        try output.replaceText(
            .unicode,
            "id old\na\nb\nc\nd",
            "id new\na\nb\nc\nd",
            1,
        ),
    );

    const normalized_output = output.normalized_output.?;
    const output_start = @intFromPtr(normalized_output.ptr);
    const output_end = output_start + normalized_output.len;
    for (output.display_lines) |display_line| for (display_line) |segment| {
        const segment_start = @intFromPtr(segment.text.ptr);
        try std.testing.expect(segment_start >= output_start);
        try std.testing.expect(segment_start + segment.text.len <= output_end);
    };
}

test "color-only changes refresh styles without diff highlighting" {
    var output: Output = .init(std.testing.allocator);
    defer output.deinit();

    const before = "\x1b[32mRunning\nstill\x1b[m plain";
    for ([_][]const u8{
        "",
        "\x1b[1:99m",
        "\x1b[1;38:5:196:7m",
        "\x1b[38:2:1:2:3:4:5m",
        "\x1b[4:1:99m",
    }) |malformed| {
        const after = try std.fmt.allocPrint(
            std.testing.allocator,
            "\x1b[2K\x1b[1;38;2;12;34;56mRunning\n\x1b[38;5;196mstill\x1b[m{s} plain",
            .{malformed},
        );
        defer std.testing.allocator.free(after);
        _ = try output.replaceText(.unicode, null, before, null);
        try std.testing.expectEqual(@as(?u32, 1), try output.replaceText(.unicode, before, after, 1));

        try std.testing.expectEqualStrings("Running\nstill plain", output.normalized_output.?);
        try std.testing.expectEqualDeep(&[_]vaxis.Segment{.{
            .text = "Running",
            .style = .{ .bold = true, .fg = .{ .rgb = .{ 12, 34, 56 } } },
        }}, output.display_lines[0]);
        try std.testing.expectEqualDeep(&[_]vaxis.Segment{
            .{ .text = "still", .style = .{ .bold = true, .fg = .{ .index = 196 } } },
            .{ .text = " plain" },
        }, output.display_lines[1]);
    }
}

test "text edits override only the foreground and preserve styles across tabs" {
    var output: Output = .init(std.testing.allocator);
    defer output.deinit();

    const before = "\x1b[31mapi\told\x1b[m\nstable";
    const after = "\x1b[1;31mapi\t\x1b[34mnew\x1b[m\nstable\n\x1b[38:2::12:34:56;4mextra\x1b[m";
    _ = try output.replaceText(.unicode, before, after, null);

    try std.testing.expectEqualStrings("api     new\nstable\nextra", output.normalized_output.?);
    try std.testing.expectEqualDeep(&[_]vaxis.Segment{
        .{ .text = "api     ", .style = .{ .bold = true, .fg = .{ .index = 1 } } },
        .{ .text = "new", .style = .{ .bold = true, .fg = .{ .index = 2 } } },
    }, output.display_lines[0]);
    try std.testing.expectEqualDeep(&[_]vaxis.Segment{.{ .text = "stable" }}, output.display_lines[1]);
    try std.testing.expectEqualDeep(&[_]vaxis.Segment{.{
        .text = "extra",
        .style = .{ .fg = .{ .index = 2 }, .ul_style = .single },
    }}, output.display_lines[2]);
}
