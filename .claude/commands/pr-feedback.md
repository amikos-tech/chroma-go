Let's review the PR feedback since the last push.

## Step 1: Prerequisites & PR Detection

First, verify the environment and detect the current PR.

**Verify gh CLI is available and authenticated:**
```bash
command -v gh >/dev/null 2>&1
gh auth status >/dev/null 2>&1
```

**Get current branch and detect PR:**
```bash
CURRENT_BRANCH=$(git branch --show-current)
gh pr list --head "$CURRENT_BRANCH" --json number,title,state,url --limit 1
```

**Error handling:**
- If gh CLI not found: "Error: gh CLI not installed. Install from https://cli.github.com/"
- If not authenticated: "Error: Not authenticated with GitHub. Run 'gh auth login'"
- If no PR found: "Error: No PR found for branch '<branch>'. Create a PR first with 'gh pr create'."
- If PR is MERGED or CLOSED: "Warning: PR #N is <STATE>. Analyzing historical feedback."

## Step 2: Get Last Push Timestamp

Get the committer date of the last commit on the PR to determine the "last push" timestamp:

```bash
gh pr view <PR_NUMBER> --json commits --jq '.commits | last | .committedDate'
```

This represents when the last commit was pushed to the PR branch.

## Step 3: Fetch All Comments Since Last Push

Fetch three types of PR feedback, filtering to only include items created after the last push timestamp:

**1. Conversation comments** (PR discussion thread):
```bash
gh pr view <PR_NUMBER> --json comments --jq '.comments | [.[] | select(.createdAt > "<TIMESTAMP>") | {type: "conversation", author: .author.login, body: .body, createdAt: .createdAt, url: .url}]'
```

**2. Review summaries** (APPROVED, CHANGES_REQUESTED, COMMENTED):
```bash
gh pr view <PR_NUMBER> --json reviews --jq '.reviews | [.[] | select(.submittedAt > "<TIMESTAMP>") | select(.body | length > 0) | {type: "review", author: .author.login, state: .state, body: .body, createdAt: .submittedAt}]'
```

**3. Inline review comments** (code-level comments on specific lines):
```bash
gh api repos/{owner}/{repo}/pulls/<PR_NUMBER>/comments --jq '[.[] | select(.created_at > "<TIMESTAMP>") | {type: "inline", author: .user.login, body: .body, createdAt: .created_at, path: .path, line: .line, url: .html_url}]'
```

Merge all three arrays and sort by `createdAt` to present chronologically.

**If no comments found since last push:**
"Info: No new comments since last push (<TIMESTAMP>). The PR has not received new feedback since your last push."

## Step 4: Present Feedback Summary

Display a summary header with PR context:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PR #<NUMBER>: <TITLE>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Branch: <BRANCH>
Last push: <TIMESTAMP>
PR URL: <URL>

Comments since push: <COUNT> total
  - Conversation: <N>
  - Reviews: <N>
  - Inline: <N>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

For each comment, display:
```
--- [TYPE] Comment by @<AUTHOR> at <TIME> ---
Location: <PATH>:<LINE> (for inline) or "PR conversation"

> <COMMENT BODY>

---
```

## Step 5: Analyze Each Feedback Item

Think about each item critically and do the necessary research in code and online to confirm or refute the claim. Use research agents to verify claims in parallel.

When analyzing feedback, evaluate each item across these **review dimensions**:
- **🔧 Correctness**: Does code work correctly? Edge cases? Logic errors? Error handling?
- **🔒 Security**: Input validation? Hardcoded secrets? Auth/authz correct? Injection risks?
- **⚡ Performance**: Algorithm complexity? N+1 queries? Memory usage? Caching opportunities?
- **🛠️ Maintainability**: Readability? Architecture fit? Testing quality? Standards compliance?

Also assess your **confidence level** in each finding:
- 🎯 **Certain**: Verified through code, tests, or documentation
- 🔍 **Likely**: Strong evidence but may benefit from verification
- ❓ **Uncertain**: Needs discussion, more context, or team input

