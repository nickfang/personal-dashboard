# AI Agent Instructions

This document contains mandatory rules for any AI agent interacting with this repository. Adherence to these rules is required to maintain architectural integrity and developer trust.

## Core Mandates

### 1. Verify, Don't Assume
- **Never assume** a file exists, a library is used, or a configuration is present.
- **Always verify** the actual state of the codebase using tools (`ls`, `grep`, `read_file`, `glob`) before making statements or suggestions.
- If you suspect an issue (e.g., "there might be a go.mod in the frontend"), you **must check first**. Do not present a hypothetical problem as a fact.

### 2. Respect the Architectural Vision
- Adhere strictly to the decisions documented in the `docs/` folder (e.g., Shared Gen Module strategy).
- Do not suggest "standard" or "idiomatic" patterns that contradict established project decisions without a compelling reason and explicit confirmation.

### 3. Stop and Confirm
- Do not implement multiple steps in a single turn unless explicitly directed.
- After completing a task or identifying a problem, stop and ask for direction.
- Do not "helpfully" implement follow-up tasks (like creating tests or updating related files) without asking first.

### 4. Architectural Integrity
- **Separation of Concerns:** Ensure logic belongs in the correct layer (Transport, Service, Repository). Do not leak implementation details across boundaries.
- **DRY (Don't Repeat Yourself):** Actively look for duplicated logic or patterns. Solve problems at the highest appropriate level (e.g., shared library vs copy-paste) rather than patching locally.
- **Holistic Approach:** When solving an issue, do not just fix the symptom in front of you. Analyze the system-wide implications. Ask: "Is this the right place for this logic? Does this create debt elsewhere?"

### 5. Direct and Concise
- Keep explanations technical and brief.
- Avoid conversational filler or apologies. Focus on the data and the task at hand.

---

*Any violation of these rules, especially making unverified assumptions about the codebase, is considered a failure in the agent's primary mission.*
