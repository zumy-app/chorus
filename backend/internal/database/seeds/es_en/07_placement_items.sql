-- 07_placement_items.sql: Calibrated Item Bank for Placement Engine (English course)
-- Structure: 8 receptive_vocab -> 8 grammar_production -> 4 discourse_reading.
DELETE FROM placement_items WHERE course_id = 'c0000000-0000-0000-0000-000000000002';

INSERT INTO placement_items (
    course_id, cefr_level, module, item_type, prompt, choices, correct, accept_variants, difficulty_value, is_active
)
VALUES
    -- A1 Receptive Vocab (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'A1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"The boy is five years old.","target_word":"five","context_translation":"El niño tiene ___ años."}'::jsonb,
        ARRAY['cinco','quince','cincuenta','doce'],
        'cinco', ARRAY['5']::text[], 125, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'A1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"I want a cup of tea.","target_word":"tea","context_translation":"Quiero una taza de ___."}'::jsonb,
        ARRAY['té','café','leche','agua'],
        'té', ARRAY['te']::text[], 150, true
    ),
    -- A2 Receptive Vocab (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'A2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Yesterday I bought a new shirt.","target_word":"bought","context_translation":"Ayer me ___ una camisa nueva."}'::jsonb,
        ARRAY['compré','vendí','lavé','llevé'],
        'compré', ARRAY[]::text[], 350, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'A2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"We stayed in a hotel near the airport.","target_word":"airport","context_translation":"Nos quedamos en un hotel cerca del ___."}'::jsonb,
        ARRAY['aeropuerto','estación','puerto','museo'],
        'aeropuerto', ARRAY[]::text[], 375, true
    ),
    -- B1 Receptive Vocab (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"The report concludes that investing more in research is essential.","target_word":"essential","context_translation":"El informe concluye que invertir más en investigación es ___."}'::jsonb,
        ARRAY['imprescindible','opcional','prohibido','demorado'],
        'imprescindible', ARRAY[]::text[], 575, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Despite the setbacks, the project went ahead.","target_word":"setbacks","context_translation":"A pesar de los ___, el proyecto siguió adelante."}'::jsonb,
        ARRAY['contratiempos','presupuestos','revisiones','plazos'],
        'contratiempos', ARRAY[]::text[], 600, true
    ),
    -- B2 Receptive Vocab (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"The measure had an adverse effect on the regional economy.","target_word":"adverse","context_translation":"La medida tuvo un efecto ___ en la economía regional."}'::jsonb,
        ARRAY['adverso','favorable','neutral','demorado'],
        'adverso', ARRAY[]::text[], 810, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Experts stress the need for a comprehensive approach to the problem.","target_word":"comprehensive","context_translation":"Los expertos subrayan la necesidad de un enfoque ___ del problema."}'::jsonb,
        ARRAY['integral','parcial','temporal','externo'],
        'integral', ARRAY[]::text[], 830, true
    ),
    -- A1 Grammar Production (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'A1', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"My sister _____ in a store.","verb":"to work (present simple)","translation":"Mi hermana trabaja en una tienda."}'::jsonb,
        ARRAY[]::text[],
        'works', ARRAY['Works']::text[], 180, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'A1', 'grammar_production', 'sentence_reconstruction',
        '{"instruction":"Order the words to make a correct sentence.","words":["is","Carlos","from","Madrid"]}'::jsonb,
        ARRAY['Carlos','is','from','Madrid'],
        'Carlos is from Madrid', ARRAY[]::text[], 200, true
    ),
    -- A2 Grammar Production (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'A2', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"Last year we _____ to Italy.","verb":"to travel (past simple)","translation":"El año pasado viajamos a Italia."}'::jsonb,
        ARRAY[]::text[],
        'traveled', ARRAY['Travelled']::text[], 400, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'A2', 'grammar_production', 'sentence_reconstruction',
        '{"instruction":"Order the words to make a correct sentence.","words":["have","I","booked","a","room"]}'::jsonb,
        ARRAY['I','have','booked','a','room'],
        'I have booked a room', ARRAY['I''ve booked a room']::text[], 420, true
    ),
    -- B1 Grammar Production (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"When I was a boy, I always _____ to the park on Sundays.","verb":"to go (past continuous)","translation":"Cuando era niño, siempre iba al parque los domingos."}'::jsonb,
        ARRAY[]::text[],
        'was going', ARRAY['Was going']::text[], 600, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"If I had more time, I _____ a cooking class.","verb":"would + take (second conditional)","translation":"Si tuviera más tiempo, haría un curso de cocina."}'::jsonb,
        ARRAY[]::text[],
        'would take', ARRAY['Would take','would have taken']::text[], 625, true
    ),
    -- B2 Grammar Production (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"They said the meeting _____ until Monday.","verb":"to postpone (reported speech, past perfect)","translation":"Dijeron que la reunión se había aplazado hasta el lunes."}'::jsonb,
        ARRAY[]::text[],
        'had been postponed', ARRAY['Had been postponed']::text[], 800, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'grammar_production', 'sentence_reconstruction',
        '{"instruction":"Order the words to make a correct sentence.","words":["had","if","studied","would","I","I","passed","have"]}'::jsonb,
        ARRAY['if','I','had','studied','I','would','have','passed'],
        'if I had studied I would have passed', ARRAY['If I had studied, I would have passed.']::text[], 860, true
    ),
    -- B1 Discourse Reading (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'discourse_reading', 'passage_mcq',
        '{"passage":"Rural tourism has gained popularity over the last decade. Many families prefer to spend their vacations in small villages to disconnect from urban stress and support the local economy.","question":"Why do many families choose rural tourism?"}'::jsonb,
        ARRAY['To disconnect from stress and support local economies','Because there are no hotels in cities','Because they dislike traveling','To visit large museums'],
        'To disconnect from stress and support local economies', ARRAY[]::text[], 650, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B1', 'discourse_reading', 'passage_mcq',
        '{"passage":"Although remote work offers flexibility, many professionals say they find it hard to separate personal life from work when home is also the office.","question":"What difficulty do remote workers mention?"}'::jsonb,
        ARRAY['Separating personal life from work','Finding a place with internet','Getting permission from the company','Saving on commuting'],
        'Separating personal life from work', ARRAY[]::text[], 675, true
    ),
    -- B2 Discourse Reading (2)
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'discourse_reading', 'passage_mcq',
        '{"passage":"Despite legislative advances in the energy transition, experts warn that without substantial investment in storage and distribution infrastructure, the decarbonization targets set for 2030 will be unattainable.","question":"What condition do experts consider essential?"}'::jsonb,
        ARRAY['Investment in storage and distribution','Passing more laws without a budget','Eliminating all renewable energy','Cutting consumption by 90%'],
        'Investment in storage and distribution', ARRAY[]::text[], 850, true
    ),
    (
        'c0000000-0000-0000-0000-000000000002', 'B2', 'discourse_reading', 'passage_mcq',
        '{"passage":"Artificial intelligence is transforming the labor market unevenly: while some sectors absorb automation without net job losses, others watch certain tasks disappear without equivalent alternatives being created for the affected workers.","question":"What is the main idea of the text?"}'::jsonb,
        ARRAY['Automation affects labor sectors unevenly','Artificial intelligence will eliminate all jobs','The labor market is unaffected by technology','Workers reject artificial intelligence'],
        'Automation affects labor sectors unevenly', ARRAY[]::text[], 880, true
    );
