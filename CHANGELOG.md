## Release v0.1.5-beta (2026-08-29)

### 🚀 New Features & Enhancements
- **Hash-Anchored Edit Tool (`replace_file_content`)**: Surgical line-anchored file edits with SHA-256 preflight checks to prevent race conditions and stale edits.
- **LSP Semantic Code Intelligence (`lsp_tool`)**: Native Language Server Protocol support for Go, TypeScript, and Python with definition, references, hover, diagnostics, and atomic multi-file rename.
- **Persistent REPL Kernel (`persistent_kernel`)**: Stateful execution environment for Python and JavaScript/Node with isolated namespaces, process lifecycle bounds, and stdout/stderr capture.
- **Debug Adapter Protocol Integration (`dap_tool`)**: Full debugging capabilities supporting Delve (Go) and debugpy (Python) with breakpoints, stepping, stack trace inspection, and variable evaluation.
- **Subagent Peer-to-Peer Hub Messaging (`subagent_message`)**: Bounded asynchronous peer communication allowing sibling subagents to coordinate directly without round-tripping through the orchestrator.
- **ShareGPT Trajectory Export (`/export`)**: Export complete conversation, reasoning, and tool execution trajectories formatted for evaluation, fine-tuning, and sharing.
- **Memory Context Stream Scrubber**: Real-time scrubbing of private `<memory-context>` and `<local_memories>` tags from streaming provider responses.
- **Parallel Evaluation Harness (`pkg/evals`)**: Benchmarking suite measuring edit reliability, stale rejection rates, and tool invocation correctness.

### 🐛 Bug Fixes & Stability
- fix(lsp): add lock ordering and transaction rollback on preflight check failure during workspace rename.
- fix(agent): isolate subagent config and provider instances to prevent concurrent session races.
- fix(tools): handle Windows process tree cleanup gracefully with taskkill process group termination.
- fix(mcp): stream-bound individual JSON-RPC frames instead of capping the lifetime stdout reader.
- fix(browserctl): ensure terminal close state flag is set when browser session terminates.
- fix(version): implement unambiguous `compareVersion` to prevent replay and rollback false positives.
## Release v0.1.1-beta (2026-08-22)

### 🚀 New Features
- feat: connect /share directly with mncode-web and add 25+ specialized engineering skills (b344c0f)
- feat(skills): add security-audit, clean-architecture, performance-optimization, api-design, docker-kubernetes and enforce mandatory skill-first agent protocol (9555684)
- feat: add /resolve, /db, /api, /tree, /changelog and embed cost savings in /status (204ac62)
- feat: add /undo, /rewind, /commit, /test, /heal, /review, /share, /pr, /doctor, /symbol, and /scratch (de8c111)
- feat(onboarding): bundle troll mode automatically into brainrot mode selection (b8cbce0)
- feat(onboarding): add interactive brainrot and troll mode selection flow (4369fd0)
- feat(settings): add interrupt_mode setting to toggle default un-prefixed action between queue and steer (da963cf)
- feat: add /diff, /steer, /queue commands and harmless troll prank mode (adfe3eb)
- feat(settings): wire real implementations for worktree_base, copy_on_select, artifacts, auto_mode_plan, and show_tips (ef2599d)
- feat(release): bump version to v0.1.1-beta and clean up prompt slash login command (71cc58c)
- feat(installer): add clear Windows PATH guide and manual recovery steps in install.ps1 (b0658e5)
- feat: add opencode API key config and account login support (1faaff3)
- feat: render /status as interactive temporary overlay dismissable with Esc/Enter/q (85a4c97)
- feat: add mncode web onboarding login, whoami, and feedback integration (b202208)
- feat: add ox-alpha 1M context stealth reasoning model support (0056ec2)
- feat: add /sync telemetry push slash command for mncode-web integration (ac51d9e)
- feat: render live subagent tree and status underneath prompt like Claude Code (0aa3cd3)
- feat: enrich brainrot footer and status spinners with high risk high reward and unhinged dev quotes (67dc342)
- feat: add interactive context window size configuration (200K, 300K, 500K, 1M) and remove emojis from prompt footer (3d75901)
- feat: add view_image tool, clipboard image paste, and multi-line paste collapsing pill (e1cbea4)
- feat: enhance Brainrot Mode with Rizzing statuses, minion tracking, and sigma phrasing (74b841d)

### 🐛 Bug Fixes
- fix(ui): erase model selector table cleanly on Enter and show single success message (4c4f329)
- fix: implement smooth raw-mode ReadModalInput dialog for account logins and handle multi-byte pasted Enter (cef18f2)
- fix: use readLine in /account login to support carriage return Enter on all terminal modes (dbee9b1)
- fix: execute /model immediately on Enter instead of inserting space into prompt (f00e3e9)
- fix: strictly separate English and Vietnamese exit messages based on configured language (6b3d5fa)
- fix: align /workflow spectrum slider layout and rename all ULTRACODE references to ULTRA WORKFLOW (c4915a6)

### ⚡ Performance & Refactoring
- refactor(config): purge dummy Claude Code settings and wire auto_compact and permissions directly (b388a25)
- refactor: rename ULTRACODE to ULTRA WORKFLOW and remove ck: prefix from all skills (1f03535)

### 📝 Documentation
- docs: point README to logo-transparent.svg to invalidate GitHub camo cache (ba852c0)
- docs: store logo asset locally in repo for public README rendering (1eea546)
- docs: update README with web platform sync, ox-alpha, MCP, and interactive overlays (71bbdd0)

