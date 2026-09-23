-- 01_course.sql: English for Spanish Speakers Course and Capabilities
-- (legacy-id adoption runs first in 00_adopt_legacy_ids.sql)
INSERT INTO curriculum_courses (
    id, native_language, target_language, title, version, support_tier, is_active
)
VALUES (
    'c0000000-0000-0000-0000-000000000002',
    'es', 'en', 'English for Spanish Speakers', 'v1', 'full_course', true
)
ON CONFLICT (target_language, native_language, version) DO UPDATE SET
    title = EXCLUDED.title,
    support_tier = EXCLUDED.support_tier,
    is_active = true;

INSERT INTO learning_pair_capabilities (
    native_language, target_language, support_tier, active_course_id,
    placement_enabled, roadmap_enabled, scenarios_enabled,
    srs_enabled, mining_enabled, grammar_feedback_enabled, quality_notes
)
VALUES (
    'es', 'en', 'full_course',
    'c0000000-0000-0000-0000-000000000002',
    true, true, true, true, true, true,
    'Second pair (V3 Phase 9): curated A1-B2 English path for Spanish speakers.'
)
ON CONFLICT (native_language, target_language) DO UPDATE SET
    support_tier = EXCLUDED.support_tier,
    active_course_id = EXCLUDED.active_course_id,
    placement_enabled = EXCLUDED.placement_enabled,
    roadmap_enabled = EXCLUDED.roadmap_enabled,
    scenarios_enabled = EXCLUDED.scenarios_enabled,
    srs_enabled = EXCLUDED.srs_enabled,
    mining_enabled = EXCLUDED.mining_enabled,
    grammar_feedback_enabled = EXCLUDED.grammar_feedback_enabled,
    quality_notes = EXCLUDED.quality_notes,
    updated_at = CURRENT_TIMESTAMP;
