package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

type CurriculumService struct {
	db *sql.DB
}

func NewCurriculumService(db *sql.DB) *CurriculumService {
	return &CurriculumService{db: db}
}

type curriculumUnitSeed struct {
	Slug             string
	Level            string
	Ordinal          int
	Title            string
	CanDo            string
	Description      string
	EstimatedMinutes int
	Checkpoint       bool
	GrammarSlug      string
	GrammarTitle     string
}

type lexicalSeed struct {
	UnitSlug     string
	Lemma        string
	DisplayText  string
	PartOfSpeech string
	Level        string
	Translation  string
	Tags         []string
	Frequency    int
	IsChunk      bool
}

type scenarioPhaseSeed struct {
	Ordinal         int
	Title           string
	LearnerGoal     string
	RequiredIntents []string
	ChunkBank       []map[string]string
}

func (s *CurriculumService) SeedDefaultCourses(ctx context.Context) error {
	courseID, err := s.upsertCourse(ctx, "en", "es", "Spanish for English Speakers", "v1", "full_course")
	if err != nil {
		return err
	}

	unitIDs := make(map[string]string, len(spanishUnits))
	for _, unit := range spanishUnits {
		id, err := s.upsertUnit(ctx, courseID, unit)
		if err != nil {
			return err
		}
		unitIDs[unit.Slug] = id
		if err := s.seedLessons(ctx, id, unit); err != nil {
			return err
		}
		if err := s.seedGrammarPoint(ctx, courseID, id, unit); err != nil {
			return err
		}
	}

	for _, item := range spanishLexicalItems {
		unitID := unitIDs[item.UnitSlug]
		if unitID == "" {
			return fmt.Errorf("missing unit %q for lexical seed %q", item.UnitSlug, item.Lemma)
		}
		if err := s.upsertLexicalItem(ctx, courseID, unitID, item); err != nil {
			return err
		}
	}

	if err := s.seedOrderingCoffeeScenario(ctx, courseID, unitIDs["a1-ordering-food"]); err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO learning_pair_capabilities (
			native_language, target_language, support_tier, active_course_id,
			placement_enabled, roadmap_enabled, scenarios_enabled,
			srs_enabled, mining_enabled, grammar_feedback_enabled, quality_notes
		)
		VALUES ('en', 'es', 'full_course', $1, true, true, true, true, true, true,
		        'Launch course: curated A1-B2 Spanish learning path for English speakers.')
		ON CONFLICT (native_language, target_language) DO UPDATE SET
			support_tier = EXCLUDED.support_tier,
			active_course_id = EXCLUDED.active_course_id,
			placement_enabled = EXCLUDED.placement_enabled,
			roadmap_enabled = EXCLUDED.roadmap_enabled,
			scenarios_enabled = EXCLUDED.scenarios_enabled,
			srs_enabled = EXCLUDED.srs_enabled,
			mining_enabled = EXCLUDED.mining_enabled,
			grammar_feedback_enabled = EXCLUDED.grammar_feedback_enabled,
			quality_notes = EXCLUDED.quality_notes,
			updated_at = CURRENT_TIMESTAMP
	`, courseID)
	return err
}

func (s *CurriculumService) upsertCourse(ctx context.Context, nativeLanguage, targetLanguage, title, version, supportTier string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO curriculum_courses (native_language, target_language, title, version, support_tier, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (target_language, native_language, version) DO UPDATE SET
			title = EXCLUDED.title,
			support_tier = EXCLUDED.support_tier,
			is_active = true
		RETURNING id::text
	`, nativeLanguage, targetLanguage, title, version, supportTier).Scan(&id)
	return id, err
}

