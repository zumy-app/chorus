-- 00_adopt_legacy_ids.sql: Legacy-content adoption (en->es).
-- Deploy-synced seeds assume canonical UUIDs. Databases that ran the legacy
-- runtime-seeded curriculum hold the same content under random UUIDs, which
-- makes the natural-key upserts in 01-09 match WITHOUT inserting the canonical
-- rows (and the FKs to canonical ids then fail).
-- This file swaps any legacy row whose (course, slug) matches a canonical
-- entity onto the canonical UUID, preserving all learner data:
--   1. stage the canonical row under a temporary slug (unique-index safe),
--   2. repoint every referencing table from the legacy id to the canonical id,
--   3. delete the legacy row,
--   4. restore the real slug on the canonical row.
-- Every intermediate state satisfies the foreign keys. On a fresh database all
-- lookups come back empty and the whole file is a no-op.

-- ── A. Course ───────────────────────────────────────────────────────────────
DO $$
DECLARE
    canonical uuid := 'c0000000-0000-0000-0000-000000000001';
    legacy uuid;
BEGIN
    SELECT id INTO legacy FROM curriculum_courses
    WHERE native_language = 'en' AND target_language = 'es' AND version = 'v1'
      AND id <> canonical
    LIMIT 1;
    IF legacy IS NOT NULL THEN
        INSERT INTO curriculum_courses (id, native_language, target_language, title, version, support_tier, is_active)
        VALUES (canonical, 'en', 'es', 'Spanish for English Speakers', 'v1-adopt-tmp', 'full_course', true)
        ON CONFLICT (id) DO NOTHING;

        UPDATE learning_pair_capabilities SET active_course_id = canonical WHERE active_course_id = legacy;
        UPDATE user_language_profiles SET active_course_id = canonical WHERE active_course_id = legacy;
        UPDATE curriculum_units SET course_id = canonical WHERE course_id = legacy;
        UPDATE grammar_points SET course_id = canonical WHERE course_id = legacy;
        UPDATE lexical_items SET course_id = canonical WHERE course_id = legacy;
        UPDATE scenario_scripts SET course_id = canonical WHERE course_id = legacy;
        UPDATE placement_items SET course_id = canonical WHERE course_id = legacy;
        UPDATE reading_passages SET course_id = canonical WHERE course_id = legacy;
        UPDATE real_talk_prompts SET course_id = canonical WHERE course_id = legacy;

        DELETE FROM curriculum_courses WHERE id = legacy;
        UPDATE curriculum_courses SET version = 'v1' WHERE id = canonical;
    END IF;
END $$;

