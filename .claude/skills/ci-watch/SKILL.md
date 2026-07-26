---
name: ci-watch
description: Observe Mew pull-request CI, distinguish real failures from infrastructure problems, retrieve failing job details, and hand actionable evidence back to the implementation owner. Use after a verified push or when the user asks to monitor or diagnose CI.
---

# CI watch

Track the exact head SHA. Do not report a previous commit's green result as current.

On failure:

1. identify workflow, job, step, runner OS, and exact SHA
2. retrieve the relevant log range
3. determine whether the failure reproduces locally
4. classify code defect, test defect, platform defect, flaky infrastructure, or external outage
5. provide the smallest actionable diagnosis

Do not reflexively rerun a deterministic failure. Do not change code inside a watcher unless the dispatch explicitly includes CI repair ownership.
