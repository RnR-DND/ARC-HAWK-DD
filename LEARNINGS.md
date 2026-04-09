# Operational Learnings — ARC-HAWK-DD

Captured during the DPDPA gap-close session (2026-04-09).

---

1. **Subagent line number hallucination on large diffs**

   When subagents analyze diffs spanning 300+ files, they hallucinate line numbers roughly 50% of the time. During this session, reported line numbers 613, 7462, and 7850 were given for files that were 275, 300, and 400 lines long respectively — none of the reported lines existed.

   **Rule:** Always read the target file before applying any subagent-suggested edit. Never trust a reported line number directly; verify it against an actual file read.

---

2. **CWD drift in Bash sessions**

   After running a command such as `cd apps/frontend && npx jest`, the working directory persists in that subdirectory for all subsequent Bash tool calls within the same session. Commands that assume the repo root will silently operate on the wrong path.

   **Rule:** Always use absolute paths (e.g. `/Users/Bharat/Downloads/ARC-HAWK-DD/`) in Bash commands, or prefix every command with `cd /repo/root &&` to reset the working directory explicitly.

---

3. **Gin router: static routes must be registered before wildcard routes**

   `DELETE /scans/clear` must be registered before `DELETE /scans/:id`. Gin matches the first registered route that fits, so a wildcard route registered earlier will capture literal path segments like `clear` before the static route is reached. This bug only surfaces on DELETE routes where a literal segment looks like an ID.

   **Rule:** In Gin, always register static (literal) routes before any wildcard route that shares the same path prefix.

---

4. **Postgres 11+ constant default optimization**

   `ALTER TABLE ... ADD COLUMN col TEXT NOT NULL DEFAULT 'literal'` does not cause a full table rewrite on Postgres 11+. The constant default is stored in the pg_attribute catalog and applied lazily. Only expression defaults or volatile functions (e.g. `now()`, `gen_random_uuid()`) trigger a rewrite.

   **Rule:** Constant-default `ADD COLUMN` migrations are safe on large tables in Postgres 11+. Document this explicitly in migration review checklists to avoid unnecessary table-copy migration patterns.

---

5. **Jest vs Vitest detection**

   Before running frontend tests, always check the `"test"` script in `package.json`. This project uses Jest (`"test": "jest"`), not Vitest. Running `npx vitest run` will exit with zero findings because it silently discovers no test files, giving a false green signal.

   **Rule:** Check `package.json` scripts before invoking any test runner. Do not assume Vitest just because the project uses React or Vite.

---

6. **ReDoS pattern for user-supplied regex**

   Any API that accepts user-supplied regular expressions is a ReDoS vector. The safe gating pattern is: (1) length check of 512 characters or fewer, (2) reject nested quantifiers via the pattern `(\([^)]*[+*][^)]*\)[+*]|\([^)]*\|[^)]*\)[+*])`, (3) compile with `re.compile()` inside a timeout. Compiled patterns should be cached by a `(name, regex)` tuple to avoid recompilation overhead.

   **Rule:** Gate all user-supplied regex through length + nested-quantifier checks and a compile timeout before use. Cache compiled results.

---

7. **Locale-dependent number formatting in tests**

   `toLocaleString()` produces different output across environments. In an Indian locale, `(1500000).toLocaleString()` returns `"15,00,000"` instead of `"1,500,000"`. Tests that hardcode the expected formatted string will fail in CI environments with a different locale.

   **Rule:** Tests that assert on formatted numbers must use `(n).toLocaleString()` dynamically to generate the expected value, rather than hardcoding a locale-specific string.

---

8. **JSX comment in ternary expression**

   Placing `{/* comment */}` inside a JSX ternary arm causes a TypeScript parse error. The JSX comment syntax is treated as an expression node, which is not valid as a standalone expression inside a ternary arm.

   **Rule:** Never put `{/* ... */}` comments inside a ternary arm. Move comments outside the ternary or remove them.
