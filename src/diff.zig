//! Computes terminal-independent line and word changes with Dizzy.

const std = @import("std");
const dizzy = @import("dizzy");

const word_delimiters = delimiters: {
    var delimiter_set = std.StaticBitSet(256).initEmpty();
    for (" \t\r\n.,:;/|()[]") |delimiter| delimiter_set.set(delimiter);
    break :delimiters delimiter_set;
};

/// Part lists and their arena-backed Dizzy edits.
/// Delete ranges index `before`, insert ranges index `after`, and equal ranges pair both.
/// The slices borrow input text and live in the caller's arena.
pub const Changes = struct {
    before: []const []const u8,
    after: []const []const u8,
    edits: []const dizzy.Edit,

    /// Map a before line to its after position or deletion point.
    pub fn mapBeforeLine(self: Changes, before_line: u32) u32 {
        var after_line_index: u32 = 0;

        for (self.edits) |edit| {
            switch (edit.kind) {
                .insert => after_line_index = edit.range.end,
                .delete => if (before_line < edit.range.end) return after_line_index,
                .equal => {
                    if (before_line < edit.range.end)
                        return after_line_index + before_line - edit.range.start;
                    after_line_index += edit.range.end - edit.range.start;
                },
            }
        }

        return after_line_index;
    }
};

/// Diff visible lines, treating the first output as unchanged.
pub fn compareLines(
    arena: std.mem.Allocator,
    before_output: ?[]const u8,
    after_output: []const u8,
) !Changes {
    const after_lines = try splitLines(arena, after_output);
    const previous_output = before_output orelse return unchangedLines(arena, after_lines);
    if (std.mem.eql(u8, previous_output, after_output))
        return unchangedLines(arena, after_lines);
    const before_lines = try splitLines(arena, previous_output);

    const before_columns_by_line = try splitColumns(arena, before_lines);
    const after_columns_by_line = try splitColumns(arena, after_lines);
    const stable_column = try chooseStableColumn(
        arena,
        before_columns_by_line,
        after_columns_by_line,
    );
    const before_keys = try selectKeys(
        arena,
        before_lines,
        before_columns_by_line,
        stable_column,
    );
    const after_keys = try selectKeys(
        arena,
        after_lines,
        after_columns_by_line,
        stable_column,
    );

    return compareParts(arena, before_lines, after_lines, before_keys, after_keys);
}

/// Diff words while keeping delimiter runs as parts.
pub fn compareWords(
    arena: std.mem.Allocator,
    before_line: []const u8,
    after_line: []const u8,
) !Changes {
    const before_parts = try splitWords(arena, before_line);
    const after_parts = try splitWords(arena, after_line);
    return compareParts(arena, before_parts, after_parts, before_parts, after_parts);
}

/// Represent the first output as unchanged lines.
fn unchangedLines(arena: std.mem.Allocator, lines: []const []const u8) !Changes {
    const edits = if (lines.len == 0)
        &.{}
    else
        try arena.dupe(dizzy.Edit, &.{.{
            .kind = .equal,
            .range = .{ .start = 0, .end = @intCast(lines.len) },
        }});
    return .{ .before = lines, .after = lines, .edits = edits };
}

/// Split LF-delimited text and omit its final empty line and trailing carriage returns.
fn splitLines(arena: std.mem.Allocator, output: []const u8) ![]const []const u8 {
    if (output.len == 0) return &.{};

    var lines: std.ArrayList([]const u8) = .empty;
    const text = if (output[output.len - 1] == '\n') output[0 .. output.len - 1] else output;
    var line_iterator = std.mem.splitScalar(u8, text, '\n');
    while (line_iterator.next()) |line|
        try lines.append(arena, std.mem.trimEnd(u8, line, "\r"));

    return lines.items;
}

/// Split one line into alternating delimiter and non-delimiter runs.
fn splitWords(arena: std.mem.Allocator, line: []const u8) ![]const []const u8 {
    var parts: std.ArrayList([]const u8) = .empty;

    var part_start: usize = 0;
    for (line, 0..) |byte, byte_index| {
        if (word_delimiters.isSet(byte) == word_delimiters.isSet(line[part_start])) continue;
        try parts.append(arena, line[part_start..byte_index]);
        part_start = byte_index;
    }
    if (part_start < line.len) try parts.append(arena, line[part_start..]);

    return parts.items;
}

/// Split each line into whitespace-delimited columns.
fn splitColumns(
    arena: std.mem.Allocator,
    lines: []const []const u8,
) ![]const []const []const u8 {
    var columns_by_line: std.ArrayList([]const []const u8) = .empty;

    for (lines) |line| {
        var line_columns: std.ArrayList([]const u8) = .empty;
        var column_iterator = std.mem.tokenizeAny(u8, line, " \t\r\n");
        while (column_iterator.next()) |column| try line_columns.append(arena, column);
        try columns_by_line.append(arena, line_columns.items);
    }

    return columns_by_line.items;
}

const KeyOccurrences = struct {
    before_count: usize = 0,
    after_count: usize = 0,
};

