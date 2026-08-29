# Watch TUI

`wch` repeatedly runs one command and keeps recent output available in a terminal interface.

## Language

**Run**:
One execution attempt of the watched command, with its completion time and captured output.
_Avoid_: Sample, result

**Output**:
The normalized, display-ready form of one run.
_Avoid_: Picture, pane content

**History**:
The retained sequence of distinct runs, ordered from oldest to newest.
_Avoid_: Log, archive

**History cursor**:
A stable selection that can be resolved while its run remains in history.
_Avoid_: Anchor, index

**Live mode**:
The mode that follows the newest run.

**History mode**:
The mode that shows one selected run instead of following the newest run.

**Viewport**:
The scrollable visible region of output.
_Avoid_: Pane, picture

**Status bar**:
The bottom row that reports mode, run time, history position, and available keys.
_Avoid_: Dock, footer
