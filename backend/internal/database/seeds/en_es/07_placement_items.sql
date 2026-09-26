-- 07_placement_items.sql: Calibrated Item Bank for Placement Engine (V3)
-- Deploy-synced content: the bank is the single source of truth for this course,
-- so re-running a modified file replaces the previous bank rows.
-- Structure: 8 receptive_vocab -> 8 grammar_production -> 4 discourse_reading.
DELETE FROM placement_items WHERE course_id = 'c0000000-0000-0000-0000-000000000001';

INSERT INTO placement_items (
    course_id, cefr_level, module, item_type, prompt, choices, correct, accept_variants, difficulty_value, is_active
)
VALUES
    -- A1 Receptive Vocab (2)
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
    -- A2 Receptive Vocab (2)
    (
        'c0000000-0000-0000-0000-000000000001', 'A2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Ayer compré una camisa nueva.","target_word":"compré","context_translation":"Yesterday I ___ a new shirt."}'::jsonb,
        ARRAY['bought','sold','wore','washed'],
        'bought', ARRAY[]::text[], 350, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'A2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Nos quedamos en un hotel cerca del aeropuerto.","target_word":"aeropuerto","context_translation":"We stayed in a hotel near the ___."}'::jsonb,
        ARRAY['airport','station','harbor','museum'],
        'airport', ARRAY[]::text[], 375, true
    ),
    -- B1 Receptive Vocab (2)
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"El informe concluye que es imprescindible invertir más en investigación.","target_word":"imprescindible","context_translation":"The report concludes that it is ___ to invest more in research."}'::jsonb,
        ARRAY['essential','optional','prohibited','delayed'],
        'essential', ARRAY[]::text[], 575, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"A pesar de los contratiempos, el proyecto siguió adelante.","target_word":"contratiempos","context_translation":"Despite the ___, the project went ahead."}'::jsonb,
        ARRAY['setbacks','budgets','revisions','deadlines'],
        'setbacks', ARRAY[]::text[], 600, true
    ),
    -- B2 Receptive Vocab (2)
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"La medida tuvo un efecto adverso en la economía regional.","target_word":"adverso","context_translation":"The measure had an ___ effect on the regional economy."}'::jsonb,
        ARRAY['adverse','favorable','neutral','delayed'],
        'adverse', ARRAY[]::text[], 810, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'receptive_vocab', 'L2_to_L1_context',
        '{"sentence":"Los expertos subrayan la necesidad de un enfoque integral del problema.","target_word":"integral","context_translation":"Experts stress the need for a(n) ___ approach to the problem."}'::jsonb,
        ARRAY['comprehensive','partial','temporary','external'],
        'comprehensive', ARRAY[]::text[], 830, true
    ),
    -- A1 Grammar Production (2)
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
    -- A2 Grammar Production (2)
    (
        'c0000000-0000-0000-0000-000000000001', 'A2', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"El año pasado nosotros _____ a Italia.","verb":"viajar (pretérito)","translation":"Last year we traveled to Italy."}'::jsonb,
        ARRAY[]::text[],
        'viajamos', ARRAY['Viajamos']::text[], 400, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'A2', 'grammar_production', 'sentence_reconstruction',
        '{"instruction":"Order the words to make a correct sentence.","words":["una","reservado","he","habitación"]}'::jsonb,
        ARRAY['he','reservado','una','habitación'],
        'he reservado una habitación', ARRAY['He reservado una habitación']::text[], 420, true
    ),
    -- B1 Grammar Production (2)
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"Cuando era niño, siempre _____ al parque los domingos.","verb":"ir (imperfecto)","translation":"When I was a boy, I always went to the park on Sundays."}'::jsonb,
        ARRAY[]::text[],
        'iba', ARRAY['Iba']::text[], 600, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"Si tuviera más tiempo, _____ un curso de cocina.","verb":"hacer (condicional)","translation":"If I had more time, I would take a cooking class."}'::jsonb,
        ARRAY[]::text[],
        'haría', ARRAY['Haría']::text[], 625, true
    ),
    -- B2 Grammar Production (2)
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'grammar_production', 'gap_fill_type',
        '{"sentence_with_blank":"Dudo mucho que ellos _____ la verdad en este momento.","verb":"saber (subjuntivo presente)","translation":"I strongly doubt that they know the truth right now."}'::jsonb,
        ARRAY[]::text[],
        'sepan', ARRAY['Sepan']::text[], 800, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'grammar_production', 'sentence_reconstruction',
        '{"instruction":"Order the words to make a correct sentence.","words":["hubiera","si","estudiado","habría","más","aprobado"]}'::jsonb,
        ARRAY['si','hubiera','estudiado','habría','aprobado','más'],
        'si hubiera estudiado habría aprobado', ARRAY['Si hubiera estudiado, habría aprobado.']::text[], 860, true
    ),
    -- B1 Discourse Reading (1)
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'discourse_reading', 'passage_mcq',
        '{"passage":"El turismo rural ha ganado popularidad en España durante la última década. Muchas familias prefieren pasar sus vacaciones en pequeños pueblos para desconectar del estrés urbano y apoyar la economía local.","question":"¿Por qué eligen muchas familias el turismo rural?"}'::jsonb,
        ARRAY['Para desconectar del estrés y apoyar economías locales','Porque no hay hoteles en las ciudades','Porque no les gusta viajar','Para visitar grandes museos'],
        'Para desconectar del estrés y apoyar economías locales', ARRAY[]::text[], 650, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B1', 'discourse_reading', 'passage_mcq',
        '{"passage":"Aunque el teletrabajo ofrece flexibilidad, muchos profesionales afirman que les resulta difícil separar la vida personal del trabajo cuando la casa es también la oficina.","question":"¿Qué dificultad mencionan los profesionales del teletrabajo?"}'::jsonb,
        ARRAY['Separar la vida personal del trabajo','Encontrar un espacio con internet','Conseguir permiso de la empresa','Ahorrar en transporte'],
        'Separar la vida personal del trabajo', ARRAY[]::text[], 675, true
    ),
    -- B2 Discourse Reading (2)
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'discourse_reading', 'passage_mcq',
        '{"passage":"A pesar de los avances legislativos en materia de transición energética, los expertos advierten que sin una inversión sustancial en infraestructuras de almacenamiento y distribución, será inviable alcanzar los objetivos de descarbonización fijados para 2030.","question":"¿Qué condición señalan los expertos como imprescindible?"}'::jsonb,
        ARRAY['Inversión en almacenamiento y distribución','Aprobar más leyes sin presupuesto','Eliminar todas las energías renovables','Reducir el consumo un 90%'],
        'Inversión en almacenamiento y distribución', ARRAY[]::text[], 850, true
    ),
    (
        'c0000000-0000-0000-0000-000000000001', 'B2', 'discourse_reading', 'passage_mcq',
        '{"passage":"La inteligencia artificial está transformando el mercado laboral de forma desigual: mientras algunos sectores absorben la automatización sin pérdidas netas de empleo, otros ven cómo ciertas tareas desaparecen sin que se creen alternativas equivalentes para los trabajadores afectados.","question":"¿Cuál es la idea principal del texto?"}'::jsonb,
        ARRAY['La automatización afecta de forma desigual a los sectores laborales','La inteligencia artificial eliminará todos los empleos','El mercado laboral no se ve afectado por la tecnología','Los trabajadores rechazan la inteligencia artificial'],
        'La automatización afecta de forma desigual a los sectores laborales', ARRAY[]::text[], 880, true
    );