/// Pick the column with the most values unique on both sides.
fn chooseStableColumn(
    arena: std.mem.Allocator,
    before_columns_by_line: []const []const []const u8,
    after_columns_by_line: []const []const []const u8,
) !?usize {
    var maximum_column_count: usize = 0;
    for (before_columns_by_line) |line_columns|
        maximum_column_count = @max(maximum_column_count, line_columns.len);
    for (after_columns_by_line) |line_columns|
        maximum_column_count = @max(maximum_column_count, line_columns.len);

    var occurrences_by_key = std.StringHashMap(KeyOccurrences).init(arena);
    defer occurrences_by_key.deinit();

    const maximum_unique_key_count = @min(
        before_columns_by_line.len,
        after_columns_by_line.len,
    );
    var best_column: ?usize = null;
    var best_unique_key_count: usize = 0;

    for (0..maximum_column_count) |column_index| {
        occurrences_by_key.clearRetainingCapacity();

        for (before_columns_by_line) |line_columns| {
            if (column_index >= line_columns.len) continue;
            const occurrence_entry = try occurrences_by_key.getOrPut(line_columns[column_index]);
            if (!occurrence_entry.found_existing) occurrence_entry.value_ptr.* = .{};
            occurrence_entry.value_ptr.before_count += 1;
        }

        for (after_columns_by_line) |line_columns| {
            if (column_index >= line_columns.len) continue;
            const occurrence_entry = try occurrences_by_key.getOrPut(line_columns[column_index]);
            if (!occurrence_entry.found_existing) occurrence_entry.value_ptr.* = .{};
            occurrence_entry.value_ptr.after_count += 1;
        }

        var unique_key_count: usize = 0;
        var occurrence_iterator = occurrences_by_key.valueIterator();
        while (occurrence_iterator.next()) |occurrences| {
            if (occurrences.before_count == 1 and occurrences.after_count == 1)
                unique_key_count += 1;
        }

        if (unique_key_count <= best_unique_key_count) continue;

        best_unique_key_count = unique_key_count;
        best_column = column_index;
        if (unique_key_count == maximum_unique_key_count) return column_index;
    }

    return best_column;
}

/// Read keys from one column, falling back to the complete line.
fn selectKeys(
    arena: std.mem.Allocator,
    lines: []const []const u8,
    columns_by_line: []const []const []const u8,
    stable_column: ?usize,
) ![]const []const u8 {
    const column_index = stable_column orelse return lines;

    const comparison_keys = try arena.alloc([]const u8, lines.len);
    for (lines, columns_by_line, comparison_keys) |line, line_columns, *comparison_key|
        comparison_key.* = if (column_index < line_columns.len)
            line_columns[column_index]
        else
            line;

    return comparison_keys;
}

const PartDiffer = dizzy.SliceDiffer([]const u8, std.hash_map.StringContext);

/// Run Dizzy on comparison keys and return edits over the original parts.
fn compareParts(
    arena: std.mem.Allocator,
    before_parts: []const []const u8,
    after_parts: []const []const u8,
    before_keys: []const []const u8,
    after_keys: []const []const u8,
) !Changes {
    std.debug.assert(before_parts.len == before_keys.len);
    std.debug.assert(after_parts.len == after_keys.len);

    const scratch = try arena.alloc(u32, 4 * (before_parts.len + after_parts.len) + 2);
    var edits: std.ArrayList(dizzy.Edit) = .empty;
    try PartDiffer.diff(arena, &edits, before_keys, after_keys, scratch);

    // Keep deletions before insertions so replacements read old then new.
    for (0..edits.items.len -| 1) |edit_index|
        if (edits.items[edit_index].kind == .insert and
            edits.items[edit_index + 1].kind == .delete)
            std.mem.swap(
                dizzy.Edit,
                &edits.items[edit_index],
                &edits.items[edit_index + 1],
            );

    return .{ .before = before_parts, .after = after_parts, .edits = edits.items };
}

test "pairs changed rows by their stable key" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    const line_changes = try compareLines(
        arena.allocator(),
        "api-1 Running 0 5m\r\napi-2 Running 0 5m\r\n",
        "api-1 Running 0 6m\napi-2 Pending 0 6m",
    );
    try std.testing.expectEqualDeep(&[_]dizzy.Edit{.{
        .kind = .equal,
        .range = .{ .start = 0, .end = 2 },
    }}, line_changes.edits);
}

test "leaves unrelated rows unpaired" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    const line_changes = try compareLines(
        arena.allocator(),
        "api-1 Running 0 5m\nweb-1 Running 0 5m",
        "api-1 Running 0 6m\ndb-9 Pending 3 1s",
    );
    try std.testing.expectEqualDeep(&[_]dizzy.Edit{
        .{ .kind = .equal, .range = .{ .start = 0, .end = 1 } },
        .{ .kind = .delete, .range = .{ .start = 1, .end = 2 } },
        .{ .kind = .insert, .range = .{ .start = 1, .end = 2 } },
    }, line_changes.edits);
}

test "falls back to whole lines without a key column" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    var deleted_lines: usize = 0;
    var inserted_lines: usize = 0;
    const line_changes = try compareLines(
        arena.allocator(),
        "group a\ngroup b",
        "group c\ngroup d",
    );
    for (line_changes.edits) |edit| switch (edit.kind) {
        .delete => deleted_lines += edit.range.end - edit.range.start,
        .insert => inserted_lines += edit.range.end - edit.range.start,
        else => return error.UnexpectedEdit,
    };

    try std.testing.expectEqual(@as(usize, 2), deleted_lines);
    try std.testing.expectEqual(@as(usize, 2), inserted_lines);
}

test "diffs words and keeps delimiter runs" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    const word_changes = try compareWords(arena.allocator(), "cpu: 12", "cpu: 13");
    try std.testing.expectEqualDeep(&[_]dizzy.Edit{
        .{ .kind = .equal, .range = .{ .start = 0, .end = 2 } },
        .{ .kind = .delete, .range = .{ .start = 2, .end = 3 } },
        .{ .kind = .insert, .range = .{ .start = 2, .end = 3 } },
    }, word_changes.edits);
}
