-- 09_real_talk_prompts.sql: Colloquial Real-Talk Prompts for Chat and Practice
INSERT INTO real_talk_prompts (
    course_id, cefr_level, category, prompt_for_learner, target_phrase, why_useful, follow_up_chunks, is_active
)
VALUES
    (
        'c0000000-0000-0000-0000-000000000001', 'A1', 'conversation_starter',
        'Ask someone how their day is going in a natural, friendly way.',
        '¿Qué tal el día?',
        'Much more natural and conversational in Spain and Latin America than the textbook "¿Cómo estás?".',
        '["Todo bien, ¿y tú?","Bastante ocupado","Tranquilo, como siempre"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'A1', 'opinion_phrase',
        'Express that something seems great to you.',
        '¡Me parece genial!',
        'Essential enthusiastic reaction used daily by native speakers to show agreement and support.',
        '["¿Quedamos a las siete?","Buena idea","Perfecto entonces"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'A2', 'topic_injector',
        'Transition the conversation to talk about plans for the weekend.',
        'Por cierto, ¿qué planes tienes para el fin de semana?',
        '"Por cierto" (by the way) is the quintessential smooth conversational pivot.',
        '["Pues nada especial","Voy a salir al campo","A ver si nos vemos"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'opinion_phrase',
        'Politely state your perspective when offering a different opinion.',
        'Desde mi punto de vista, depende bastante de la situación.',
        'Provides professional nuance without sounding combative.',
        '["Claro, tiene sentido","No lo había pensado así","Puede ser"]'::jsonb,
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'topic_injector',
        'Bring up an interesting article or news item in a discussion.',
        'El otro día leí un artículo interesantísimo sobre este tema...',
        'Native formula for introducing depth and outside references into a debate.',
        '["¿Y qué decía?","Me lo tienes que pasar","Justo lo estaba comentando ayer"]'::jsonb,
        true
    )
ON CONFLICT DO NOTHING;