func (s *CurriculumService) upsertUnit(ctx context.Context, courseID string, unit curriculumUnitSeed) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO curriculum_units (
			course_id, cefr_level, ordinal, slug, title, can_do_statement,
			description, estimated_minutes, checkpoint_required
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (course_id, slug) DO UPDATE SET
			cefr_level = EXCLUDED.cefr_level,
			ordinal = EXCLUDED.ordinal,
			title = EXCLUDED.title,
			can_do_statement = EXCLUDED.can_do_statement,
			description = EXCLUDED.description,
			estimated_minutes = EXCLUDED.estimated_minutes,
			checkpoint_required = EXCLUDED.checkpoint_required
		RETURNING id::text
	`, courseID, unit.Level, unit.Ordinal, unit.Slug, unit.Title, unit.CanDo,
		unit.Description, unit.EstimatedMinutes, unit.Checkpoint).Scan(&id)
	return id, err
}

func (s *CurriculumService) seedLessons(ctx context.Context, unitID string, unit curriculumUnitSeed) error {
	lessons := []struct {
		Slug      string
		Type      string
		Title     string
		Objective string
	}{
		{"vocabulary", "vocabulary", unit.Title + " Vocabulary", "Practice the most useful words and chunks for: " + unit.CanDo},
		{"grammar", "grammar", unit.Title + " Grammar", "Understand the key grammar pattern for this unit."},
		{"reading", "reading", unit.Title + " In Context", "Read a short real-world passage using this unit's language."},
		{"production", "production", unit.Title + " Practice", "Produce short answers connected to your real conversations."},
		{"scenario", "scenario_intro", unit.Title + " Scenario", "Prepare for a real-world roleplay."},
	}
	if !unit.Checkpoint {
		lessons = append(lessons, struct {
			Slug      string
			Type      string
			Title     string
			Objective string
		}{"checkpoint", "checkpoint", unit.Title + " Checkpoint", "Show you can use this unit's language independently."})
	}
	if unit.Checkpoint {
		lessons = []struct {
			Slug      string
			Type      string
			Title     string
			Objective string
		}{
			{"mixed-vocab", "vocabulary", unit.Title + " Vocabulary Review", "Review vocabulary across this CEFR band."},
			{"mixed-grammar", "grammar", unit.Title + " Grammar Review", "Review grammar across this CEFR band."},
			{"reading-check", "reading", unit.Title + " Reading Check", "Read and answer level-appropriate questions."},
			{"scenario-check", "scenario_intro", unit.Title + " Scenario Check", "Complete a multi-step real-world task."},
			{"checkpoint", "checkpoint", unit.Title, "Pass the checkpoint to unlock the next band."},
		}
	}

	for i, lesson := range lessons {
		var lessonID string
		err := s.db.QueryRowContext(ctx, `
			INSERT INTO curriculum_lessons (unit_id, ordinal, slug, type, title, objective, estimated_minutes)
			VALUES ($1, $2, $3, $4, $5, $6, 5)
			ON CONFLICT (unit_id, slug) DO UPDATE SET
				ordinal = EXCLUDED.ordinal,
				type = EXCLUDED.type,
				title = EXCLUDED.title,
				objective = EXCLUDED.objective,
				estimated_minutes = EXCLUDED.estimated_minutes
			RETURNING id::text
		`, unitID, i+1, lesson.Slug, lesson.Type, lesson.Title, lesson.Objective).Scan(&lessonID)
		if err != nil {
			return err
		}
		if err := s.seedIntroStep(ctx, lessonID, lesson.Title, lesson.Objective); err != nil {
			return err
		}
	}
	return nil
}

func (s *CurriculumService) seedIntroStep(ctx context.Context, lessonID, title, objective string) error {
	prompt, _ := json.Marshal(map[string]string{
		"title":     title,
		"objective": objective,
	})
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO curriculum_lesson_steps (lesson_id, ordinal, type, prompt, answer_key, content_refs)
		VALUES ($1, 1, 'intro', $2, '{}', '{}')
		ON CONFLICT (lesson_id, ordinal) DO UPDATE SET
			type = EXCLUDED.type,
			prompt = EXCLUDED.prompt
	`, lessonID, prompt)
	return err
}

