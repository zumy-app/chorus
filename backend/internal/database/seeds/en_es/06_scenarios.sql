-- 06_scenarios.sql: Interactive AI Roleplay Scenarios
INSERT INTO scenario_scripts (
    id, course_id, unit_id, slug, title, domain, cefr_level, can_do_statement,
    ai_role_name, ai_role_description, opening_line, max_turns, estimated_minutes,
    scaffold_initial, completion_criteria
)
VALUES
    (
        's0000000-0000-0000-0000-000000000001',
        'c0000000-0000-0000-0000-000000000001',
        'u0000000-0000-0000-0000-000000000004',
        'ordering-coffee',
        'Ordering Coffee',
        'service',
        'A1',
        'I can order a drink, specify size, milk, and pay politely.',
        'Mateo (Barista)',
        'Friendly barista at Café Central in Madrid who speaks natural Spanish with learner-friendly clarity.',
        '¡Hola! Buenos días, ¿qué te pongo hoy?',
        10,
        5,
        'guided',
        '{"required_phase_count":5,"required_intents":["greet","order_drink","specify_milk","ask_price","say_thanks"],"min_score":700}'::jsonb
    ),
    (
        's0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000001',
        'u0000000-0000-0000-0000-000000000011',
        'hotel-checkin',
        'Hotel Check-In',
        'travel',
        'A2',
        'I can check into a hotel, confirm dates, and ask about amenities.',
        'Valeria (Recepcionista)',
        'Front desk clerk at Hotel Sol in Valencia.',
        'Buenas tardes, bienvenido al Hotel Sol. ¿Tiene una reserva?',
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
        's0000000-0000-0000-0000-000000000001', 1, 'Greeting & Base Order',
        'Greet the barista and order a coffee.',
        ARRAY['greet','order_drink'],
        '["Start with Hola or Buenos días","Use Quisiera un café or Para mí un café"]'::jsonb,
        '¿Cómo lo quieres, con leche o solo?',
        '["Hola, quisiera un café con leche por favor.","Buenos días, un café solo para llevar."]'::jsonb,
        '[{"text":"Quisiera un café","translation":"I would like a coffee"},{"text":"Para llevar","translation":"To go"},{"text":"Por favor","translation":"Please"}]'::jsonb
    ),
    (
        's0000000-0000-0000-0000-000000000001', 2, 'Milk & Temperature',
        'Specify your milk preference or drink temperature.',
        ARRAY['specify_milk'],
        '["Mention con leche de avena, con leche desnatada, or solo"]'::jsonb,
        'Muy bien. ¿Le pongo azúcar o sacarina?',
        '["Con leche de avena, por favor.","Solo y bien caliente."]'::jsonb,
        '[{"text":"Con leche de avena","translation":"With oat milk"},{"text":"Sin azúcar","translation":"Without sugar"},{"text":"Templado","translation":"Lukewarm"}]'::jsonb
    ),
    (
        's0000000-0000-0000-0000-000000000001', 3, 'Size & Extras',
        'Choose the size or add a pastry.',
        ARRAY['specify_size'],
        '["You can say grande, mediano, or add un cruasán"]'::jsonb,
        'Perfecto, aquí tienes. ¿Algo más?',
        '["Tamaño grande, por favor.","Mediano, y un cruasán."]'::jsonb,
        '[{"text":"Tamaño mediano","translation":"Medium size"},{"text":"Y un cruasán","translation":"And a croissant"},{"text":"Nada más","translation":"Nothing else"}]'::jsonb
    ),
    (
        's0000000-0000-0000-0000-000000000001', 4, 'Price & Payment',
        'Ask how much it costs and state how you will pay.',
        ARRAY['ask_price','pay_method'],
        '["Ask ¿Cuánto es? and offer tarjeta or efectivo"]'::jsonb,
        'Son dos con cincuenta. Puedes pagar con tarjeta aquí.',
        '["¿Cuánto cuesta? Pago con tarjeta.","¿Cuánto es en total?"]'::jsonb,
        '[{"text":"¿Cuánto es?","translation":"How much is it?"},{"text":"Con tarjeta","translation":"With card"},{"text":"En efectivo","translation":"In cash"}]'::jsonb
    ),
    (
        's0000000-0000-0000-0000-000000000001', 5, 'Farewell & Politeness',
        'Say thank you and have a good day.',
        ARRAY['say_thanks'],
        '["Say Muchas gracias, hasta luego or que tengas buen día"]'::jsonb,
        '¡A ti! Que tengas un excelente día, ¡hasta luego!',
        '["Muchas gracias, buen día.","¡Gracias, hasta pronto!"]'::jsonb,
        '[{"text":"Muchas gracias","translation":"Thank you very much"},{"text":"Buen día","translation":"Good day"},{"text":"Hasta luego","translation":"See you later"}]'::jsonb
    )
ON CONFLICT (scenario_id, ordinal) DO UPDATE SET
    title = EXCLUDED.title,
    learner_goal = EXCLUDED.learner_goal,
    required_intents = EXCLUDED.required_intents,
    scaffold_hints = EXCLUDED.scaffold_hints,
    ai_follow_up = EXCLUDED.ai_follow_up,
    success_examples = EXCLUDED.success_examples,
    chunk_bank = EXCLUDED.chunk_bank;
