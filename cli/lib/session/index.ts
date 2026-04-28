/**
 * Session library — public re-exports.
 *
 * Imports from this module:
 *
 *   import {
 *     findActiveSessionForBranch,
 *     findSessionByClaudeId,
 *     resolveCurrentSession,
 *     parseHookSessionId,
 *     readSessionFile,
 *     SessionFrontmatter,
 *   } from "../lib/session/index.js";
 */

export {
  consolidateSession,
  extractJournalLines,
  getDateTimeString,
  getTimestamp,
  readSessionFile,
  writeFileAtomic,
  type SessionFrontmatter,
  type SpecFrontmatterWithBranch,
} from "./store.js";

export { findActiveSessionForBranch, findSessionByClaudeId } from "./find.js";

export {
  parseHookSessionId,
  resolveCurrentSession,
  type ResolveCurrentSessionOptions,
  type ResolvedSession,
} from "./resolve.js";
