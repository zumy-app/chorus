-- 07_placement_items.sql: Calibrated Item Bank for Placement Engine
INSERT INTO placement_items (
    course_id, cefr_level, module, item_type, prompt, choices, correct, accept_variants, difficulty_value, is_active
)
VALUES
    -- A1 Receptive Vocab
    (
        'c0000000-0000-0000-0000-000000000001', 'A1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"El niño tiene cinco años.","target_word":"cinco","context_translation":"The child is ___ years old."}'::jsonb,
        ARRAY['five','fifteen','yesterday','many'],
        'five', ARRAY['5']::text[], 125, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'A1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Quiero una taza de té.","target_word":"té","context_translation":"I want a cup of ___."}'::jsonb,
        ARRAY['tea','water','milk','coffee'],
        'tea', ARRAY[]::text[], 150, true
    ),
    -- A1 Grammar Production
    (
        'c0000000-0000-0000-0000-000000000001', 'A1', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"Mi hermana _____ en una tienda.","verb":"trabajar","translation":"My sister works in a store."}'::jsonb,
        ARRAY[]::text[],
        'trabaja', ARRAY['Trabaja']::text[], 180, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'A1', 'grammar_production', 'sentence_reconstruction',
        '{"instruction":"Order the words to make a correct sentence.","words":["de","Madrid","es","Carlos"]}'::jsonb,
        ARRAY['Carlos','es','de','Madrid'],
        'Carlos es de Madrid', ARRAY[]::text[], 200, true
    ),
    -- A2 Receptive Vocab
    (
        'c0000000-0000-0000-0000-000000000001', 'A2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Ayer compré una camisa nueva.","target_word":"compré","context_translation":"Yesterday I ___ a new shirt."}'::jsonb,
        ARRAY['bought','sold','wore','washed'],
        'bought', ARRAY[]::text[], 350, true
    ),
    -- A2 Grammar Production
    (
        'c0000000-0000-0000-0000-000000000001', 'A2', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"El año pasado nosotros _____ a Italia.","verb":"viajar (pretérito)","translation":"Last year we traveled to Italy."}'::jsonb,
        ARRAY[]::text[],
        'viajamos', ARRAY['Viajamos']::text[], 400, true
    ),
    -- B1 Receptive Vocab & Grammar
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"Cuando era niño, siempre _____ al parque los domingos.","verb":"ir (imperfecto)","translation":"When I was a boy, I always went to the park on Sundays."}'::jsonb,
        ARRAY[]::text[],
        'iba', ARRAY['Iba']::text[], 600, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'discourse_reading', 'passage_mcq',
        '{"passage":"El turismo rural ha ganado popularidad en España durante la última década. Muchas familias prefieren pasar sus vacaciones en pequeños pueblos para desconectar del estrés urbano y apoyar la economía local.","question":"¿Por qué eligen muchas familias el turismo rural?"}'::jsonb,
        ARRAY['Para desconectar del estrés y apoyar economías locales','Porque no hay hoteles en las ciudades','Porque no les gusta viajar','Para visitar grandes museos'],
        'Para desconectar del estrés y apoyar economías locales', ARRAY[]::text[], 650, true
    ),
    -- B2 Discourse & Grammar
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"Dudo mucho que ellos _____ la verdad en este momento.","verb":"saber (subjuntivo presente)","translation":"I strongly doubt that they know the truth right now."}'::jsonb,
        ARRAY[]::text[],
        'sepan', ARRAY['Sepan']::text[], 800, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'discourse_reading', 'passage_mcq',
        '{"passage":"A pesar de los avances legislativos en materia de transición energética, los expertos advierten que sin una inversión sustancial en infraestructuras de almacenamiento y distribución, será inviable alcanzar los objetivos de descarbonización fijados para 2030.","question":"¿Qué condición señalan los expertos como imprescindible?"}'::jsonb,
        ARRAY['Inversión en almacenamiento y distribución','Aprobar más leyes sin presupuesto','Eliminar todas las energías renovables','Reducir el consumo un 90%'],
        'Inversión en almacenamiento y distribución', ARRAY[]::text[], 850, true
    );