-- ── B. Units ───────────────────────────────────────────────────────────────
DO $$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT u.id AS legacy_id, u.slug, u.ordinal, m.canonical_id
        FROM curriculum_units u
        JOIN curriculum_courses c ON c.id = u.course_id
        JOIN (VALUES
            ('a1-introductions',              'a0000000-0000-0000-0000-000000000001'::uuid),
            ('a1-basics-ii',                  'a0000000-0000-0000-0000-000000000002'::uuid),
            ('a1-daily-routine',              'a0000000-0000-0000-0000-000000000003'::uuid),
            ('a1-ordering-food',              'a0000000-0000-0000-0000-000000000004'::uuid),
            ('a1-around-town',                'a0000000-0000-0000-0000-000000000005'::uuid),
            ('a1-checkpoint',                 'a0000000-0000-0000-0000-000000000006'::uuid),
            ('a2-past-weekend',               'a0000000-0000-0000-0000-000000000007'::uuid),
            ('a2-shopping',                   'a0000000-0000-0000-0000-000000000008'::uuid),
            ('a2-plans',                      'a0000000-0000-0000-0000-000000000009'::uuid),
            ('a2-health-feelings',            'a0000000-0000-0000-0000-000000000010'::uuid),
            ('a2-travel-basics',              'a0000000-0000-0000-0000-000000000011'::uuid),
            ('a2-checkpoint',                 'a0000000-0000-0000-0000-000000000012'::uuid),
            ('b1-stories',                    'a0000000-0000-0000-0000-000000000013'::uuid),
            ('b1-opinions',                   'a0000000-0000-0000-0000-000000000014'::uuid),
            ('b1-problems',                   'a0000000-0000-0000-0000-000000000015'::uuid),
            ('b1-social-plans',               'a0000000-0000-0000-0000-000000000016'::uuid),
            ('b1-work-study',                 'a0000000-0000-0000-0000-000000000017'::uuid),
            ('b1-checkpoint',                 'a0000000-0000-0000-0000-000000000018'::uuid),
            ('b2-nuanced-opinions',           'a0000000-0000-0000-0000-000000000019'::uuid),
            ('b2-hypotheticals',              'a0000000-0000-0000-0000-000000000020'::uuid),
            ('b2-professional-communication', 'a0000000-0000-0000-0000-000000000021'::uuid),
            ('b2-media-culture',              'a0000000-0000-0000-0000-000000000022'::uuid),
            ('b2-conflict-repair',            'a0000000-0000-0000-0000-000000000023'::uuid),
            ('b2-checkpoint',                 'a0000000-0000-0000-0000-000000000024'::uuid)
        ) AS m(slug, canonical_id) ON m.slug = u.slug
        WHERE c.id = 'c0000000-0000-0000-0000-000000000001'
          AND u.id <> m.canonical_id
    LOOP
        INSERT INTO curriculum_units (id, course_id, cefr_level, ordinal, slug, title, can_do_statement, description, estimated_minutes, checkpoint_required)
        SELECT r.canonical_id, u.course_id, u.cefr_level, u.ordinal + 1000, u.slug || '-adopt-tmp', u.title,
               u.can_do_statement, u.description, u.estimated_minutes, u.checkpoint_required
        FROM curriculum_units u WHERE u.id = r.legacy_id
        ON CONFLICT (id) DO NOTHING;

        UPDATE vocabulary SET curriculum_unit_id = r.canonical_id WHERE curriculum_unit_id = r.legacy_id;
        UPDATE user_language_profiles SET active_unit_id = r.canonical_id WHERE active_unit_id = r.legacy_id;
        UPDATE grammar_points SET unit_id = r.canonical_id WHERE unit_id = r.legacy_id;
        UPDATE lexical_items SET unit_id = r.canonical_id WHERE unit_id = r.legacy_id;
        UPDATE curriculum_lessons SET unit_id = r.canonical_id WHERE unit_id = r.legacy_id;
        UPDATE scenario_scripts SET unit_id = r.canonical_id WHERE unit_id = r.legacy_id;
        UPDATE user_unit_progress SET unit_id = r.canonical_id WHERE unit_id = r.legacy_id;
        UPDATE mined_items SET curriculum_unit_id = r.canonical_id WHERE curriculum_unit_id = r.legacy_id;
        UPDATE learning_sessions SET source_unit_id = r.canonical_id WHERE source_unit_id = r.legacy_id;
        UPDATE reading_passages SET unit_id = r.canonical_id WHERE unit_id = r.legacy_id;

        DELETE FROM curriculum_units WHERE id = r.legacy_id;
        UPDATE curriculum_units SET slug = r.slug, ordinal = r.ordinal WHERE id = r.canonical_id;
    END LOOP;
END $$;

