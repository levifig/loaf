/**
 * Slug Generation
 *
 * Generates URL-safe slugs from task titles. Used by `loaf task create`.
 */
export function generateSlug(title: string): string {
  return title
    .toLowerCase()
    .replace(/[`'"]/g, "")           // Remove quotes/backticks
    .replace(/[^a-z0-9]+/g, "-")     // Replace non-alphanumeric with hyphens
    .replace(/^-+|-+$/g, "")         // Trim leading/trailing hyphens
    .slice(0, 50);                    // Max length
}
