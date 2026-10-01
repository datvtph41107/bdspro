# Active Coding & Source-Trace Practice Protocol

## Purpose

This document is the durable working contract for how BDSPro is learned and
implemented.

The objective is not fast source production. The objective is to build the
reflexes of a working backend engineer:

- derive code from requirements and pressure;
- write code personally rather than reproduce supplied solutions;
- read compiler/test/runtime feedback;
- trace unfamiliar APIs through primary documentation and source;
- form a hypothesis before changing code;
- explain mechanisms in the user's own words;
- build speed only after understanding and repetition.

Slower progress is acceptable when it produces transferable engineering skill.

## Non-negotiable active-coding rule

The user writes the implementation.

For a new function, test, query, Make target, shell workflow or configuration
mechanism, the default interaction is:

```text
PROBLEM / REQUIREMENT
  -> USER EXPLAINS CURRENT UNDERSTANDING
  -> IDENTIFY INPUT / OUTPUT / INVARIANT / SIDE EFFECT
  -> RESEARCH PRIMARY SOURCE WHEN NEEDED
  -> USER PREDICTS BEHAVIOR
  -> USER WRITES THE SMALLEST CODE
  -> RUN / COMPILE / TEST
  -> READ REAL OUTPUT
  -> FORM HYPOTHESIS
  -> DEBUG / REVISE
  -> USER EXPLAINS WHAT WAS LEARNED
  -> GENERALIZE
```

The assistant must not make copy/paste the normal path.

## Solution-escalation ladder

For a new coding task, use the weakest assistance that lets the user continue:

```text
LEVEL 0 — requirement / pressure only
LEVEL 1 — keywords, API names, or syntax shape
LEVEL 2 — pseudocode / control-flow skeleton
LEVEL 3 — one focused fragment around the blocked mechanism
LEVEL 4 — complete implementation only after a real attempt,
          when debugging/recovery requires it,
          or when the user explicitly asks for the full solution
```

Do not jump to Level 4 merely because it is faster.

A long complete function/file shown before the user has reasoned about it is a
training failure unless there is a specific recovery/debugging reason.

## Code-from-intent reflex

Before writing implementation, the user should be able to answer, in ordinary
language:

```text
What problem am I solving?
What must be true before this code runs?
What inputs does it receive?
What output or state change should it produce?
What invariant must remain true?
What can fail?
Who owns any resource created here?
How will I observe that it worked?
```

Names should emerge from the responsibility. A test name, function name or Make
target should be explainable before its body is written.

## Test-writing reflex

Do not begin a new test by copying a complete `TestXxx`.

First write the scenario mentally or in notes:

```text
REQUIREMENT
INVARIANT
GIVEN / PRECONDITION
WHEN / ACTION
THEN / EXPECTED OUTCOME
OBSERVATION POINT
WHAT THIS TEST DOES NOT PROVE
```

Then choose a test name that expresses the behavior. Only then write the Go test
body personally.

The assistant may provide the Go testing API or a small syntax skeleton, but
should not supply the complete test by default.

## Source-trace reflex

When an unfamiliar library function, CLI flag, language mechanism or framework
behavior appears, research is part of the coding task rather than optional
background reading.

Preferred evidence order:

```text
1. language/tool official documentation or specification
2. package/API documentation
3. implementation source when semantics remain unclear
4. focused local experiment
5. mature OSS usage as comparative evidence
6. tutorials/blogs only as secondary explanation
```

For an unfamiliar symbol, practice this trace:

```text
CALL SITE
  -> SYMBOL / SIGNATURE
  -> OFFICIAL DOC CONTRACT
  -> RELEVANT SOURCE IMPLEMENTATION IF NEEDED
  -> INPUT / OUTPUT / ERRORS / SIDE EFFECTS / LIFECYCLE
  -> LOCAL EXPERIMENT
  -> EXPLAIN IN OWN WORDS
  -> USE IN BDSPro
```

Do not accept an API merely because the assistant says what it does.

## Documentation-reading requirement

When a mechanism is material to the current pressure, the user should read the
relevant primary documentation and produce a short interpretation in their own
words.

The interpretation should answer:

```text
What does the documentation actually guarantee?
What does it not guarantee?
Which sentence/behavior matters to BDSPro?
How does that map to the code I am about to write?
```

The assistant should help locate the smallest relevant section rather than dump a
large documentation summary first.

## Debugging reflex

When code fails, do not immediately replace it with working code.

Default loop:

```text
READ exact error/output
  -> LOCATE failing boundary
  -> EXPLAIN what the message literally says
  -> FORM one or more hypotheses
  -> INSPECT docs/source/state if needed
  -> RUN smallest discriminating experiment
  -> CHANGE smallest thing
  -> RUN again
  -> EXPLAIN why the fix worked
```

Compiler errors, SQL errors, test failures and runtime failures are training
material.

## Repetition and fluency

Repeatedly typing small patterns is desirable early in learning.

Do not abstract repetition merely because it exists once or twice. First acquire
the underlying reflex. Extract helpers/abstractions when repetition creates real
maintenance or correctness pressure and the user can still explain what the
abstraction hides.

The target progression is:

```text
understand slowly
 -> type repeatedly
 -> recognize pattern
 -> predict behavior
 -> debug naturally
 -> abstract deliberately
 -> work faster without losing mechanism
```

## SimpleBank and mature OSS

`techschool/simplebank` is a learning/reference case, not a source template.

For every candidate idea found there:

```text
What pressure exists in SimpleBank?
What mechanism was chosen?
What does the code actually do?
Does BDSPro have the same pressure now?
What is different in BDSPro?
Can I reproduce the mechanism myself without copying the implementation?
```

Only then may the idea influence BDSPro.

The same rule applies to all mature OSS.

## Assistant behavior contract

The assistant should normally:

- ask/derive the current pressure before code;
- provide small syntax shapes, signatures, API names and search terms;
- ask the user to predict behavior when useful;
- require the user to type the implementation;
- review the user's actual code rather than replace it;
- use official sources and source tracing when APIs/mechanisms are unfamiliar;
- connect errors to mechanisms;
- preserve the larger engineering goal while zooming into one blocked concept.

The assistant should normally avoid:

- complete paste-ready functions/files before the user attempts them;
- giant scaffolds;
- generators that hide a mechanism currently being learned;
- copying SimpleBank/OSS structures;
- premature helpers/interfaces/layers;
- answering a documentation question solely from memory when the source can be
  inspected;
- moving on after a green command if the user cannot explain why it is green.

## Progress evidence

A learning step is accepted when the user can do most of the following:

```text
state the requirement
predict the mechanism
write the code with limited prompting
run it
interpret the output
debug a plausible failure
explain the code in their own words
identify the relevant official/source reference
connect it to the larger system pressure
```

Passing code alone is not sufficient evidence of learning.

## Durable rule

This protocol is part of BDSPro's working method.

A future assistant must read it before proposing implementation. If convenience
or speed conflicts with active skill formation, prefer active skill formation
unless the user explicitly asks to temporarily switch modes.

## Long-horizon repetition

A single correct attempt is not the training endpoint. Important primitives must
recur across realistic work until the user can recognize, research, execute,
debug and transfer them with limited prompting.

The broader repetition, failure-escalation and multi-role model is defined in
`docs/continuity/DEVELOPER-IMMERSION.md`. This protocol governs each individual
practice step inside that larger model.