func (s *CurriculumService) seedGrammarPoint(ctx context.Context, courseID, unitID string, unit curriculumUnitSeed) error {
	examples, _ := json.Marshal([]string{unit.CanDo})
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO grammar_points (course_id, unit_id, slug, cefr_level, title, short_explanation, examples)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (course_id, slug) DO UPDATE SET
			unit_id = EXCLUDED.unit_id,
			cefr_level = EXCLUDED.cefr_level,
			title = EXCLUDED.title,
			short_explanation = EXCLUDED.short_explanation,
			examples = EXCLUDED.examples
	`, courseID, unitID, unit.GrammarSlug, unit.Level, unit.GrammarTitle,
		"Core pattern for "+unit.Title+". Explanations should be adapted for English speakers learning Spanish.",
		examples)
	return err
}

func (s *CurriculumService) upsertLexicalItem(ctx context.Context, courseID, unitID string, item lexicalSeed) error {
	translations, _ := json.Marshal(map[string]string{"en": item.Translation})
	forms, _ := json.Marshal(map[string]any{})
	var freq any
	if item.Frequency > 0 {
		freq = item.Frequency
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO lexical_items (
			course_id, unit_id, language, lemma, display_text, part_of_speech,
			cefr_level, translations, forms, tags, frequency_rank, is_chunk
		)
		VALUES ($1, $2, 'es', $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (course_id, language, lemma, part_of_speech) DO UPDATE SET
			unit_id = EXCLUDED.unit_id,
			display_text = EXCLUDED.display_text,
			cefr_level = EXCLUDED.cefr_level,
			translations = EXCLUDED.translations,
			forms = EXCLUDED.forms,
			tags = EXCLUDED.tags,
			frequency_rank = EXCLUDED.frequency_rank,
			is_chunk = EXCLUDED.is_chunk
	`, courseID, unitID, item.Lemma, item.DisplayText, item.PartOfSpeech,
		item.Level, translations, forms, pq.Array(item.Tags), freq, item.IsChunk)
	return err
}

