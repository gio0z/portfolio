import content from '../content/portfolio.json'
import type { Profile, Project, SkillCategory } from '../types'

/**
 * Typed access to the authored portfolio content.
 *
 * The JSON is generated from the Go source of truth by tools/contentexport, so
 * the prerendered pages and the live /api/* endpoints serve the same data.
 * Never edit the JSON: change pkg/api/content.go and regenerate.
 */
export const profile = content.profile as Profile
export const projects = content.projects as Project[]
export const skillCategories = content.skill_categories as SkillCategory[]
