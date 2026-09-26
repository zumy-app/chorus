-- 05_grammar_items.sql: Interactive drill items for Grammar Points
INSERT INTO grammar_items (
    grammar_point_id, ordinal, item_type, prompt, sentence_with_blank, choices, correct, accept_variants, note, is_active
)
VALUES
    -- Point 1: ser-personal-pronouns
    ('b0000000-0000-0000-0000-000000000001', 1, 'cloze', 'Complete the sentence with the correct form of ser.', 'Pedro _____ de Madrid.', ARRAY[]::text[], 'es', ARRAY['Es'], 'Pedro is 3rd person singular (él), so use "es".', true),
    ('b0000000-0000-0000-0000-000000000001', 2, 'mcq', 'Select the correct pronoun form.', '¿De dónde _____ vosotros?', ARRAY['sois','somos','son','eres'], 'sois', ARRAY[]::text[], 'Vosotros pairs with "sois".', true),
    ('b0000000-0000-0000-0000-000000000001', 3, 'reconstruction', 'Reconstruct the sentence in the correct order.', '', ARRAY['Yo','estudiante','español','de','soy'], 'Yo soy estudiante de español', ARRAY['Soy estudiante de español'], 'Subject + verb + predicate.', true),
    ('b0000000-0000-0000-0000-000000000001', 4, 'cloze', 'Fill in the blank with ser.', 'Nosotros _____ amigos desde la infancia.', ARRAY[]::text[], 'somos', ARRAY['Somos'], 'Nosotros takes "somos".', true),

    -- Point 2: yes-no-questions-question-words
    ('b0000000-0000-0000-0000-000000000002', 1, 'mcq', 'Which word asks for location?', '¿_____ está la estación de tren?', ARRAY['Dónde','Cuándo','Cómo','Quién'], 'Dónde', ARRAY['dónde'], 'Dónde asks "where".', true),
    ('b0000000-0000-0000-0000-000000000002', 2, 'cloze', 'Complete the question for asking someone''s name.', '¿_____ te llamas?', ARRAY[]::text[], 'Cómo', ARRAY['cómo','Como'], '¿Cómo te llamas? literally means "How do you call yourself?".', true),
    ('b0000000-0000-0000-0000-000000000002', 3, 'mcq', 'Which question word means "how much"?', '¿_____ cuesta este libro?', ARRAY['Cuánto','Cuándo','Cuál','Qué'], 'Cuánto', ARRAY[]::text[], 'Cuánto asks for price or quantity.', true),
    ('b0000000-0000-0000-0000-000000000002', 4, 'reconstruction', 'Arrange into a question.', '', ARRAY['español?','¿Hablas','tú','bien'], '¿Hablas tú bien español?', ARRAY['¿Tú hablas bien español?'], 'Question word order in Spanish.', true),

    -- Point 3: present-regular-reflexive-intro
    ('b0000000-0000-0000-0000-000000000003', 1, 'cloze', 'Complete with the present tense of hablar.', 'Ellos _____ español y francés.', ARRAY[]::text[], 'hablan', ARRAY['Hablan'], '-ar ending for ellos/ellas is -an.', true),
    ('b0000000-0000-0000-0000-000000000003', 2, 'mcq', 'Choose the correct reflexive pronoun.', 'Yo _____ levanto a las siete de la mañana.', ARRAY['me','te','se','nos'], 'me', ARRAY[]::text[], '1st person singular reflexive pronoun is "me".', true),
    ('b0000000-0000-0000-0000-000000000003', 3, 'cloze', 'Complete with the present tense of vivir.', 'María _____ en el centro de Barcelona.', ARRAY[]::text[], 'vive', ARRAY['Vive'], '-ir 3rd person singular ending is -e.', true),
    ('b0000000-0000-0000-0000-000000000003', 4, 'reconstruction', 'Reconstruct the daily routine sentence.', '', ARRAY['a','las','Me','ocho','duermo'], 'Me duermo a las ocho', ARRAY[]::text[], 'Reflexive pronoun + verb + time phrase.', true),

    -- Point 4: querer-quisiera-articles
    ('b0000000-0000-0000-0000-000000000004', 1, 'cloze', 'Complete with the polite ordering chunk.', '_____ un café solo, por favor.', ARRAY[]::text[], 'Quisiera', ARRAY['quisiera'], 'Quisiera is polite formula for ordering.', true),
    ('b0000000-0000-0000-0000-000000000004', 2, 'mcq', 'Select the correct definite article.', '_____ agua con gas está muy fría.', ARRAY['El','La','Los','Las'], 'El', ARRAY[]::text[], 'Agua is feminine but takes "el" in singular to avoid vowel clash.', true),
    ('b0000000-0000-0000-0000-000000000004', 3, 'mcq', 'Select the polite request phrase.', '¿_____ traer la cuenta, por favor?', ARRAY['Podría','Quiero','Trae','Debes'], 'Podría', ARRAY[]::text[], 'Podría brings formal politeness to a request.', true),
    ('b0000000-0000-0000-0000-000000000004', 4, 'reconstruction', 'Form an order.', '', ARRAY['un','por','Quisiera','favor','café'], 'Quisiera un café por favor', ARRAY[]::text[], 'Polite request word order.', true),

    -- Point 5: estar-hay-prepositions
    ('b0000000-0000-0000-0000-000000000005', 1, 'cloze', 'Complete with hay or está.', '_____ una farmacia cerca del hotel.', ARRAY[]::text[], 'hay', ARRAY['Hay']::text[], 'Use hay for existence with un/una.', true),
    ('b0000000-0000-0000-0000-000000000005', 2, 'mcq', 'Select the correct form.', 'El museo _____ al lado del parque.', ARRAY['está','hay','es','son'], 'está', ARRAY[]::text[], 'Use estar for specific location.', true),
    ('b0000000-0000-0000-0000-000000000005', 3, 'cloze', 'Complete the direction.', 'Gira a la _____ en la esquina.', ARRAY[]::text[], 'izquierda', ARRAY['Izquierda']::text[], 'izquierda = left, derecha = right.', true),

    -- Point 6: preterite-regular-ir-ser
    ('b0000000-0000-0000-0000-000000000006', 1, 'cloze', 'Complete in preterite.', 'Ayer _____ al cine con mis amigos.', ARRAY[]::text[], 'fui', ARRAY['Fui']::text[], 'Ir and ser share fui in the preterite.', true),
    ('b0000000-0000-0000-0000-000000000006', 2, 'mcq', 'Select the correct preterite ending.', 'Ayer nosotros _____ una paella riquísima.', ARRAY['comimos','comemos','comamos','comeremos'], 'comimos', ARRAY[]::text[], '-er/-ir preterite for nosotros is -imos.', true),
    ('b0000000-0000-0000-0000-000000000006', 3, 'cloze', 'Complete in preterite.', 'Ella _____ (salir) con sus amigos ayer.', ARRAY[]::text[], 'salió', ARRAY['Salió']::text[], 'Preterite -ir verb 3rd person: salió.', true),

    -- Point 7: demonstratives-direct-objects
    ('b0000000-0000-0000-0000-000000000007', 1, 'mcq', 'Select the correct demonstrative.', '¿Cuánto cuesta _____ camisa?', ARRAY['esta','ese','esos','esas'], 'esta', ARRAY[]::text[], 'camisa is feminine singular -> esta.', true),
    ('b0000000-0000-0000-0000-000000000007', 2, 'cloze', 'Replace the noun with a direct object pronoun.', '¿Compras el sombrero? Sí, lo _____ hoy.', ARRAY[]::text[], 'compro', ARRAY['Compro']::text[], 'Direct object pronouns go before the conjugated verb.', true),

    -- Point 8: ir-a-infinitive-tener-que
    ('b0000000-0000-0000-0000-000000000008', 1, 'cloze', 'Complete with ir a + infinitive.', 'Voy a _____ a México el mes que viene.', ARRAY[]::text[], 'viajar', ARRAY['Viajar']::text[], 'ir a + infinitive = going to + verb.', true),
    ('b0000000-0000-0000-0000-000000000008', 2, 'mcq', 'Select the correct obligation phrase.', '_____ trabajar mañana todo el día.', ARRAY['Tengo que','Tengo a','Debo que','Hay que yo'], 'Tengo que', ARRAY[]::text[], 'tener que + infinitive = have to.', true),

    -- Point 9: preterite-vs-imperfect
    ('b0000000-0000-0000-0000-000000000009', 1, 'mcq', 'Select preterite or imperfect.', 'Cuando llegué a la playa, _____ mucho sol.', ARRAY['hacía','hizo','hará','hace'], 'hacía', ARRAY[]::text[], 'Background weather in a story takes the imperfect.', true),
    ('b0000000-0000-0000-0000-000000000009', 2, 'cloze', 'Complete with the imperfect.', 'De niño yo _____ al fútbol todos los días.', ARRAY[]::text[], 'jugaba', ARRAY['Jugaba']::text[], 'Habitual past actions take the imperfect.', true),

    -- Point 10: subjunctive-triggers-intro
    ('b0000000-0000-0000-0000-000000000010', 1, 'cloze', 'Complete with the present subjunctive.', 'Espero que _____ un buen viaje.', ARRAY[]::text[], 'tengas', ARRAY['Tengas']::text[], 'Espero que + subjunctive for wishes.', true),
    ('b0000000-0000-0000-0000-000000000010', 2, 'mcq', 'Select the correct mood.', 'Dudo que _____ la verdad.', ARRAY['sea','es','ser','está'], 'sea', ARRAY[]::text[], 'Doubt triggers the subjunctive after dudo que.', true)ON CONFLICT (grammar_point_id, ordinal) DO UPDATE SET
    item_type = EXCLUDED.item_type,
    prompt = EXCLUDED.prompt,
    sentence_with_blank = EXCLUDED.sentence_with_blank,
    choices = EXCLUDED.choices,
    correct = EXCLUDED.correct,
    accept_variants = EXCLUDED.accept_variants,
    note = EXCLUDED.note,
    is_active = EXCLUDED.is_active;
