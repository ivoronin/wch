//! Normalizes captured output and records styles at final text offsets.

const std = @import("std");
const vaxis = @import("vaxis");

pub const Span = struct { start: usize, style: vaxis.Style };
pub const Text = struct { text: []const u8, styles: []const Span };

/// Expand tabs and consume escape sequences without executing terminal commands.
pub fn normalize(allocator: std.mem.Allocator, width_method: vaxis.gwidth.Method, raw: []const u8) !Text {
    var text: std.ArrayList(u8) = .empty;
    var styles: std.ArrayList(Span) = .empty;
    try styles.append(allocator, .{ .start = 0, .style = .{} });
    var style: vaxis.Style = .{};
    var index: usize = 0;
    var column: usize = 0;
    var measured_until: usize = 0;
    while (index < raw.len) {
        const byte = raw[index];
        if (byte == 0x1b) {
            index += 1;
            if (index == raw.len) break;
            const kind = raw[index];
            index += 1;
            switch (kind) {
                '[' => {
                    const start = index;
                    while (index < raw.len and raw[index] >= 0x20 and raw[index] <= 0x3f) : (index += 1) {}
                    if (index < raw.len and raw[index] >= 0x40 and raw[index] <= 0x7e) {
                        if (raw[index] == 'm') applySgr(&style, raw[start..index]);
                        index += 1;
                    }
                },
                ']', 'P', 'X', '^', '_' => {
                    while (index < raw.len) : (index += 1) {
                        if (kind == ']' and raw[index] == 0x07) {
                            index += 1;
                            break;
                        }
                        if (raw[index] == 0x1b and index + 1 < raw.len and raw[index + 1] == '\\') {
                            index += 2;
                            break;
                        }
                    }
                },
                0x20...0x2f => {
                    while (index < raw.len and raw[index] >= 0x20 and raw[index] <= 0x2f) : (index += 1) {}
                    if (index < raw.len and raw[index] >= 0x30 and raw[index] <= 0x7e) index += 1;
                },
                else => {
                    // Leave a following line control or escape for the next iteration.
                    if (kind < 0x20 or kind >= 0x7f) index -= 1;
                },
            }
            continue;
        }
        index += 1;
        if ((byte < 0x20 and byte != '\n' and byte != '\r' and byte != '\t') or byte == 0x7f) continue;
        const last = &styles.items[styles.items.len - 1];
        if (!last.style.eql(style)) {
            if (last.start == text.items.len)
                last.style = style
            else
                try styles.append(allocator, .{ .start = text.items.len, .style = style });
        }
        if (byte == '\t') {
            // Measure accumulated text so style changes do not split graphemes.
            column += vaxis.gwidth.gwidth(text.items[measured_until..], width_method);
            const spaces = 8 - column % 8;
            try text.appendNTimes(allocator, ' ', spaces);
            column += spaces;
            measured_until = text.items.len;
        } else {
            try text.append(allocator, byte);
            if (byte == '\n' or byte == '\r') {
                column = 0;
                measured_until = text.items.len;
            }
        }
    }
    return .{ .text = text.items, .styles = styles.items };
}

const Parameters = std.mem.SplitIterator(u8, .scalar);

/// Apply a complete SGR sequence; malformed parameters leave the style unchanged.
fn applySgr(style: *vaxis.Style, sequence: []const u8) void {
    var next = style.*;
    var parameters = std.mem.splitScalar(u8, sequence, ';');
    while (parameters.next()) |parameter| {
        var sub = std.mem.splitScalar(u8, parameter, ':');
        const code_text = sub.next().?;
        const code = if (code_text.len == 0) 0 else std.fmt.parseInt(u16, code_text, 10) catch return;
        switch (code) {
            0 => next = .{},
            1 => next.bold = true,
            2 => next.dim = true,
            3 => next.italic = true,
            4 => {
                const kind = if (sub.next()) |value| std.fmt.parseInt(u8, value, 10) catch return else 1;
                next.ul_style = std.enums.fromInt(vaxis.Style.Underline, kind) orelse return;
            },
            5, 6 => next.blink = true,
            7 => next.reverse = true,
            8 => next.invisible = true,
            9 => next.strikethrough = true,
            21 => next.ul_style = .double,
            22 => {
                next.bold = false;
                next.dim = false;
            },
            23 => next.italic = false,
            24 => next.ul_style = .off,
            25 => next.blink = false,
            27 => next.reverse = false,
            28 => next.invisible = false,
            29 => next.strikethrough = false,
            30...37 => next.fg = .{ .index = @intCast(code - 30) },
            40...47 => next.bg = .{ .index = @intCast(code - 40) },
            90...97 => next.fg = .{ .index = @intCast(code - 90 + 8) },
            100...107 => next.bg = .{ .index = @intCast(code - 100 + 8) },
            38, 48, 58 => {
                const colon = std.mem.findScalar(u8, parameter, ':') != null;
                const color = readColor(if (colon) &sub else &parameters, colon) orelse return;
                switch (code) {
                    38 => next.fg = color,
                    48 => next.bg = color,
                    58 => next.ul = color,
                    else => unreachable,
                }
            },
            39 => next.fg = .default,
            49 => next.bg = .default,
            59 => next.ul = .default,
            else => {},
        }
        if (sub.next() != null) return;
    }
    style.* = next;
}

fn readColor(parameters: *Parameters, colon: bool) ?vaxis.Color {
    const kind = parameters.next() orelse return null;
    if (std.mem.eql(u8, kind, "5"))
        return .{ .index = std.fmt.parseInt(u8, parameters.next() orelse return null, 10) catch return null };
    if (!std.mem.eql(u8, kind, "2")) return null;
    // Colon RGB may include an empty or zero color-space field before RGB.
    if (colon) switch (std.mem.count(u8, parameters.rest(), ":")) {
        2 => {},
        3 => {
            const space = parameters.next().?;
            if (space.len != 0 and !std.mem.eql(u8, space, "0")) return null;
        },
        else => return null,
    };
    var rgb: [3]u8 = undefined;
    for (&rgb) |*channel|
        channel.* = std.fmt.parseInt(u8, parameters.next() orelse return null, 10) catch return null;
    return .{ .rgb = rgb };
}
