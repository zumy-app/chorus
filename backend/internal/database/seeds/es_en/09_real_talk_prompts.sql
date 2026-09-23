-- 09_real_talk_prompts.sql: Colloquial Real-Talk Prompts (English course)
DELETE FROM real_talk_prompts WHERE course_id = 'c0000000-0000-0000-0000-000000000002';

INSERT INTO real_talk_prompts (
    course_id, cefr_level, category, prompt_for_learner, target_phrase, why_useful, follow_up_chunks, is_active
)
VALUES
    (
        'c0000000-0000-0000-0000-000000000002', 'A1', 'conversation_starter',
        'Ask someone how their day is going in a natural, friendly way.',
        'How''s your day going?',
        'Much more natural and conversational than the textbook "How are you?".',
        '["Pretty good, you?","Quite busy","Chill, as always"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'A1', 'opinion_phrase',
        'Express that something seems great to you.',
        'That sounds awesome!',
        'Essential enthusiastic reaction used daily by native speakers to show agreement and support.',
        '["Shall we meet at seven?","Good idea","Perfect then"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'A2', 'topic_injector',
        'Transition the conversation to talk about plans for the weekend.',
        'By the way, any plans for the weekend?',
        '"By the way" (por cierto) is the quintessential smooth conversational pivot.',
        '["Nothing special, really","I''m going to the countryside","We should hang out"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'A2', 'opinion_phrase',
        'Politely disagree before giving your point of view.',
        'I see your point, but I''m not sure I agree.',
        'Softens disagreement so conversations stay friendly and open.',
        '["That makes sense","I hadn''t thought of it that way","Fair enough"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'topic_injector',
        'Bring up a recent personal experience in a discussion.',
        'Funny story: the same thing happened to me last week.',
        'Native formula for weaving personal stories into conversations.',
        '["No way! What happened?","You''re kidding","Tell me more"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'conversation_starter',
        'Ask for someone''s honest take on a decision you are making.',
        'Can I pick your brain about something?',
        '"Pick your brain" invites thoughtful advice without pressure.',
        '["Of course, what''s up?","Sure, shoot","Go ahead"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'opinion_phrase',
        'Politely state your perspective when offering a different opinion.',
        'From my point of view, it really depends on the situation.',
        'Provides professional nuance without sounding combative.',
        '["That makes sense","I hadn''t thought about it that way","Fair enough"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'topic_injector',
        'Bring up an interesting article or news item in a discussion.',
        'I read a really interesting article about this the other day...',
        'Native formula for introducing depth and outside references into a debate.',
        '["What did it say?","You have to send it to me","I was just talking about that"]'::jsonb,
        true
    );
