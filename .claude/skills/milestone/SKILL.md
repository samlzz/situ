---
name: milestone
description: Plan, then implement, one situ milestone from docs/dev/milestones.md
disable-model-invocation: true
argument-hint: "<milestone id, e.g. M3>"
---

You are working on milestone $ARGUMENTS of situ.

## Plan (now, in plan mode)

1. Read the section `$ARGUMENTS` of `docs/dev/milestones.md`, then every spec section it lists, in `docs/spec.md` and `docs/format.md`. Read the existing code you will change.
2. Write a plan with:
   - the packages and files to create or change, and the main types and interfaces;
   - the tests to write, each mapped to the numbered cases of the "Tests" section of `docs/spec.md`;
   - every point of the spec you find ambiguous, contradictory or missing. Ask about them now instead of guessing.
3. Leave out everything the milestone lists as out of scope.
4. End the plan with the section "After approval" below, copied word for word: the planning context may be cleared when the plan is approved.

## After approval

1. Implement the plan. Write tests alongside the code. Keep each rule traceable to the spec.
2. Run `make check` and show its full output. Iterate until it passes.
3. Run the verification steps of the milestone and show their output.
4. Report: what was done, the spec test cases covered, any deviation from the plan or the spec, and open questions. Hide nothing.
5. Propose a Conventional Commit message (or several commits). Commit only after the maintainer confirms. Never push, never tag.
