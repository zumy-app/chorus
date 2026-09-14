import storage from './storage';

export type FeatureFlag =
  | 'learning_v3_engine'
  | 'multi_mode_placement'
  | 'grammar_drills'
  | 'async_ai_partner'
  | 'teacher_assignments'
  | 'quick_drill_mode';

const DEFAULT_FLAGS: Record<FeatureFlag, boolean> = {
  learning_v3_engine: true,
  multi_mode_placement: true,
  grammar_drills: true,
  async_ai_partner: true,
  teacher_assignments: true,
  quick_drill_mode: true,
};

class FeatureFlagManager {
  private cache: Record<string, boolean> = { ...DEFAULT_FLAGS };

  async init() {
    try {
      const stored = await storage.getItem('feature_flags');
      if (stored) {
        const parsed = JSON.parse(stored);
        this.cache = { ...this.cache, ...parsed };
      }
    } catch {
      // Use defaults
    }
  }

  isEnabled(flag: FeatureFlag): boolean {
    return this.cache[flag] ?? DEFAULT_FLAGS[flag] ?? false;
  }

  async setOverride(flag: FeatureFlag, value: boolean) {
    this.cache[flag] = value;
    try {
      await storage.setItem('feature_flags', JSON.stringify(this.cache));
    } catch {
      // Ignore
    }
  }
}

export const featureFlags = new FeatureFlagManager();
export default featureFlags;
