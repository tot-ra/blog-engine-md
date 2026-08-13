# Lessons learned

- When inserting a new script into an existing template block, inspect the surrounding closing tags first; line-based insertion can accidentally nest `<script>` elements while still passing `git diff --check`.
- If an external reviewer times out after the full deadline, continue with a different available reviewer and local generated-output checks instead of retrying the same agent.
- When a chained `commit && push` exits 128 without output, inspect `HEAD`, branch divergence, and `.git/index.lock` before retrying; the commit may already have succeeded and only the push failed.
- `sectionChildrenWithMediaHTML` builds a poster grid for any child preview image, not only embeds. Keep the play cue (`section-video-preview-play`) only when the thumb came from YouTube/Vimeo; article photo indexes (e.g. events) should stay image cards without a play button.
