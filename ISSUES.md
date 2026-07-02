# Notes Management System - GitHub Issue Processing Workflow

This document serves as the entry point and index for Seraphine's issue-processing workflows. It outlines the general rules and lists the specific workflow files for each stage in the issue lifecycle.

**Note:** Always use **native GitHub sub-issues** when defining parent-child issue relationships to ensure proper tracking within GitHub.

---

## 🚫 Critical General Rules
1. **Scope Adherence**: The agent should only address the labeled issue, and it must stop once the issue is unlabeled.
2. **Termination Rule**: **The agent should not proceed to the next label.** Once you have removed a label from the bug (or a PR is merged), you should stop execution immediately. Do not trigger or begin processing the next stage or label in the same run.
3. **Issue Assignment**: **Whenever a new issue (or sub-issue) is created, it MUST be assigned to `brotherlogic-automation`.**

---

## 🏷️ Workflow Stages & Labels

When an issue is labeled, refer to the corresponding workflow document under `.agent/workflows/` for detailed step-by-step instructions:

1. **Deep Research**
   - **Label**: `rose-needs-deep-research`
   - **Workflow Guideline**: [rose-needs-deep-research.md](file:///workspaces/rose/.agent/workflows/rose-needs-deep-research.md)

2. **Requirements gathering**
   - **Label**: `rose-needs-requirements` (or variant `rose-need-requirements`)
   - **Workflow Guideline**: [rose-needs-requirements.md](file:///workspaces/rose/.agent/workflows/rose-needs-requirements.md)

3. **Technical implementation plan formulation**
   - **Label**: `rose-needs-implementation-plan`
   - **Workflow Guideline**: [rose-needs-implementation-plan.md](file:///workspaces/rose/.agent/workflows/rose-needs-implementation-plan.md)

4. **Issue breakdown**
   - **Label**: `rose-break-down-issue`
   - **Workflow Guideline**: [rose-break-down-issue.md](file:///workspaces/rose/.agent/workflows/rose-break-down-issue.md)

5. **Component implementation**
   - **Label**: `rose-ready-to-implement`
   - **Workflow Guideline**: [rose-ready-to-implement.md](file:///workspaces/rose/.agent/workflows/rose-ready-to-implement.md)

6. **Bug triage and resolution**
   - **Label**: `rose-bug`
   - **Workflow Guideline**: [rose-bug.md](file:///workspaces/rose/.agent/workflows/rose-bug.md)

---

## 🛠️ Summary of Expected Label State Transitions

| Phase | Parent Issue Label(s) | Sub-Issue Title & Label(s) |
| :--- | :--- | :--- |
| **Deep Research** | `rose-needs-deep-research` | *None (Not yet created)* |
| **Deep Research Complete** | `rose-needs-deep-research` (Removed) | Labeled with `rose-needs-requirements` to initiate requirements gathering |
| **Requirements Gathering** | `rose-needs-requirements` | *None (Not yet created)* |
| **Requirements Approved** | *(Label Removed)* | `[Implementation Plan] <Title>` labeled with `rose-needs-implementation-plan` |
| **Implementation Plan Drafting** | *None* | `[Implementation Plan] <Title>` labeled with `rose-needs-implementation-plan` |
| **Implementation Plan Approved** | *None* | **Implementation Plan:** Label removed (remains Open).<br>**Breakdown Sub-Issue:** `[Breakdown] <Title>` labeled with `rose-break-down-issue` |
| **Issue Breakdown** | *None* | **Breakdown Issue:** `rose-break-down-issue` removed (remains Open).<br>**Child Sub-Issues:** `[Sub-Issue] <Action>` labeled with `rose-ready-to-implement` |
| **Implementation** | *None* | **Breakdown Issue:** Closed when all child sub-issues are closed (cascading to close Implementation Plan and Parent issues).<br>**Child Sub-Issues:** Labeled with `rose-ready-to-implement`. Closed programmatically via PR submission. |
| **Bug Triage (Simple)** | `rose-bug` | *None (Direct fix implemented and PR submitted)* |
| **Bug Triage (Complex/Failed)** | `rose-bug` (Removed) | New issue labeled with `rose-needs-requirements` to initiate requirements gathering |
