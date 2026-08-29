//! Owns normalized command output and its display-ready terminal segments.

const std = @import("std");
const vaxis = @import("vaxis");
const diff = @import("diff.zig");

const added_style: vaxis.Style = .{ .fg = .{ .index = 2 } };
const tab_stop = 8;
const DisplayLine = []const vaxis.Segment;

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
            try expandTabs(line_allocator, width_method, raw_previous_output)
        else
            null;
        const normalized_current_output = try expandTabs(
            retained_allocator,
            width_method,
            current_output,
        );
        const line_changes = try diff.compareLines(
            line_allocator,
            normalized_previous_output,
            normalized_current_output,
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
                    normalized_current_output,
                );
            mapped_visible_line = mapping_changes.mapBeforeLine(line_index);
        };

        const rendered_lines = try renderLines(
            retained_allocator,
            &word_scratch,
            line_changes,
        );

        self.storage.deinit();
        self.* = .{
            .normalized_output = normalized_current_output,
            .display_lines = rendered_lines,
            .storage = next_storage,
        };
        return mapped_visible_line;
    }
};

/// Expand tabs to terminal stops using display-cell widths.
fn expandTabs(
    arena: std.mem.Allocator,
    width_method: vaxis.gwidth.Method,
    output_text: []const u8,
) ![]const u8 {
    if (std.mem.findScalar(u8, output_text, '\t') == null)
        return arena.dupe(u8, output_text);

    var expanded_output: std.ArrayList(u8) = .empty;
    var column: usize = 0;
    var remaining_output = output_text;

    while (std.mem.findAny(u8, remaining_output, "\t\n\r")) |control_index| {
        const preceding_text = remaining_output[0..control_index];
        try expanded_output.appendSlice(arena, preceding_text);

        if (remaining_output[control_index] == '\t') {
            column += vaxis.gwidth.gwidth(preceding_text, width_method);
            const space_count = tab_stop - column % tab_stop;
            try expanded_output.appendNTimes(arena, ' ', space_count);
            column += space_count;
        } else {
            try expanded_output.append(arena, remaining_output[control_index]);
            column = 0;
        }
        remaining_output = remaining_output[control_index + 1 ..];
    }

    try expanded_output.appendSlice(arena, remaining_output);
    return expanded_output.items;
}

/// Render the new side of line changes with additions highlighted.
fn renderLines(
    retained_allocator: std.mem.Allocator,
    word_scratch: *std.heap.ArenaAllocator,
    line_changes: diff.Changes,
) ![]const DisplayLine {
    var rendered_lines: std.ArrayList(DisplayLine) = .empty;
    var after_line_index: usize = 0;

    for (line_changes.edits) |edit| {
        const line_count: usize = @intCast(edit.range.end - edit.range.start);
        switch (edit.kind) {
            .equal => {
                for (
                    line_changes.before[edit.range.start..edit.range.end],
                    line_changes.after[after_line_index..][0..line_count],
                ) |before_line, after_line| {
                    const rendered_line = if (std.mem.eql(u8, before_line, after_line))
                        try retained_allocator.dupe(vaxis.Segment, &.{.{ .text = after_line }})
                    else
                        try renderWordChanges(
                            retained_allocator,
                            word_scratch,
                            before_line,
                            after_line,
                        );
                    try rendered_lines.append(retained_allocator, rendered_line);
                }
                after_line_index += line_count;
            },
            .delete => {},
            .insert => {
                for (line_changes.after[after_line_index..][0..line_count]) |inserted_line|
                    try rendered_lines.append(
                        retained_allocator,
                        try retained_allocator.dupe(vaxis.Segment, &.{.{
                            .text = inserted_line,
                            .style = added_style,
                        }}),
                    );
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
) !DisplayLine {
    _ = word_scratch.reset(.retain_capacity);
    const word_changes = try diff.compareWords(
        word_scratch.allocator(),
        before_line,
        after_line,
    );

    var rendered_segments: std.ArrayList(vaxis.Segment) = .empty;
    var after_part_index: usize = 0;

    for (word_changes.edits) |edit| {
        const part_count: usize = @intCast(edit.range.end - edit.range.start);
        switch (edit.kind) {
            .delete => {},
            .equal, .insert => {
                const after_parts = word_changes.after[after_part_index..][0..part_count];
                try rendered_segments.append(retained_allocator, .{
                    .text = partSpan(after_parts),
                    .style = if (edit.kind == .insert) added_style else .{},
                });
                after_part_index += part_count;
            },
        }
    }
    std.debug.assert(after_part_index == word_changes.after.len);

    return rendered_segments.items;
}

/// Return the text covered by adjacent parts without copying it.
fn partSpan(parts: []const []const u8) []const u8 {
    std.debug.assert(parts.len > 0);

    const first_part = parts[0];
    const last_part = parts[parts.len - 1];
    const span_length = @intFromPtr(last_part.ptr) + last_part.len - @intFromPtr(first_part.ptr);
    return first_part.ptr[0..span_length];
}

test "tabs follow terminal stops" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    const expanded = try expandTabs(arena.allocator(), .unicode, "漢\tx\nabcdefgh\ty");
    try std.testing.expectEqualStrings("漢      x\nabcdefgh        y", expanded);
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
