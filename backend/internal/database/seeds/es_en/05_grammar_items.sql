-- 05_grammar_items.sql: Interactive drill items for English Grammar Points
INSERT INTO grammar_items (
    grammar_point_id, ordinal, item_type, prompt, sentence_with_blank, choices, correct, accept_variants, note, is_active
)
VALUES
    -- Point 1: to-be-subject-pronouns
    ('b0000001-0000-0000-0000-000000000001', 1, 'cloze', 'Complete the sentence with the correct form of to be.', 'Pedro _____ from Madrid.', ARRAY[]::text[], 'is', ARRAY['Is']::text[], 'Pedro is 3rd person singular, so use "is".', true),
    ('b0000001-0000-0000-0000-000000000001', 2, 'mcq', 'Select the correct pronoun form.', 'Where _____ you from?', ARRAY['are','is','am','be'], 'are', ARRAY[]::text[], '"you" pairs with "are".', true),
    ('b0000001-0000-0000-0000-000000000001', 3, 'reconstruction', 'Reconstruct the sentence in the correct order.', '', ARRAY['I','a','am','student'], 'I am a student', ARRAY['I''m a student']::text[], 'Subject + verb + article + noun. English needs the article.', true),
    ('b0000001-0000-0000-0000-000000000001', 4, 'cloze', 'Fill in the blank with to be.', 'We _____ friends since childhood.', ARRAY[]::text[], 'are', ARRAY['Are']::text[], '"We" takes "are".', true),

    -- Point 2: questions-with-do
    ('b0000001-0000-0000-0000-000000000002', 1, 'mcq', 'Which question is correct?', '_____ you speak English?', ARRAY['Do','Are','Is','Does'], 'Do', ARRAY[]::text[], 'Yes/no questions with "you" use Do.', true),
    ('b0000001-0000-0000-0000-000000000002', 2, 'cloze', 'Complete the question asking someone''s name.', 'What _____ your name?', ARRAY[]::text[], 'is', ARRAY['Is']::text[], 'What is your name? — the question word + is.', true),
    ('b0000001-0000-0000-0000-000000000002', 3, 'mcq', 'Which question asks for a place?', '_____ do you live?', ARRAY['Where','When','What','Who'], 'Where', ARRAY[]::text[], 'Where asks "in what place".', true),
    ('b0000001-0000-0000-0000-000000000002', 4, 'reconstruction', 'Arrange into a question.', '', ARRAY['you','Do','English','speak','well'], 'Do you speak English well', ARRAY['Do you speak English well?']::text[], 'Do + subject + verb + rest.', true),

    -- Point 3: present-simple-third-person
    ('b0000001-0000-0000-0000-000000000003', 1, 'cloze', 'Complete with the present simple of work.', 'She _____ in a store.', ARRAY[]::text[], 'works', ARRAY['Works']::text[], 'Third person singular adds -s.', true),
    ('b0000001-0000-0000-0000-000000000003', 2, 'mcq', 'Choose the correct sentence.', 'Maria _____ in the center of Barcelona.', ARRAY['lives','live','living','to live'], 'lives', ARRAY[]::text[], 'She + lives (third person -s).', true),
    ('b0000001-0000-0000-0000-000000000003', 3, 'cloze', 'Complete with the present simple of have.', 'I _____ breakfast at eight.', ARRAY[]::text[], 'have', ARRAY['Have']::text[], 'I/you/we/they take the base form.', true),
    ('b0000001-0000-0000-0000-000000000003', 4, 'reconstruction', 'Reconstruct the daily routine sentence.', '', ARRAY['at','seven','I','up','get'], 'I get up at seven', ARRAY[]::text[], 'Subject + verb + time phrase.', true),

    -- Point 4: would-like-articles
    ('b0000001-0000-0000-0000-000000000004', 1, 'cloze', 'Complete with the polite ordering phrase.', '_____ a black coffee, please.', ARRAY[]::text[], 'I''d like', ARRAY['I would like','Id like']::text[], 'I''d like is the polite request formula.', true),
    ('b0000001-0000-0000-0000-000000000004', 2, 'mcq', 'Select the correct article.', '_____ orange juice is very cold.', ARRAY['An','A','The some','Some'], 'An', ARRAY[]::text[], 'An before vowel sounds: an orange.', true),
    ('b0000001-0000-0000-0000-000000000004', 3, 'mcq', 'Select the polite request phrase.', '_____ you bring the check, please?', ARRAY['Could','Want','Bring','Must'], 'Could', ARRAY[]::text[], 'Could you...? brings formal politeness.', true),
    ('b0000001-0000-0000-0000-000000000004', 4, 'reconstruction', 'Form an order.', '', ARRAY['a','please','I''d','coffee','like'], 'I''d like a coffee please', ARRAY['I would like a coffee please']::text[], 'Polite request word order.', true)
