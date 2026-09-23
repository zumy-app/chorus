-- 08_reading_passages.sql: Reading Comprehension Passages for English Units
DELETE FROM reading_passages WHERE course_id = 'c0000000-0000-0000-0000-000000000002';

INSERT INTO reading_passages (
    course_id, unit_id, cefr_level, title, body, word_count, questions, vocabulary_ids, is_active
)
VALUES
    (
        'c0000000-0000-0000-0000-000000000002',
        'a0000001-0000-0000-0000-000000000001',
        'A1',
        'A new friend',
        'Hi, my name is Carlos. I am from Madrid, but I live in Valencia. I study computer science at the university and work in a café on weekends. I really like learning languages and meeting new people.',
        39,
        '[{"question":"Where does Carlos live?","choices":["In Valencia","In Madrid","In Seville","In Barcelona"],"correct":"In Valencia"}]'::jsonb,
        ARRAY[]::uuid[],
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002',
        'a0000001-0000-0000-0000-000000000004',
        'A1',
        'At the coffee shop',
        'Emma walks into the coffee shop near her office. She would like a large coffee with oat milk and a croissant to go. The barista asks if she wants anything else, and she says that is all. She pays by card and says thank you.',
        48,
        '[{"question":"What does Emma order?","choices":["A large coffee with oat milk","A small black coffee","A tea with milk","A sandwich"],"correct":"A large coffee with oat milk"},{"question":"How does she pay?","choices":["By card","In cash","With her phone","She doesn''t pay"],"correct":"By card"}]'::jsonb,
        ARRAY[]::uuid[],
        true
    ),
    (
        'c0000000-0000-0000-0000-000000000002',
        'a0000001-0000-0000-0000-000000000013',
        'B1',
        'The missed train',
        'Last summer I was traveling to Brighton for a job interview. I had prepared everything the night before, but my alarm didn''t go off. When I woke up, I realized I only had forty minutes to get to the station. I took a taxi, but while we were driving, there was an accident and the traffic stopped completely. I finally arrived at the platform just as my train was leaving. It turned out that the interview was rescheduled for the next day, and I got the job anyway.',
        88,
        '[{"question":"Why was the narrator late?","choices":["The alarm didn''t ring and traffic stopped","The taxi broke down","The train left early","They forgot the interview"],"correct":"The alarm didn''t ring and traffic stopped"},{"question":"What happened in the end?","choices":["The interview was moved and they got the job","They missed the interview and lost the job","They arrived on time","They took another train"],"correct":"The interview was moved and they got the job"}]'::jsonb,
        ARRAY[]::uuid[],
        true
    );
