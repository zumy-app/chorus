import { useMemo } from 'react'
import type { RolloutFlagKey } from '@chorus/shared'
import { useStore } from '../store'

// Default flags mirror the server-side stable tier for the web client.
// They are only used while flags have not loaded yet or if the request fails.
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
}

export function useFeatureFlag(key: RolloutFlagKey): { enabled: boolean; loading: boolean } {
  const flags = useStore((state) => state.rolloutFlags)

  return useMemo(() => {
    const loading = flags === null
    const enabled = flags ? flags[key] ?? DEFAULT_FLAGS[key] ?? false : DEFAULT_FLAGS[key] ?? false
    return { enabled, loading }
  }, [flags, key])
}

export function useAllFeatureFlags(): { flags: Record<RolloutFlagKey, boolean>; loading: boolean } {
  const flags = useStore((state) => state.rolloutFlags)

  return useMemo(() => {
    const loading = flags === null
    const resolved = { ...DEFAULT_FLAGS, ...(flags ?? {}) } as Record<RolloutFlagKey, boolean>
    return { flags: resolved, loading }
  }, [flags])
}
