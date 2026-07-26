#!/usr/bin/env node

const MAX_BODY_CHARS = 700;
const REMINDER = [
  "GitHub prose must be factual, neutral, and terse.",
  "The initial acknowledgement of an external issue is exactly \"Investigating.\".",
  "Do not converse with automated bots or criticize previous comments.",
].join(" ");

function readStdin() {
  return new Promise((resolve) => {
    let data = "";
    process.stdin.setEncoding("utf8");
    process.stdin.on("data", (chunk) => (data += chunk));
    process.stdin.on("end", () => resolve(data));
    process.stdin.on("error", () => resolve(""));
  });
}

function allow(reminder) {
  if (reminder) process.stderr.write(`${reminder}\n`);
  process.exit(0);
}

function deny(reason) {
  process.stdout.write(`${JSON.stringify({
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: reason,
    },
  })}\n`);
  process.exit(0);
}

function isGuardedCommand(command) {
  const compact = command.replace(/\s+/g, " ");
  return /\bgh\b\s+(issue|pr)\s+(comment|create)\b/.test(compact);
}

function extractBody(command) {
  if (/(--body-file|(?<![\w-])-F)(\s+|=)/.test(command)) {
    return { body: null, fromFile: true };
  }

  const candidates = [];
  for (const match of command.matchAll(/(?:--body|(?<![\w-])-b)(?:\s+|=)(['"])([\s\S]*?)\1/g)) {
    candidates.push(match[2]);
  }
  for (const match of command.matchAll(/(?:--body|(?<![\w-])-b)=(\S+)/g)) {
    candidates.push(match[1]);
  }
  for (const match of command.matchAll(/<<-?\s*(['"]?)(\w+)\1\s*\n([\s\S]*?)\n\s*\2\b/g)) {
    candidates.push(match[3]);
  }

  if (candidates.length === 0) return { body: null, fromFile: false };
  return { body: candidates.reduce((a, b) => (a.length >= b.length ? a : b), ""), fromFile: false };
}

async function main() {
  let payload;
  try {
    payload = JSON.parse(await readStdin());
  } catch {
    allow();
    return;
  }

  const command = payload?.tool_input?.command;
  if (typeof command !== "string" || !command.trim() || !isGuardedCommand(command)) {
    allow();
    return;
  }

  if (process.env.MEW_ALLOW_LONG_COMMENT === "1") {
    allow(`${REMINDER} [MEW_ALLOW_LONG_COMMENT=1: length check bypassed]`);
    return;
  }

  const { body, fromFile } = extractBody(command);
  if (fromFile || body == null) {
    allow(REMINDER);
    return;
  }

  if (body.length > MAX_BODY_CHARS) {
    deny(
      `Blocked: GitHub body is ${body.length} characters; the limit is ${MAX_BODY_CHARS}. ` +
      `${REMINDER} Rewrite it to the fewest words that preserve the facts. ` +
      "Use MEW_ALLOW_LONG_COMMENT=1 only when a genuinely longer body is required."
    );
    return;
  }

  allow(REMINDER);
}

main();
