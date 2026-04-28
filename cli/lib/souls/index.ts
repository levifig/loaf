/**
 * Souls library — public re-exports.
 *
 * Imports from this module:
 *
 *   import {
 *     listSouls,
 *     readSoul,
 *     checkDivergence,
 *     copySoulToProject,
 *     readActiveSoul,
 *     writeActiveSoul,
 *   } from "../lib/souls/index.js";
 */

export {
  extractDescription,
  listSouls,
  readSoul,
  soulPathFor,
  type SoulEntry,
} from "./catalog.js";

export {
  catalogHashes,
  checkDivergence,
  hashFile,
  sha256,
  type DivergenceResult,
} from "./divergence.js";

export {
  copySoulToProject,
  loafConfigPath,
  localSoulPath,
  readActiveSoul,
  writeActiveSoul,
} from "./install.js";

export { findLoafRoot, getCatalogDir } from "./paths.js";
