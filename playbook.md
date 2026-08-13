# Lessons learned
- Before delegating review, discover available agents instead of assuming a generic `code-reviewer` agent exists.
- When inserting a helper near a function boundary, read exact numbered lines first and insert before the function declaration, not after a line inside its opening guard.
- Registry discovery status can become stale before delegation; if an agent is reported inactive, do not retry it and complete review with local diff/build verification.

- Generated pages can repeat an article title in sidebar, timeline, and cards; integration assertions must locate the title inside `section-article-preview`, not from the first global occurrence.
- HTML escaping assertions should compare against the actual serialization layer: after `html.UnescapeString` followed by `template.HTMLEscapeString`, `&amp;` remains singly escaped, not `&amp;amp;`.
- When inserting a new script into an existing template block, inspect the surrounding closing tags first; line-based insertion can accidentally nest `<script>` elements while still passing `git diff --check`.
- If an external reviewer times out after the full deadline, continue with a different available reviewer and local generated-output checks instead of retrying the same agent.
- When a chained `commit && push` exits 128 without output, inspect `HEAD`, branch divergence, and `.git/index.lock` before retrying; the commit may already have succeeded and only the push failed.
- `sectionChildrenWithMediaHTML` builds a poster grid for any child preview image, not only embeds. Keep the play cue (`section-video-preview-play`) only when the thumb came from YouTube/Vimeo; article photo indexes (e.g. events) should stay image cards without a play button.
- Sidebar `enableCategories` is a `*bool`: omit or null keeps categories (historical default); set `false` for time-only sections. `defaultMode: time` alone still shows the Categories/Time switch whenever both panes exist.
- `showChildren` media grids inherit nav A-Z order. When children have frontmatter/inferred dates (events), sort newest-first in `sectionChildrenWithMediaHTML` via `sortSectionChildrenByDateDescIfDated`; leave undated docs trees in nav order.
