---
name: address-issue
description: Triage and address a Mew GitHub issue from acknowledgement through reproduction, root cause, fix, regression test, documentation, and a linked pull request. Use whenever work starts from an issue report or an issue needs a factual maintainer response.
---

# Address an issue

1. Determine whether the reporter is external. Post exactly `Investigating.` when beginning work on an external report.
2. Read the issue body, all comments, labels, linked PRs, and closure history.
3. Reproduce with the smallest fixture and record environment versions.
4. Confirm whether the behavior is a Mew defect, reference-tool behavior, unsupported input, or usage problem.
5. Fix the root cause inside approved scope.
6. Add a regression test and update docs for user-visible behavior.
7. Run the local verification loop.
8. Open a PR whose body includes `Closes #N` when it resolves the issue.
9. Keep follow-up comments factual and brief.

Do not converse with bots. Do not promise a release date.