-- ── C. Grammar points ──────────────────────────────────────────────────────
DO $$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT gp.id AS legacy_id, gp.slug, m.canonical_id
        FROM grammar_points gp
        JOIN (VALUES
            ('ser-personal-pronouns',           'b0000000-0000-0000-0000-000000000001'::uuid),
            ('yes-no-questions-question-words', 'b0000000-0000-0000-0000-000000000002'::uuid),
            ('present-regular-reflexive-intro', 'b0000000-0000-0000-0000-000000000003'::uuid),
            ('querer-quisiera-articles',         'b0000000-0000-0000-0000-000000000004'::uuid),
            ('estar-hay-prepositions',           'b0000000-0000-0000-0000-000000000005'::uuid),
            ('preterite-regular-ir-ser',         'b0000000-0000-0000-0000-000000000006'::uuid),
            ('demonstratives-direct-objects',    'b0000000-0000-0000-0000-000000000007'::uuid),
            ('ir-a-infinitive-tener-que',        'b0000000-0000-0000-0000-000000000008'::uuid),
            ('preterite-vs-imperfect',           'b0000000-0000-0000-0000-000000000009'::uuid),
            ('subjunctive-triggers-intro',       'b0000000-0000-0000-0000-000000000010'::uuid)
        ) AS m(slug, canonical_id) ON m.slug = gp.slug
        WHERE gp.course_id = 'c0000000-0000-0000-0000-000000000001'
          AND gp.id <> m.canonical_id
    LOOP
        INSERT INTO grammar_points (id, course_id, unit_id, slug, cefr_level, title, short_explanation,
                                    examples, prerequisites, metadata, rule_text, common_trap, ordinal, is_active)
        SELECT r.canonical_id, gp.course_id, gp.unit_id, gp.slug || '-adopt-tmp', gp.cefr_level, gp.title,
               gp.short_explanation, gp.examples, gp.prerequisites, gp.metadata,
               gp.rule_text, gp.common_trap, gp.ordinal, gp.is_active
        FROM grammar_points gp WHERE gp.id = r.legacy_id
        ON CONFLICT (id) DO NOTHING;

        UPDATE user_grammar_item_attempts SET grammar_point_id = r.canonical_id WHERE grammar_point_id = r.legacy_id;
        UPDATE user_grammar_mastery SET grammar_point_id = r.canonical_id WHERE grammar_point_id = r.legacy_id;
        UPDATE learning_session_items SET grammar_point_id = r.canonical_id WHERE grammar_point_id = r.legacy_id;
        UPDATE grammar_items SET grammar_point_id = r.canonical_id WHERE grammar_point_id = r.legacy_id;

        DELETE FROM grammar_points WHERE id = r.legacy_id;
        UPDATE grammar_points SET slug = r.slug WHERE id = r.canonical_id;
    END LOOP;
END $$;

-- ── D. Scenarios ───────────────────────────────────────────────────────────
DO $$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT ss.id AS legacy_id, ss.slug, m.canonical_id
        FROM scenario_scripts ss
        JOIN (VALUES
            ('ordering-coffee', 'd0000000-0000-0000-0000-000000000001'::uuid),
            ('hotel-checkin',   'd0000000-0000-0000-0000-000000000002'::uuid)
        ) AS m(slug, canonical_id) ON m.slug = ss.slug
        WHERE ss.course_id = 'c0000000-0000-0000-0000-000000000001'
          AND ss.id <> m.canonical_id
    LOOP
        INSERT INTO scenario_scripts (id, course_id, unit_id, slug, title, domain, cefr_level, can_do_statement,
                                      ai_role_name, ai_role_description, opening_line, max_turns, estimated_minutes,
                                      scaffold_initial, completion_criteria, metadata)
        SELECT r.canonical_id, ss.course_id, ss.unit_id, ss.slug || '-adopt-tmp', ss.title, ss.domain, ss.cefr_level,
               ss.can_do_statement, ss.ai_role_name, ss.ai_role_description, ss.opening_line, ss.max_turns,
               ss.estimated_minutes, ss.scaffold_initial, ss.completion_criteria, ss.metadata
        FROM scenario_scripts ss WHERE ss.id = r.legacy_id
        ON CONFLICT (id) DO NOTHING;

        UPDATE scenario_phases SET scenario_id = r.canonical_id WHERE scenario_id = r.legacy_id;
        UPDATE scenario_runs SET scenario_id = r.canonical_id WHERE scenario_id = r.legacy_id;

        DELETE FROM scenario_scripts WHERE id = r.legacy_id;
        UPDATE scenario_scripts SET slug = r.slug WHERE id = r.canonical_id;
    END LOOP;
END $$;