Consider **technical debt implications**:
- Does the suggested change add technical debt (shortcuts, inconsistencies, workarounds)?
- Does it address existing technical debt (fixes patterns, improves consistency)?
- Or is it neutral (neither adds nor reduces debt)?

Check for **breaking changes**:
- Would this change break API contracts, schemas, or expected behavior?
- Is backward compatibility maintained?

## Output Format

Structure your analysis as follows:

### 1. Summary of Research
Show the parallel research agents launched and their completion status.

### 2. Per-Item Analysis

**IMPORTANT**: Use consistent sequential numbering (1, 2, 3...) for all items. This enables easy follow-up discussion (e.g., "address items 2 and 4").

For each feedback item, provide:
- **Numbered heading** (e.g., "**1.**", "**2.**") with brief description and **verdict badge**:
  - ✅ VALID - ADDRESSED: Concern is correct and our code already handles it properly
  - ✅ VALID - NO ACTION: Valid observation but no change needed (already correct)
  - ⚠️ VALID - IMPROVEMENT: Valid concern that could use improvement
  - ❌ DISAGREE: Respectfully disagree with the suggestion
- **Dimension**: Which review area (🔧 Correctness / 🔒 Security / ⚡ Performance / 🛠️ Maintainability)
- **Claim**: What the reviewer is asserting
- **Research Findings**: Bulleted list with specific code references (file:line)
- **Verdict**: Clear statement on whether the concern is valid
- **Confidence**: How certain you are (🎯 Certain / 🔍 Likely / ❓ Uncertain)
- **Technical Debt Impact**: Whether this adds/addresses/neutral to tech debt
- **Breaking Change**: Whether this affects backward compatibility (💥 Yes / ✓ No / ❔ Unknown)
- **Recommendation**: Specific action (or "Keep as-is" if no change needed)

### 3. Summary Table

| # | Item | Dimension | Valid? | Risk | Confidence | Action | Tech Debt | Breaking? |
|---|------|-----------|--------|------|------------|--------|-----------|-----------|

**Dimension Column Legend** (4 review areas):
- 🔧 **Correctness** - Logic, edge cases, functionality, error handling
- 🔒 **Security** - Input validation, secrets, auth, injection risks
- ⚡ **Performance** - Complexity, queries, memory, caching
- 🛠️ **Maintainability** - Readability, architecture, testing, standards

**Risk Column Legend** (impact if issue reaches production):
- 🔥 **Critical** - Production outage, data loss, security breach → 🔴 MUST fix
- ⚠️ **High** - Functionality broken, user-facing bugs → 🟡 SHOULD fix
- 📊 **Medium** - Performance degradation, debt accumulation → 🟢 COULD fix
- 💭 **Low** - Nit, style preference, minor polish → ⚪ Optional

**Confidence Column Legend**:
- 🎯 **Certain** - Verified through code, tests, or documentation
- 🔍 **Likely** - Strong evidence but may benefit from verification
- ❓ **Uncertain** - Needs discussion, more context, or team input

**Action Column Legend**:
- 🔴 **MUST** - Blocking issue, must address before merge
- 🟡 **SHOULD** - Recommended, improves quality/maintainability
- 🟢 **COULD** - Optional improvement, nice-to-have
- ⚪ **N/A** - No action needed or not recommended
- 🚫 **MUSTN'T** - Suggested change would be harmful

**Tech Debt Column Legend**:
- ➕ **Adds** - Change would add technical debt
- ✅ **Addresses** - Change would reduce/fix existing debt
- ➖ **N/A** - Neutral impact on technical debt

**Breaking Change Column Legend**:
- 💥 **Yes** - API contract, schema, or behavior change
- ✓ **No** - Backward compatible
- ❔ **Unknown** - Needs analysis

### 4. Overall Assessment
State whether the PR is production-ready and summarize blocking vs. non-blocking issues.

---

## Example Output

⏺ Detected PR #1522 from branch `051-enh-compaction-event-column-wal-segments-deduplication`
  Last push: 2025-11-29T18:30:00Z
  Found 6 comments since last push (2 conversation, 1 review, 3 inline)

