-- 06_scenarios.sql: Interactive AI Roleplay Scenarios (English course)
INSERT INTO scenario_scripts (
    id, course_id, unit_id, slug, title, domain, cefr_level, can_do_statement,
    ai_role_name, ai_role_description, opening_line, max_turns, estimated_minutes,
    scaffold_initial, completion_criteria
)
VALUES
    (
        'd0000001-0000-0000-0000-000000000001',
        'c0000000-0000-0000-0000-000000000002',
        'a0000001-0000-0000-0000-000000000004',
        'ordering-coffee',
        'Ordering Coffee',
        'service',
        'A1',
        'I can order a drink, specify size, milk, and pay politely.',
        'Emma (Barista)',
        'Friendly barista at Central Café in London who speaks natural English with learner-friendly clarity.',
        'Hi there! Good morning, what can I get you today?',
        10,
        5,
        'guided',
        '{"required_phase_count":5,"required_intents":["greet","order_drink","specify_milk","ask_price","say_thanks"],"min_score":700}'::jsonb
    ),
    (
        'd0000001-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000002',
        'a0000001-0000-0000-0000-000000000011',
        'hotel-checkin',
        'Hotel Check-In',
        'travel',
        'A2',
        'I can check into a hotel, confirm dates, and ask about amenities.',
        'Olivia (Receptionist)',
        'Front desk clerk at the Sun Hotel in Brighton.',
        'Good afternoon, welcome to the Sun Hotel. Do you have a reservation?',
        10,
        6,
        'supported',
        '{"required_phase_count":4,"required_intents":["confirm_reservation","provide_id","ask_wifi","ask_breakfast"],"min_score":700}'::jsonb
    )
ON CONFLICT (course_id, slug) DO UPDATE SET
    title = EXCLUDED.title,
    domain = EXCLUDED.domain,
    cefr_level = EXCLUDED.cefr_level,
    can_do_statement = EXCLUDED.can_do_statement,
    ai_role_name = EXCLUDED.ai_role_name,
    ai_role_description = EXCLUDED.ai_role_description,
    opening_line = EXCLUDED.opening_line,
    max_turns = EXCLUDED.max_turns,
    estimated_minutes = EXCLUDED.estimated_minutes,
    scaffold_initial = EXCLUDED.scaffold_initial,
    completion_criteria = EXCLUDED.completion_criteria;

-- Phases for ordering-coffee
INSERT INTO scenario_phases (
    scenario_id, ordinal, title, learner_goal, required_intents, scaffold_hints, ai_follow_up, success_examples, chunk_bank
)
VALUES
    (
        'd0000001-0000-0000-0000-000000000001', 1, 'Greeting & Base Order',
        'Greet the barista and order a coffee.',
        ARRAY['greet','order_drink'],
        '["Start with Hi or Good morning","Use I would like a coffee or Can I get a coffee"]'::jsonb,
        'Sure! How would you like it, with milk or black?',
        '["Hi, I''d like a coffee with milk please.","Good morning, can I get a black coffee to go?"]'::jsonb,
        '[{"text":"I''d like a coffee","translation":"Quisiera un café"},{"text":"To go","translation":"Para llevar"},{"text":"Please","translation":"Por favor"}]'::jsonb
    ),
    (
        'd0000001-0000-0000-0000-000000000001', 2, 'Milk & Temperature',
        'Specify your milk preference or drink temperature.',
        ARRAY['specify_milk'],
        '["Say with oat milk, with skimmed milk, or black"]'::jsonb,
        'Great. Would you like any sugar or sweetener?',
        '["With oat milk, please.","Black and extra hot."]'::jsonb,
        '[{"text":"With oat milk","translation":"Con leche de avena"},{"text":"No sugar","translation":"Sin azúcar"},{"text":"Lukewarm","translation":"Templado"}]'::jsonb
    ),
    (
        'd0000001-0000-0000-0000-000000000001', 3, 'Size & Extras',
        'Choose the size or add a pastry.',
        ARRAY['specify_size'],
        '["You can say large, medium, or add a croissant"]'::jsonb,
        'Perfect, here you go. Anything else?',
        '["A large one, please.","Medium, and a croissant."]'::jsonb,
        '[{"text":"A medium one","translation":"Un mediano"},{"text":"And a croissant","translation":"Y un cruasán"},{"text":"Nothing else","translation":"Nada más"}]'::jsonb
    ),
    (
        'd0000001-0000-0000-0000-000000000001', 4, 'Price & Payment',
        'Ask how much it costs and state how you will pay.',
        ARRAY['ask_price','pay_method'],
        '["Ask How much is it? and offer card or cash"]'::jsonb,
        'That''s two fifty. You can pay by card here.',
        '["How much is it? I''ll pay by card.","How much is the total?"]'::jsonb,
        '[{"text":"How much is it?","translation":"¿Cuánto es?"},{"text":"By card","translation":"Con tarjeta"},{"text":"In cash","translation":"En efectivo"}]'::jsonb
    ),
    (
        'd0000001-0000-0000-0000-000000000001', 5, 'Farewell & Politeness',
        'Say thank you and have a good day.',
        ARRAY['say_thanks'],
        '["Say Thanks a lot, see you later or have a nice day"]'::jsonb,
        'You''re welcome! Have a great day, see you later!',
        '["Thanks a lot, have a good day.","Thank you, see you soon!"]'::jsonb,
        '[{"text":"Thanks a lot","translation":"Muchas gracias"},{"text":"Have a good day","translation":"Que tengas buen día"},{"text":"See you later","translation":"Hasta luego"}]'::jsonb
    )
ON CONFLICT (scenario_id, ordinal) DO UPDATE SET
    title = EXCLUDED.title,
    learner_goal = EXCLUDED.learner_goal,
    required_intents = EXCLUDED.required_intents,
    scaffold_hints = EXCLUDED.scaffold_hints,
    ai_follow_up = EXCLUDED.ai_follow_up,
    success_examples = EXCLUDED.success_examples,
    chunk_bank = EXCLUDED.chunk_bank;