func (s *CurriculumService) seedOrderingCoffeeScenario(ctx context.Context, courseID, unitID string) error {
	if unitID == "" {
		return fmt.Errorf("ordering coffee scenario requires a unit")
	}
	criteria, _ := json.Marshal(map[string]any{
		"required_phase_count":          4,
		"required_intents":              []string{"greet", "order_drink", "pay", "close"},
		"min_user_turns":                4,
		"min_score":                     700,
		"allowed_native_language_turns": 1,
	})

	var scenarioID string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO scenario_scripts (
			course_id, unit_id, slug, title, domain, cefr_level, can_do_statement,
			ai_role_name, ai_role_description, opening_line, max_turns,
			estimated_minutes, completion_criteria
		)
		VALUES ($1, $2, 'ordering-coffee', 'Ordering Coffee at a Cafe', 'food_drink', 'A1',
		        'I can order a drink, ask about prices, and respond to simple service questions.',
		        'Sparky', 'Friendly cafe barista. Speaks clear A1 Spanish.',
		        'Hola. ¿Qué te gustaría pedir hoy?', 10, 5, $3)
		ON CONFLICT (course_id, slug) DO UPDATE SET
			unit_id = EXCLUDED.unit_id,
			title = EXCLUDED.title,
			domain = EXCLUDED.domain,
			cefr_level = EXCLUDED.cefr_level,
			can_do_statement = EXCLUDED.can_do_statement,
			ai_role_name = EXCLUDED.ai_role_name,
			ai_role_description = EXCLUDED.ai_role_description,
			opening_line = EXCLUDED.opening_line,
			max_turns = EXCLUDED.max_turns,
			estimated_minutes = EXCLUDED.estimated_minutes,
			completion_criteria = EXCLUDED.completion_criteria
		RETURNING id::text
	`, courseID, unitID, criteria).Scan(&scenarioID)
	if err != nil {
		return err
	}

	for _, phase := range orderingCoffeePhases {
		chunks, _ := json.Marshal(phase.ChunkBank)
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO scenario_phases (scenario_id, ordinal, title, learner_goal, required_intents, chunk_bank)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (scenario_id, ordinal) DO UPDATE SET
				title = EXCLUDED.title,
				learner_goal = EXCLUDED.learner_goal,
				required_intents = EXCLUDED.required_intents,
				chunk_bank = EXCLUDED.chunk_bank
		`, scenarioID, phase.Ordinal, phase.Title, phase.LearnerGoal,
			pq.Array(phase.RequiredIntents), chunks)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CurriculumService) GetFirstUnitID(ctx context.Context, courseID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text
		FROM curriculum_units
		WHERE course_id = $1
		ORDER BY ordinal
		LIMIT 1
	`, courseID).Scan(&id)
	return id, err
}

func (s *CurriculumService) GetLearningPath(ctx context.Context, userID string, profile *models.UserLanguageProfile, capability *models.LearningPairCapability) (*models.LearningPath, error) {
	path := &models.LearningPath{
		Capability: *capability,
		Profile:    *profile,
		Units:      []models.UnitProgressSummary{},
	}
	if capability.SupportTier != string(models.LearningSupportFullCourse) || capability.ActiveCourseID == "" {
		return path, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			u.id::text, u.course_id::text, u.cefr_level, u.ordinal, u.slug, u.title,
			u.can_do_statement, u.description, u.estimated_minutes, u.checkpoint_required,
			COALESCE(p.status, '') AS status,
			COALESCE(p.progress_pct, 0) AS progress_pct,
			COALESCE(p.competency_score, 0) AS competency_score,
			COALESCE(p.lessons_completed, 0) AS lessons_completed,
			p.checkpoint_score, p.started_at, p.completed_at
		FROM curriculum_units u
		LEFT JOIN user_unit_progress p
			ON p.unit_id = u.id AND p.user_id = $2
		WHERE u.course_id = $1
		ORDER BY u.ordinal
	`, capability.ActiveCourseID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var unit models.UnitProgressSummary
		var checkpointScore sql.NullInt64
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(
			&unit.ID, &unit.CourseID, &unit.CEFRLevel, &unit.Ordinal, &unit.Slug,
			&unit.Title, &unit.CanDoStatement, &unit.Description,
			&unit.EstimatedMinutes, &unit.CheckpointRequired, &unit.Status,
			&unit.ProgressPct, &unit.CompetencyScore, &unit.LessonsCompleted,
			&checkpointScore, &startedAt, &completedAt,
		); err != nil {
			return nil, err
		}
		if unit.Status == "" {
			if unit.ID == profile.ActiveUnitID {
				unit.Status = "available"
			} else {
				unit.Status = "locked"
			}
		}
		if checkpointScore.Valid {
			score := int(checkpointScore.Int64)
			unit.CheckpointScore = &score
		}
		if startedAt.Valid {
			unit.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			unit.CompletedAt = &completedAt.Time
		}
		lessons, err := s.getLessonSummaries(ctx, userID, unit.ID)
		if err != nil {
			return nil, err
		}
		unit.Lessons = lessons
		path.Units = append(path.Units, unit)
	}
	return path, rows.Err()
}