⏺ 6 Explore agents finished
   ├─ DuckDB connection serialization · 16 tool uses · 40.6k tokens
   │  ⎿ Done
   ├─ SERVICE_ID env var usage · 17 tool uses · 54.5k tokens
   │  ⎿ Done
   ├─ GIN index migration review · 12 tool uses · 37.5k tokens
   │  ⎿ Done
   ├─ YAML validation script review · 1 tool use · 15.7k tokens
   │  ⎿ Done
   ├─ Test helper file review · 7 tool uses · 18.7k tokens
   │  ⎿ Done
   └─ Error wrapping patterns review · 10 tool uses · 28.9k tokens
      ⎿ Done

⏺ PR Feedback Analysis - Critical Review

  I've researched all 6 feedback items since the last push. Here's my assessment:

  ---
  **1.** Race Condition Concern (DuckDB single connection) ✅ VALID - ADDRESSED

  **Dimension**: 🔧 Correctness

  **Claim**: The count query at deduplication.go:122-126 runs without a lock and could have a race condition.

  **Research Findings**:
  - TRUE: DuckDB is configured with SetMaxOpenConns(1) at planner.go:212
  - This is intentional for in-memory DuckDB (separate connections = separate databases)
  - Queries are serialized through the single connection

  **Verdict**: The code IS safe. A comment explaining the serialization dependency could help future maintainers, but it's not a bug. The reviewer's concern is valid to raise but the code is correct.

  **Confidence**: 🎯 Certain - Verified in code at planner.go:212

  **Technical Debt Impact**: ➖ N/A - No debt implications, architecture is sound.

  **Breaking Change**: ✓ No - No behavioral change needed.

  **Recommendation**: Keep as-is. Optional: add explanatory comment.

  ---
  **2.** Environment Variable Coupling (SERVICE_ID) ⚠️ VALID - IMPROVEMENT

  **Dimension**: 🛠️ Maintainability

  **Claim**: os.Getenv("SERVICE_ID") at line 191 creates implicit coupling.

  **Research Findings**:
  - SERVICE_ID is read in 6 places across the codebase
  - The compaction service already has Config.ServiceID loaded at startup (services/compaction/internal/config/config.go:49)
  - The OTELRecorder.RecordDeduplicated() already has a defensive fallback pattern

  **Verdict**: Valid observation. The serviceID could be passed through CompactionInput for cleaner data flow. However:
  - Current code works correctly
  - The metrics recorder has a fallback
  - This is a minor improvement, not a bug

  **Confidence**: 🔍 Likely - Pattern analysis suggests improvement, but needs team discussion on preferred approach.

  **Technical Debt Impact**: ➕ Adds - Implicit env var coupling adds minor debt.

  **Breaking Change**: ✓ No - Internal refactoring only.

  **Recommendation**: Low priority - consider for future refactoring.

  ---
  *(Items 3-6 follow the same format, omitted for brevity)*

  ---
  ### Summary Table

  | # | Item | Dimension | Valid? | Risk | Confidence | Action | Tech Debt | Breaking? |
  |---|------|-----------|--------|------|------------|--------|-----------|-----------|
  | 1 | Race condition (DuckDB) | 🔧 Correct | ✅ Addressed | 💭 Low | 🎯 Certain | ⚪ N/A | ➖ N/A | ✓ No |
  | 2 | SERVICE_ID env coupling | 🛠️ Maintain | ⚠️ Valid | 💭 Low | 🔍 Likely | 🟢 COULD | ➕ Adds | ✓ No |

  ### Overall Assessment

  **Verdict**: ✅ **Production-ready** - The PR is ready to merge.

  - **Blocking issues**: None (no 🔥 Critical or ⚠️ High risk items)
  - **Non-blocking improvements**: Items 2 are valid minor improvements (all 💭 Low risk)
  - **Tech debt summary**: Current PR is neutral; suggestion 2 could add minor debt if not addressed eventually
  - **Breaking changes**: None (all items are backward compatible)
