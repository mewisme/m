---
name: md-toc
description: Inspect long Markdown files, generate or validate a table of contents, and navigate directly to relevant headings without loading the entire document. Use for implementation plans, agent guidance, research notes, and large compatibility documents.
---

# Markdown table of contents

Extract ATX headings outside fenced code blocks. Preserve heading text and order. Generate GitHub-compatible anchors and report duplicate anchors.

For a long file:

1. list headings with line numbers
2. identify the required section
3. read only that range
4. update the TOC when headings change

Do not include the document title in an in-page TOC unless the repository convention explicitly requires it.