func (s *CurriculumService) getLessonSummaries(ctx context.Context, userID, unitID string) ([]models.LessonSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id::text, l.unit_id::text, l.ordinal, l.slug, l.type,
		       l.title, l.objective, l.estimated_minutes,
		       CASE WHEN EXISTS (
		         SELECT 1 FROM user_lesson_attempts a
		         WHERE a.user_id = $2 AND a.lesson_id = l.id AND a.status = 'completed'
		       ) THEN 'completed' ELSE 'available' END AS status
		FROM curriculum_lessons l
		WHERE l.unit_id = $1
		ORDER BY l.ordinal
	`, unitID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lessons := []models.LessonSummary{}
	for rows.Next() {
		var lesson models.LessonSummary
		if err := rows.Scan(
			&lesson.ID,
			&lesson.UnitID,
			&lesson.Ordinal,
			&lesson.Slug,
			&lesson.Type,
			&lesson.Title,
			&lesson.Objective,
			&lesson.EstimatedMinutes,
			&lesson.Status,
		); err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	return lessons, rows.Err()
}

var spanishUnits = []curriculumUnitSeed{
	{"a1-introductions", "A1", 1, "Introductions", "I can greet someone and introduce myself.", "Start simple conversations and share basic identity.", 30, false, "subject-pronouns-ser-llamarse", "Subject pronouns, ser, and llamarse"},
	{"a1-basics-ii", "A1", 2, "Basics II", "I can ask simple identity and language questions.", "Ask and answer simple questions about people and languages.", 30, false, "yes-no-questions-question-words", "Yes/no questions and question words"},
	{"a1-daily-routine", "A1", 3, "Daily Routine", "I can talk about simple daily habits.", "Describe everyday actions and simple routines.", 35, false, "present-regular-reflexive-intro", "Present tense regular verbs and reflexive intro"},
	{"a1-ordering-food", "A1", 4, "Ordering Food", "I can order food or drink and ask prices.", "Use polite chunks for cafes, restaurants, and checkout moments.", 35, false, "querer-quisiera-articles", "Querer, quisiera chunks, gender, and articles"},
	{"a1-around-town", "A1", 5, "Around Town", "I can ask where something is and follow simple directions.", "Find places and understand basic directions.", 35, false, "estar-hay-prepositions", "Estar for location, hay, and prepositions"},
	{"a1-checkpoint", "A1", 6, "A1 Checkpoint", "I can complete basic social and service interactions.", "Review A1 foundations before moving to A2.", 40, true, "a1-review", "A1 review"},
	{"a2-past-weekend", "A2", 7, "Past Weekend", "I can describe simple completed events.", "Talk about yesterday, weekends, and completed plans.", 35, false, "preterite-regular-ir-ser", "Preterite regular verbs plus ir and ser"},
	{"a2-shopping", "A2", 8, "Shopping", "I can ask for items, sizes, and totals.", "Handle common store and grocery interactions.", 35, false, "demonstratives-direct-objects", "Demonstratives and direct objects"},
	{"a2-plans", "A2", 9, "Plans", "I can talk about near-future plans.", "Make plans, invite people, and talk about errands.", 35, false, "ir-a-infinitive-tener-que", "Ir a + infinitive and tener que"},
	{"a2-health-feelings", "A2", 10, "Health And Feelings", "I can describe how I feel and ask for help.", "Talk about symptoms, moods, and simple needs.", 35, false, "estar-vs-ser-doler", "Estar vs ser and doler chunks"},
	{"a2-travel-basics", "A2", 11, "Travel Basics", "I can book simple travel and lodging.", "Book rooms, ask transport questions, and handle travel basics.", 35, false, "polite-requests-poder-necesitar", "Polite requests with poder and necesitar"},
	{"a2-checkpoint", "A2", 12, "A2 Checkpoint", "I can handle predictable everyday tasks.", "Review A2 before moving into longer B1 conversations.", 45, true, "a2-review", "A2 review"},
	{"b1-stories", "B1", 13, "Stories", "I can narrate past experiences with sequence.", "Tell what happened with clear order and context.", 40, false, "preterite-vs-imperfect", "Preterite vs imperfect"},
	{"b1-opinions", "B1", 14, "Opinions", "I can give opinions and reasons.", "Explain preferences, media reactions, and simple arguments.", 40, false, "reasons-concession-comparisons", "Reasons, concession, and comparisons"},
	{"b1-problems", "B1", 15, "Problems", "I can explain a problem and request a solution.", "Describe service issues and ask for repair or help.", 40, false, "object-pronouns-formal-requests", "Object pronouns and formal requests"},
	{"b1-social-plans", "B1", 16, "Social Plans", "I can negotiate plans and preferences.", "Coordinate schedules and suggest alternatives.", 40, false, "conditional-suggestions", "Conditional intro and suggestions"},
	{"b1-work-study", "B1", 17, "Work And Study", "I can describe responsibilities and goals.", "Discuss work, study, skills, and future goals.", 40, false, "present-perfect-obligation", "Present perfect intro and obligation"},
	{"b1-checkpoint", "B1", 18, "B1 Checkpoint", "I can sustain conversations on familiar topics.", "Review B1 skills across longer conversations.", 50, true, "b1-review", "B1 review"},
	{"b2-nuanced-opinions", "B2", 19, "Nuanced Opinions", "I can defend a viewpoint with nuance.", "Discuss tradeoffs and explain a position clearly.", 45, false, "subjunctive-triggers-intro", "Subjunctive triggers intro"},
	{"b2-hypotheticals", "B2", 20, "Hypotheticals", "I can discuss imagined outcomes.", "Talk about risks, consequences, and possibilities.", 45, false, "si-clauses-conditional", "Si clauses and conditional"},
	{"b2-professional-communication", "B2", 21, "Professional Communication", "I can handle formal work conversations.", "Navigate meetings, deadlines, requests, and feedback.", 45, false, "register-formal-commands", "Register and formal commands"},
	{"b2-media-culture", "B2", 22, "Media And Culture", "I can summarize and react to articles.", "Discuss news, culture, and abstract ideas.", 45, false, "reported-speech-connectors", "Reported speech and connectors"},
	{"b2-conflict-repair", "B2", 23, "Conflict And Repair", "I can clarify misunderstandings and negotiate.", "Handle disagreement, apologies, and compromise.", 45, false, "concession-hedging-repair", "Concession, hedging, and repair phrases"},
	{"b2-checkpoint", "B2", 24, "B2 Checkpoint", "I can interact with fluency in varied contexts.", "Validate B2 readiness through integrated tasks.", 55, true, "b2-review", "B2 review"},
}

var spanishLexicalItems = []lexicalSeed{
	{"a1-introductions", "hola", "hola", "interjection", "A1", "hello", []string{"greeting"}, 1, false},
	{"a1-introductions", "me llamo", "me llamo", "verb_phrase", "A1", "my name is", []string{"introduction", "chunk"}, 2, true},
	{"a1-introductions", "gracias", "gracias", "interjection", "A1", "thank you", []string{"politeness"}, 3, false},
	{"a1-basics-ii", "idioma", "idioma", "noun", "A1", "language", []string{"identity"}, 20, false},
	{"a1-basics-ii", "¿de dónde eres?", "¿de dónde eres?", "question", "A1", "where are you from?", []string{"question", "identity"}, 21, true},
	{"a1-daily-routine", "levantarse", "levantarse", "verb", "A1", "to get up", []string{"daily_routine", "reflexive"}, 30, false},
	{"a1-daily-routine", "normalmente", "normalmente", "adverb", "A1", "usually", []string{"daily_routine"}, 31, false},
	{"a1-ordering-food", "café", "café", "noun", "A1", "coffee", []string{"cafe", "food_drink"}, 40, false},
	{"a1-ordering-food", "café con leche", "café con leche", "noun_phrase", "A1", "coffee with milk", []string{"cafe", "food_drink", "chunk"}, 41, true},
	{"a1-ordering-food", "quisiera", "quisiera", "verb", "A1", "I would like", []string{"cafe", "polite_request"}, 42, false},
	{"a1-ordering-food", "para llevar", "para llevar", "phrase", "A1", "to go", []string{"cafe", "service", "chunk"}, 43, true},
	{"a1-ordering-food", "¿cuánto cuesta?", "¿cuánto cuesta?", "question", "A1", "how much does it cost?", []string{"cafe", "payment", "chunk"}, 44, true},
	{"a1-around-town", "¿dónde está?", "¿dónde está?", "question", "A1", "where is it?", []string{"directions", "location"}, 50, true},
	{"a1-around-town", "a la izquierda", "a la izquierda", "phrase", "A1", "to the left", []string{"directions"}, 51, true},
	{"a2-past-weekend", "ayer", "ayer", "adverb", "A2", "yesterday", []string{"past_time"}, 70, false},
	{"a2-past-weekend", "el fin de semana", "el fin de semana", "noun_phrase", "A2", "the weekend", []string{"time", "leisure"}, 71, true},
	{"a2-shopping", "bolsa", "bolsa", "noun", "A2", "bag", []string{"shopping"}, 80, false},
	{"a2-shopping", "total", "total", "noun", "A2", "total", []string{"shopping", "payment"}, 81, false},
	{"a2-plans", "voy a", "voy a", "verb_phrase", "A2", "I am going to", []string{"plans", "near_future"}, 90, true},
	{"a2-health-feelings", "estoy cansado", "estoy cansado", "phrase", "A2", "I am tired", []string{"feelings", "estar"}, 100, true},
	{"a2-travel-basics", "necesito reservar", "necesito reservar", "verb_phrase", "A2", "I need to book", []string{"travel", "hotel"}, 110, true},
	{"b1-stories", "primero", "primero", "connector", "B1", "first", []string{"story", "sequence"}, 130, false},
	{"b1-stories", "después", "después", "connector", "B1", "afterward", []string{"story", "sequence"}, 131, false},
	{"b1-opinions", "aunque", "aunque", "connector", "B1", "although", []string{"opinion", "concession"}, 140, false},
	{"b1-problems", "hay un problema", "hay un problema", "phrase", "B1", "there is a problem", []string{"service", "problem"}, 150, true},
	{"b1-social-plans", "podríamos", "podríamos", "verb", "B1", "we could", []string{"plans", "conditional"}, 160, false},
	{"b1-work-study", "responsabilidad", "responsabilidad", "noun", "B1", "responsibility", []string{"work", "study"}, 170, false},
	{"b2-nuanced-opinions", "por un lado", "por un lado", "connector", "B2", "on one hand", []string{"argument", "nuance"}, 190, true},
	{"b2-hypotheticals", "si tuviera", "si tuviera", "phrase", "B2", "if I had", []string{"hypothetical"}, 200, true},
	{"b2-professional-communication", "me gustaría reprogramar", "me gustaría reprogramar", "phrase", "B2", "I would like to reschedule", []string{"professional", "formal"}, 210, true},
	{"b2-media-culture", "según el artículo", "según el artículo", "phrase", "B2", "according to the article", []string{"media", "summary"}, 220, true},
	{"b2-conflict-repair", "entiendo tu punto", "entiendo tu punto", "phrase", "B2", "I understand your point", []string{"conflict", "repair"}, 230, true},
}

var orderingCoffeePhases = []scenarioPhaseSeed{
	{1, "Greeting", "Greet the barista.", []string{"greet"}, []map[string]string{
		{"text": "Hola, buenos días.", "translation": "Hello, good morning."},
		{"text": "Buenas tardes.", "translation": "Good afternoon."},
	}},
	{2, "Order", "Order one drink.", []string{"order_drink"}, []map[string]string{
		{"text": "Quisiera un café con leche, por favor.", "translation": "I would like a coffee with milk, please."},
		{"text": "¿Me puede dar un café, por favor?", "translation": "Can you give me a coffee, please?"},
	}},
	{3, "Customization", "Answer or request a simple option.", []string{"customize"}, []map[string]string{
		{"text": "Para llevar, por favor.", "translation": "To go, please."},
		{"text": "Sin azúcar, por favor.", "translation": "Without sugar, please."},
		{"text": "¿Tiene leche de avena?", "translation": "Do you have oat milk?"},
	}},
	{4, "Payment", "Ask or understand the price.", []string{"pay"}, []map[string]string{
		{"text": "¿Cuánto cuesta?", "translation": "How much does it cost?"},
		{"text": "¿Aceptan tarjeta?", "translation": "Do you accept card?"},
	}},
	{5, "Closing", "Close politely.", []string{"close"}, []map[string]string{
		{"text": "Gracias.", "translation": "Thank you."},
		{"text": "Que tenga un buen día.", "translation": "Have a good day."},
	}},
}
