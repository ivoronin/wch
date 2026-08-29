//! Stores recent distinct runs and resolves stable cursors into display views.

const std = @import("std");
const Run = @import("run.zig").Run;

pub const History = struct {
    pub const Cursor = usize;

    /// Borrowed until History changes.
    pub const View = struct {
        cursor: Cursor,
        run: *const Run,
        previous_output: ?[]const u8,
        run_number: usize,
        run_count: usize,
    };

    allocator: std.mem.Allocator,
    runs: std.ArrayList(Run) = .empty,
    run_limit: usize,
    oldest_cursor: Cursor = 0,

    /// Create an empty history that retains at least one run.
    pub fn init(allocator: std.mem.Allocator, run_limit: usize) History {
        return .{ .allocator = allocator, .run_limit = @max(run_limit, 1) };
    }

    /// Free every retained run and the run list.
    pub fn deinit(self: *History) void {
        for (self.runs.items) |*run| run.deinit(self.allocator);
        self.runs.deinit(self.allocator);
        self.* = undefined;
    }

    /// Take ownership of a run and retain it when storage permits.
    pub fn append(self: *History, run: Run) void {
        var owned_run = run;
        if (self.runs.getLastOrNull()) |newest_run| {
            if (std.mem.eql(u8, owned_run.output, newest_run.output)) {
                owned_run.deinit(self.allocator);
                return;
            }
        }

        self.runs.append(self.allocator, owned_run) catch {
            owned_run.deinit(self.allocator);
            return;
        };
        owned_run = undefined;
        if (self.runs.items.len > self.run_limit) {
            var dropped_run = self.runs.orderedRemove(0);
            dropped_run.deinit(self.allocator);
            self.oldest_cursor += 1;
        }
    }

    /// Resolve null to the newest run and an expired cursor to the oldest.
    pub fn resolve(self: *const History, cursor: ?Cursor) ?View {
        const retained_runs = self.runs.items;
        if (retained_runs.len == 0) return null;

        const selected_run_index = if (cursor) |selected_cursor|
            self.runIndex(selected_cursor)
        else
            retained_runs.len - 1;
        const selected_run = &retained_runs[selected_run_index];
        return .{
            .cursor = self.oldest_cursor + selected_run_index,
            .run = selected_run,
            .previous_output = if (selected_run_index == 0)
                null
            else
                retained_runs[selected_run_index - 1].output,
            .run_number = selected_run_index + 1,
            .run_count = retained_runs.len,
        };
    }

    /// Return the previous cursor, clamped to the oldest retained run.
    pub fn previous(self: *const History, cursor: Cursor) Cursor {
        std.debug.assert(self.runs.items.len > 0);

        const current_run_index = self.runIndex(cursor);
        return self.oldest_cursor + (current_run_index -| 1);
    }

    /// Return the next cursor, clamped to the newest retained run.
    pub fn next(self: *const History, cursor: Cursor) Cursor {
        std.debug.assert(self.runs.items.len > 0);

        const current_run_index = self.runIndex(cursor);
        const next_run_index = @min(current_run_index + 1, self.runs.items.len - 1);
        return self.oldest_cursor + next_run_index;
    }

    /// Resolve an unretained cursor to the oldest run index.
    fn runIndex(self: *const History, cursor: Cursor) usize {
        if (cursor < self.oldest_cursor) return 0;
        const run_index = cursor - self.oldest_cursor;
        return if (run_index < self.runs.items.len) run_index else 0;
    }
};

/// Create a test run with owned output.
fn createTestRun(allocator: std.mem.Allocator, output: []const u8) !Run {
    return .{ .completed_at = .zero, .output = try allocator.dupe(u8, output) };
}

test "history resolves and moves through retained runs" {
    const allocator = std.testing.allocator;
    var history: History = .init(allocator, 2);
    defer history.deinit();

    history.append(try createTestRun(allocator, "a"));
    const expired_cursor = history.resolve(null).?.cursor;
    for ([_][]const u8{ "a", "b", "c", "d" }) |run_output|
        history.append(try createTestRun(allocator, run_output));

    const oldest_run_view = history.resolve(expired_cursor).?;
    const newest_run_view = history.resolve(null).?;

    try std.testing.expectEqualStrings("c", oldest_run_view.run.output);
    try std.testing.expect(oldest_run_view.previous_output == null);
    try std.testing.expectEqual(1, oldest_run_view.run_number);
    try std.testing.expectEqual(2, oldest_run_view.run_count);
    try std.testing.expectEqualStrings("d", newest_run_view.run.output);
    try std.testing.expectEqualStrings("c", newest_run_view.previous_output.?);
    try std.testing.expectEqual(2, newest_run_view.run_number);
    try std.testing.expectEqualSlices(
        History.Cursor,
        &.{
            oldest_run_view.cursor,
            newest_run_view.cursor,
            oldest_run_view.cursor,
            newest_run_view.cursor,
        },
        &.{
            history.previous(oldest_run_view.cursor),
            history.next(oldest_run_view.cursor),
            history.previous(newest_run_view.cursor),
            history.next(newest_run_view.cursor),
        },
    );
}

test "a run that cannot be stored is dropped" {
    var failing_allocator = std.testing.FailingAllocator.init(std.testing.allocator, .{ .fail_index = 1 });
    const allocator = failing_allocator.allocator();

    var history: History = .init(allocator, std.math.maxInt(usize));
    defer history.deinit();

    history.append(try createTestRun(allocator, "run"));
    try std.testing.expect(history.resolve(null) == null);
}
