-- 05_grammar_items.sql: Interactive drill items for Grammar Points
INSERT INTO grammar_items (
    grammar_point_id, ordinal, item_type, prompt, sentence_with_blank, choices, correct, accept_variants, note, is_active
)
VALUES
    -- Point 1: ser-personal-pronouns
    ('g0000000-0000-0000-0000-000000000001', 1, 'cloze', 'Complete the sentence with the correct form of ser.', 'Pedro _____ de Madrid.', ARRAY[]::text[], 'es', ARRAY['Es'], 'Pedro is 3rd person singular (él), so use "es".', true),
    ('g0000000-0000-0000-0000-000000000001', 2, 'mcq', 'Select the correct pronoun form.', '¿De dónde _____ vosotros?', ARRAY['sois','somos','son','eres'], 'sois', ARRAY[]::text[], 'Vosotros pairs with "sois".', true),
    ('g0000000-0000-0000-0000-000000000001', 3, 'reconstruction', 'Reconstruct the sentence in the correct order.', '', ARRAY['Yo','estudiante','español','de','soy'], 'Yo soy estudiante de español', ARRAY['Soy estudiante de español'], 'Subject + verb + predicate.', true),
    ('g0000000-0000-0000-0000-000000000001', 4, 'cloze', 'Fill in the blank with ser.', 'Nosotros _____ amigos desde la infancia.', ARRAY[]::text[], 'somos', ARRAY['Somos'], 'Nosotros takes "somos".', true),

    -- Point 2: yes-no-questions-question-words
    ('g0000000-0000-0000-0000-000000000002', 1, 'mcq', 'Which word asks for location?', '¿_____ está la estación de tren?', ARRAY['Dónde','Cuándo','Cómo','Quién'], 'Dónde', ARRAY['dónde'], 'Dónde asks "where".', true),
    ('g0000000-0000-0000-0000-000000000002', 2, 'cloze', 'Complete the question for asking someone''s name.', '¿_____ te llamas?', ARRAY[]::text[], 'Cómo', ARRAY['cómo','Como'], '¿Cómo te llamas? literally means "How do you call yourself?".', true),
    ('g0000000-0000-0000-0000-000000000002', 3, 'mcq', 'Which question word means "how much"?', '¿_____ cuesta este libro?', ARRAY['Cuánto','Cuándo','Cuál','Qué'], 'Cuánto', ARRAY[]::text[], 'Cuánto asks for price or quantity.', true),
    ('g0000000-0000-0000-0000-000000000002', 4, 'reconstruction', 'Arrange into a question.', '', ARRAY['español?','¿Hablas','tú','bien'], '¿Hablas tú bien español?', ARRAY['¿Tú hablas bien español?'], 'Question word order in Spanish.', true),

    -- Point 3: present-regular-reflexive-intro
    ('g0000000-0000-0000-0000-000000000003', 1, 'cloze', 'Complete with the present tense of hablar.', 'Ellos _____ español y francés.', ARRAY[]::text[], 'hablan', ARRAY['Hablan'], '-ar ending for ellos/ellas is -an.', true),
    ('g0000000-0000-0000-0000-000000000003', 2, 'mcq', 'Choose the correct reflexive pronoun.', 'Yo _____ levanto a las siete de la mañana.', ARRAY['me','te','se','nos'], 'me', ARRAY[]::text[], '1st person singular reflexive pronoun is "me".', true),
    ('g0000000-0000-0000-0000-000000000003', 3, 'cloze', 'Complete with the present tense of vivir.', 'María _____ en el centro de Barcelona.', ARRAY[]::text[], 'vive', ARRAY['Vive'], '-ir 3rd person singular ending is -e.', true),
    ('g0000000-0000-0000-0000-000000000003', 4, 'reconstruction', 'Reconstruct the daily routine sentence.', '', ARRAY['a','las','Me','ocho','duermo'], 'Me duermo a las ocho', ARRAY[]::text[], 'Reflexive pronoun + verb + time phrase.', true),

    -- Point 4: querer-quisiera-articles
    ('g0000000-0000-0000-0000-000000000004', 1, 'cloze', 'Complete with the polite ordering chunk.', '_____ un café solo, por favor.', ARRAY[]::text[], 'Quisiera', ARRAY['quisiera'], 'Quisiera is polite formula for ordering.', true),
    ('g0000000-0000-0000-0000-000000000004', 2, 'mcq', 'Select the correct definite article.', '_____ agua con gas está muy fría.', ARRAY['El','La','Los','Las'], 'El', ARRAY[]::text[], 'Agua is feminine but takes "el" in singular to avoid vowel clash.', true),
    ('g0000000-0000-0000-0000-000000000004', 3, 'mcq', 'Select the polite request phrase.', '¿_____ traer la cuenta, por favor?', ARRAY['Podría','Quiero','Trae','Debes'], 'Podría', ARRAY[]::text[], 'Podría brings formal politeness to a request.', true),
    ('g0000000-0000-0000-0000-000000000004', 4, 'reconstruction', 'Form an order.', '', ARRAY['un','por','Quisiera','favor','café'], 'Quisiera un café por favor', ARRAY[]::text[], 'Polite request word order.', true)
ON CONFLICT (grammar_point_id, ordinal) DO UPDATE SET
    item_type = EXCLUDED.item_type,
    prompt = EXCLUDED.prompt,
    sentence_with_blank = EXCLUDED.sentence_with_blank,
    choices = EXCLUDED.choices,
    correct = EXCLUDED.correct,
    accept_variants = EXCLUDED.accept_variants,
    note = EXCLUDED.note,
    is_active = EXCLUDED.is_active;
