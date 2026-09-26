-- 08_reading_passages.sql: Reading Comprehension Passages for Units
DELETE FROM reading_passages WHERE course_id = 'c0000000-0000-0000-0000-000000000001';

INSERT INTO reading_passages (
    course_id, unit_id, cefr_level, title, body, word_count, questions, vocabulary_ids, is_active
)
VALUES
    (
        'c0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000001',
        'A1',
        'Un nuevo amigo',
        'Hola, me llamo Carlos. Soy de Madrid, pero vivo en Valencia. Estudio informática en la universidad y trabajo en una cafetería los fines de semana. Me gusta mucho aprender idiomas y conocer gente nueva.',
        37,
        '[{"question":"¿Dónde vive Carlos?","choices":["En Valencia","En Madrid","En Sevilla","En Barcelona"],"correct":"En Valencia"}]'::jsonb,
        ARRAY[]::uuid[],
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000004',
        'A1',
        'En la cafetería',
        'Todos los días a las diez de la mañana voy a la cafetería de la esquina. Siempre pido un café con leche y una tostada con tomate y aceite de oliva. Los camareros son muy amables y el servicio es rápido.',
        42,
        '[{"question":"¿Qué desayuna el autor?","choices":["Café con leche y tostada","Té con galletas","Zumo de naranja","Chocolate caliente"],"correct":"Café con leche y tostada"}]'::jsonb,
        ARRAY[]::uuid[],
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000007',
        'A2',
        'El fin de semana en la sierra',
        'El sábado pasado fui con mis amigos a la sierra de Guadarrama. Salimos temprano por la mañana y caminamos durante cuatro horas por el bosque. Hizo un tiempo estupendo, con sol y una brisa fresca. Al final del día comimos en un mesón tradicional.',
        46,
        '[{"question":"¿Qué hicieron el sábado?","choices":["Caminaron por el bosque en la sierra","Fueron a la playa","Se quedaron en casa","Visitaron un museo de arte"],"correct":"Caminaron por el bosque en la sierra"}]'::jsonb,
        ARRAY[]::uuid[],
        true
    );
