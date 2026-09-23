-- 02_units.sql: Units for English A1-B2 Course (Spanish speakers)
INSERT INTO curriculum_units (
    id, course_id, cefr_level, ordinal, slug, title, can_do_statement, description, estimated_minutes, checkpoint_required
)
VALUES
    ('a0000001-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002', 'A1', 1, 'a1-introductions', 'Introductions', 'I can greet someone and introduce myself.', 'Start simple conversations and share basic identity.', 30, false),
    ('a0000001-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000002', 'A1', 2, 'a1-basics-ii', 'Basics II', 'I can ask simple identity and language questions.', 'Ask and answer simple questions about people and languages.', 30, false),
    ('a0000001-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000002', 'A1', 3, 'a1-daily-routine', 'Daily Routine', 'I can talk about simple daily habits.', 'Describe everyday actions and simple routines.', 35, false),
    ('a0000001-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000002', 'A1', 4, 'a1-ordering-food', 'Ordering Food', 'I can order food or drink and ask prices.', 'Use polite chunks for cafes, restaurants, and checkout moments.', 35, false),
    ('a0000001-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000002', 'A1', 5, 'a1-around-town', 'Around Town', 'I can ask where something is and follow simple directions.', 'Find places and understand basic directions.', 35, false),
    ('a0000001-0000-0000-0000-000000000006', 'c0000000-0000-0000-0000-000000000002', 'A1', 6, 'a1-checkpoint', 'A1 Checkpoint', 'I can complete basic social and service interactions.', 'Review A1 foundations before moving to A2.', 40, true),
    ('a0000001-0000-0000-0000-000000000007', 'c0000000-0000-0000-0000-000000000002', 'A2', 7, 'a2-past-weekend', 'Past Weekend', 'I can describe simple completed events.', 'Talk about yesterday, weekends, and completed plans.', 35, false),
    ('a0000001-0000-0000-0000-000000000008', 'c0000000-0000-0000-0000-000000000002', 'A2', 8, 'a2-shopping', 'Shopping', 'I can ask for items, sizes, and totals.', 'Handle common store and grocery interactions.', 35, false),
    ('a0000001-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000002', 'A2', 9, 'a2-plans', 'Plans', 'I can talk about near-future plans.', 'Make plans, invite people, and talk about errands.', 35, false),
    ('a0000001-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000002', 'A2', 10, 'a2-health-feelings', 'Health And Feelings', 'I can describe how I feel and ask for help.', 'Talk about symptoms, moods, and simple needs.', 35, false),
    ('a0000001-0000-0000-0000-000000000011', 'c0000000-0000-0000-0000-000000000002', 'A2', 11, 'a2-travel-basics', 'Travel Basics', 'I can book simple travel and lodging.', 'Book rooms, ask transport questions, and handle travel basics.', 35, false),
    ('a0000001-0000-0000-0000-000000000012', 'c0000000-0000-0000-0000-000000000002', 'A2', 12, 'a2-checkpoint', 'A2 Checkpoint', 'I can handle predictable everyday tasks.', 'Review A2 before moving into longer B1 conversations.', 45, true),
    ('a0000001-0000-0000-0000-000000000013', 'c0000000-0000-0000-0000-000000000002', 'B1', 13, 'b1-stories', 'Stories', 'I can narrate past experiences with sequence.', 'Tell what happened with clear order and context.', 40, false),
    ('a0000001-0000-0000-0000-000000000014', 'c0000000-0000-0000-0000-000000000002', 'B1', 14, 'b1-opinions', 'Opinions', 'I can give opinions and reasons.', 'Explain preferences, media reactions, and simple arguments.', 40, false),
    ('a0000001-0000-0000-0000-000000000015', 'c0000000-0000-0000-0000-000000000002', 'B1', 15, 'b1-problems', 'Problems', 'I can explain a problem and request a solution.', 'Describe service issues and ask for repair or help.', 40, false),
    ('a0000001-0000-0000-0000-000000000016', 'c0000000-0000-0000-0000-000000000002', 'B1', 16, 'b1-social-plans', 'Social Plans', 'I can negotiate plans and preferences.', 'Coordinate schedules and suggest alternatives.', 40, false),
    ('a0000001-0000-0000-0000-000000000017', 'c0000000-0000-0000-0000-000000000002', 'B1', 17, 'b1-work-study', 'Work And Study', 'I can describe responsibilities and goals.', 'Discuss work, study, skills, and future goals.', 40, false),
    ('a0000001-0000-0000-0000-000000000018', 'c0000000-0000-0000-0000-000000000002', 'B1', 18, 'b1-checkpoint', 'B1 Checkpoint', 'I can sustain conversations on familiar topics.', 'Review B1 skills across longer conversations.', 50, true),
    ('a0000001-0000-0000-0000-000000000019', 'c0000000-0000-0000-0000-000000000002', 'B2', 19, 'b2-nuanced-opinions', 'Nuanced Opinions', 'I can defend a viewpoint with nuance.', 'Discuss tradeoffs and explain a position clearly.', 45, false),
    ('a0000001-0000-0000-0000-000000000020', 'c0000000-0000-0000-0000-000000000002', 'B2', 20, 'b2-hypotheticals', 'Hypotheticals', 'I can discuss imagined outcomes.', 'Talk about risks, consequences, and possibilities.', 45, false),
    ('a0000001-0000-0000-0000-000000000021', 'c0000000-0000-0000-0000-000000000002', 'B2', 21, 'b2-professional-communication', 'Professional Communication', 'I can handle formal work conversations.', 'Navigate meetings, deadlines, requests, and feedback.', 45, false),
    ('a0000001-0000-0000-0000-000000000022', 'c0000000-0000-0000-0000-000000000002', 'B2', 22, 'b2-media-culture', 'Media And Culture', 'I can summarize and react to articles.', 'Discuss news, culture, and abstract ideas.', 45, false),
    ('a0000001-0000-0000-0000-000000000023', 'c0000000-0000-0000-0000-000000000002', 'B2', 23, 'b2-conflict-repair', 'Conflict And Repair', 'I can clarify misunderstandings and negotiate.', 'Handle disagreement, apologies, and compromise.', 45, false),
    ('a0000001-0000-0000-0000-000000000024', 'c0000000-0000-0000-0000-000000000002', 'B2', 24, 'b2-checkpoint', 'B2 Checkpoint', 'I can interact with fluency in varied contexts.', 'Validate B2 readiness through integrated tasks.', 55, true)
ON CONFLICT (course_id, slug) DO UPDATE SET
    cefr_level = EXCLUDED.cefr_level,
    ordinal = EXCLUDED.ordinal,
    title = EXCLUDED.title,
    can_do_statement = EXCLUDED.can_do_statement,
    description = EXCLUDED.description,
    estimated_minutes = EXCLUDED.estimated_minutes,
    checkpoint_required = EXCLUDED.checkpoint_required;