,
    -- Point 5: there-is-prepositions
    ('b0000001-0000-0000-0000-000000000005', 1, 'cloze', 'Complete with there is or there are.', '_____ two banks near the hotel.', ARRAY[]::text[], 'There are', ARRAY['there are']::text[], 'Plural existence uses there are.', true),
    ('b0000001-0000-0000-0000-000000000005', 2, 'mcq', 'Select the correct preposition.', 'The museum is _____ the park.', ARRAY['next to','next','across','in front'], 'next to', ARRAY[]::text[], 'next to = al lado de.', true),
    ('b0000001-0000-0000-0000-000000000005', 3, 'cloze', 'Complete the direction.', 'Turn _____ at the corner.', ARRAY[]::text[], 'left', ARRAY['Left']::text[], 'left = izquierda, right = derecha.', true),

    -- Point 6: past-simple-regular-irregular
    ('b0000001-0000-0000-0000-000000000006', 1, 'cloze', 'Complete in the past simple.', 'Yesterday I _____ to the movies with my friends.', ARRAY[]::text[], 'went', ARRAY['Went']::text[], 'go is irregular: go -> went.', true),
    ('b0000001-0000-0000-0000-000000000006', 2, 'mcq', 'Select the correct question.', '_____ you see the movie last night?', ARRAY['Did','Do','Was','Were'], 'Did', ARRAY[]::text[], 'Past questions use did + base verb.', true),
    ('b0000001-0000-0000-0000-000000000006', 3, 'cloze', 'Complete in the past simple.', 'We _____ an amazing paella.', ARRAY[]::text[], 'ate', ARRAY['Ate']::text[], 'eat is irregular: eat -> ate.', true),

    -- Point 7: going-to-have-to
    ('b0000001-0000-0000-0000-000000000007', 1, 'cloze', 'Complete with going to.', 'I''m _____ to travel to Mexico next month.', ARRAY[]::text[], 'going', ARRAY['Going']::text[], 'be going to + verb for plans.', true),
    ('b0000001-0000-0000-0000-000000000007', 2, 'mcq', 'Select the correct obligation phrase.', 'I _____ work tomorrow all day.', ARRAY['have to','have at','must to','should to'], 'have to', ARRAY[]::text[], 'have to + base verb = tener que.', true),

    -- Point 8: past-simple-vs-past-continuous
    ('b0000001-0000-0000-0000-000000000008', 1, 'mcq', 'Select past simple or past continuous.', 'When I got to the beach, _____ sunny.', ARRAY['it was','it is','it been','it would be'], 'it was', ARRAY[]::text[], 'Background states in a story take was.', true),
    ('b0000001-0000-0000-0000-000000000008', 2, 'cloze', 'Complete with the past continuous.', 'While I _____ , the phone rang.', ARRAY[]::text[], 'was studying', ARRAY['Was studying']::text[], 'While + past continuous for the background action.', true),

    -- Point 9: conditionals-second
    ('b0000001-0000-0000-0000-000000000009', 1, 'cloze', 'Complete the second conditional.', 'If I had more time, I _____ take a cooking class.', ARRAY[]::text[], 'would', ARRAY['Would']::text[], 'If + past simple, would + verb.', true),
    ('b0000001-0000-0000-0000-000000000009', 2, 'mcq', 'Select the correct clause.', 'If I _____ you, I would accept the offer.', ARRAY['were','am','will be','would be'], 'were', ARRAY[]::text[], 'Formal hypotheticals use were for all persons.', true),

    -- Point 10: reported-speech-passive
    ('b0000001-0000-0000-0000-000000000010', 1, 'cloze', 'Complete in reported speech.', 'They said the meeting _____ postponed.', ARRAY[]::text[], 'was', ARRAY['Was']::text[], 'Reported speech shifts is -> was.', true),
    ('b0000001-0000-0000-0000-000000000010', 2, 'mcq', 'Select the correct passive.', 'The bridge _____ in 1995.', ARRAY['was built','built','was build','is built'], 'was built', ARRAY[]::text[], 'Passive: be + past participle.', true)ON CONFLICT (grammar_point_id, ordinal) DO UPDATE SET
    item_type = EXCLUDED.item_type,
    prompt = EXCLUDED.prompt,
    sentence_with_blank = EXCLUDED.sentence_with_blank,
    choices = EXCLUDED.choices,
    correct = EXCLUDED.correct,
    accept_variants = EXCLUDED.accept_variants,
    note = EXCLUDED.note,
    is_active = EXCLUDED.is_active;
