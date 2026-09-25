import { RolloutFlagKey } from '@chorus/shared';
import apiService from '../services/api';
import storage from './storage';

// Local-only override set. These are intentionally independent from the
// server-controlled rollout flags so dev builds can force a feature on/off
// without affecting the backend resolution.
const DEFAULT_FLAGS: Record<RolloutFlagKey, boolean> = {
  grammar_insights: true,
  word_collector: true,
  feature_voting: true,
  referral_bump: true,
  google_oauth: false,
  push_notifications: false,
  word_flashcards: false,
  ai_writing_assistant: false,
  placement_test: false,
  scenario_roleplay: false,
  voice_message: false,
  media_sharing: false,
  document_sharing: false,
  location_sharing: false,
  video_calls: false,
  study_pods: false,
  teacher_marketplace: false,
  payout_teacher: false,
  xp_leaderboard: false,
  group_chat: false,
  gdpr_export: false,
  social_feed: false,
  learning_v3_engine: true,
  redis_session_cache: true,
};

class FeatureFlagManager {
  private serverFlags: Record<string, boolean> = {};
  private localOverrides: Record<string, boolean> = {};
  private initialized = false;

  async init() {
    if (this.initialized) return;
    try {
      const stored = await storage.getItem('feature_flags_override');
      if (stored) {
        this.localOverrides = JSON.parse(stored);
      }
    } catch {
      // Use empty overrides
    }

    try {
      this.serverFlags = await apiService.featureFlags();
    } catch {
      // Backend unavailable or not authenticated — fall back to defaults.
      this.serverFlags = {};
    }

    this.initialized = true;
  }

  isEnabled(flag: RolloutFlagKey): boolean {
    // Local override wins (dev tooling / QA).
    if (flag in this.localOverrides) {
      return this.localOverrides[flag];
    }
    // Server-resolved flag next.
    if (flag in this.serverFlags) {
      return this.serverFlags[flag];
    }
    // Safe default finally.
    return DEFAULT_FLAGS[flag] ?? false;
  }

  async setOverride(flag: RolloutFlagKey, value: boolean) {
    this.localOverrides[flag] = value;
    try {
      await storage.setItem('feature_flags_override', JSON.stringify(this.localOverrides));
    } catch {
      // Ignore storage errors
    }
  }

  async clearOverride(flag: RolloutFlagKey) {
    delete this.localOverrides[flag];
    try {
      await storage.setItem('feature_flags_override', JSON.stringify(this.localOverrides));
    } catch {
      // Ignore storage errors
    }
  }

  /** Refetch flags from the server (call after login/token refresh). */
  async refresh() {
    try {
      this.serverFlags = await apiService.featureFlags();
    } catch {
      this.serverFlags = {};
    }
  }
}

export const featureFlags = new FeatureFlagManager();
export default featureFlags;
