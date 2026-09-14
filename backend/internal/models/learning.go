package models

// Domain types have been split into focused domain files:
// - learning_profile.go     (UserLanguageProfile, LearningPairCapability, DailyGoal*, Dashboard)
// - learning_curriculum.go  (CurriculumCourse, Unit, Lesson, Step, LexicalItem, GrammarPoint, ReadingPassage)
// - learning_grammar.go     (GrammarItem, GrammarItemAttempt, UserGrammarMastery, GrammarMicroLesson, GrammarDrillItem)
// - learning_session.go     (VocabularyCard, LearningSession, SessionItem, SRSQueue*, SessionQuestion)
// - learning_placement.go   (PlacementQuestion, PlacementResult, PlacementItem, PlacementStartResponse)
// - learning_scenario.go    (ScenarioScript, Phase, Run, Turn, ScenarioAIReply)
// - learning_teacher.go     (TeacherLesson, TeacherLessonStep, TeacherAssignment, AssignmentSubmission, StudentProgressSummary)
// - learning_jobs.go        (GradingJob, GradingJobPayload, GradingJobResult)
// - vocabulary.go           (MinedItem)
