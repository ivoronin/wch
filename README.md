# wch

A command watcher that shows what changed without making you reread the rest.

<img width="1200" height="672" alt="wch-1200-10fps" src="https://github.com/user-attachments/assets/b03915ab-6a42-4f09-86a0-7e49523b78ab" />

This is a modern `watch` replacement. The main issue with `watch` is that it does character-level diffs, so any structural change in the output (a column width change, an added line) becomes a mess of highlighted cells.

This tool focuses on tabular output, such as output from `kubectl`. It distinguishes changed lines from inserted or replaced lines in the before and after outputs by detecting an "identity column" containing unique values (like `NAME` in the example above) and using it to pair the lines.

That allows it to ignore the noise (like ticking `AGE` or changing `STATUS`) and produce "structural" diffs that highlight only the parts of the output that changed, even in high-churn conditions.

```bash
brew install ivoronin/ivoronin/wch
```
