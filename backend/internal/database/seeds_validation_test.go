package database

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// Shift-Left Content Validation (plan V3): because curriculum structures
// compile directly into the application binary, seed content is verified here
// before it can reach any staging system. These checks run on the embedded
// files themselves, so a malformed seed fails `go test` instead of crashing
// server startup against a live database.

var uuidLike = regexp.MustCompile(`'([0-9a-zA-Z]{8}-[0-9a-zA-Z]{4}-[0-9a-zA-Z]{4}-[0-9a-zA-Z]{4}-[0-9a-zA-Z]{12})'`)
var hexChars = regexp.MustCompile(`^[0-9a-fA-F-]+$`)

// jsonbLiterals extracts every '...'::jsonb payload, treating doubled quotes
// (”) as escaped content so contractions inside JSON strings survive.
func jsonbLiterals(content string) []string {
	unescaped := strings.ReplaceAll(content, "''", "\x00")
	re := regexp.MustCompile(`'([^']*)'::jsonb`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(unescaped, -1) {
		out = append(out, strings.ReplaceAll(m[1], "\x00", "'"))
	}
	return out
}

func seedFilesForPair(t *testing.T, pair string) map[string]string {
	t.Helper()
	entries, err := fs.Glob(SeedFiles, "seeds/"+pair+"/*.sql")
	if err != nil {
		t.Fatalf("glob seeds for %s: %v", pair, err)
	}
	if len(entries) == 0 {
		t.Fatalf("no seed files found for pair %s", pair)
	}
	files := map[string]string{}
	for _, e := range entries {
		content, err := SeedFiles.ReadFile(e)
		if err != nil {
			t.Fatalf("read %s: %v", e, err)
		}
		files[e] = string(content)
	}
	return files
}

var seededPairs = []string{"en_es", "es_en"}

// TestSeedUUIDsAreHexValid guards against semantic-but-non-hex id prefixes
// (e.g. "u0000000-...") that Postgres silently rejects at INSERT time with
// "invalid input syntax for type uuid" — a failure mode sqlmock cannot catch.
func TestSeedUUIDsAreHexValid(t *testing.T) {
	for _, pair := range seededPairs {
		for path, content := range seedFilesForPair(t, pair) {
			for _, m := range uuidLike.FindAllStringSubmatch(content, -1) {
				id := m[1]
				if !hexChars.MatchString(id) {
					t.Errorf("%s: non-hex UUID literal %q (Postgres would reject this at insert time)", path, id)
				}
			}
		}
	}
}

// TestSeedJSONBPayloadsAreWellFormed verifies every ::jsonb payload parses.
func TestSeedJSONBPayloadsAreWellFormed(t *testing.T) {
	for _, pair := range seededPairs {
		for path, content := range seedFilesForPair(t, pair) {
			for i, lit := range jsonbLiterals(content) {
				var v any
				if err := json.Unmarshal([]byte(lit), &v); err != nil {
					t.Errorf("%s: jsonb literal #%d does not parse: %v\n%s", path, i+1, err, lit)
				}
			}
		}
	}
}

// uuidsByPrefix collects distinct UUID literals starting with the given hex
// prefix (course=c, unit=a, grammar point=b, scenario=d).
func uuidsByPrefix(content, prefix string) map[string]bool {
	out := map[string]bool{}
	for _, m := range uuidLike.FindAllStringSubmatch(content, -1) {
		if strings.HasPrefix(m[1], prefix) {
			out[m[1]] = true
		}
	}
	return out
}

// TestSeedGrammarPointsHaveAdequateItems: every grammar point seeded in
// 04_grammar_points must carry at least two drill items in 05_grammar_items.
func TestSeedGrammarPointsHaveAdequateItems(t *testing.T) {
	for _, pair := range seededPairs {
		files := seedFilesForPair(t, pair)
		var points04, items05 string
		for path, content := range files {
			switch {
			case strings.Contains(path, "04_grammar_points"):
				points04 = content
			case strings.Contains(path, "05_grammar_items"):
				items05 = content
			}
		}
		if points04 == "" || items05 == "" {
			t.Fatalf("%s: missing grammar seed files (04=%v 05=%v)", pair, points04 != "", items05 != "")
		}
		points := uuidsByPrefix(points04, "b")
		if len(points) == 0 {
			t.Fatalf("%s: no grammar points seeded", pair)
		}
		itemRefCount := map[string]int{}
		for _, m := range uuidLike.FindAllStringSubmatch(items05, -1) {
			if strings.HasPrefix(m[1], "b") {
				itemRefCount[m[1]]++
			}
		}
		for point := range points {
			if itemRefCount[point] < 2 {
				t.Errorf("%s: grammar point %s has %d drill items (want >= 2)", pair, point, itemRefCount[point])
			}
		}
	}
}

// TestSeedRelationalLinksExist: unit/point/scenario references must resolve to
// rows created earlier in the seed chain (courses -> units -> content).
func TestSeedRelationalLinksExist(t *testing.T) {
	for _, pair := range seededPairs {
		files := seedFilesForPair(t, pair)
		var courses, units string
		for path, content := range files {
			switch {
			case strings.Contains(path, "01_course"):
				courses = content
			case strings.Contains(path, "02_units"):
				units = content
			}
		}
		courseIDs := uuidsByPrefix(courses, "c")
		unitIDs := uuidsByPrefix(units, "a")
		if len(courseIDs) == 0 || len(unitIDs) == 0 {
			t.Fatalf("%s: missing course/unit seeds", pair)
		}
		for path, content := range files {
			for id := range uuidsByPrefix(content, "c") {
				if strings.Contains(path, "01_course") {
					continue
				}
				if !courseIDs[id] {
					t.Errorf("%s: references unknown course %s", path, id)
				}
			}
			if !strings.Contains(path, "02_units") && !strings.Contains(path, "01_course") {
				for id := range uuidsByPrefix(content, "a") {
					if !unitIDs[id] {
						t.Errorf("%s: references unknown unit %s", path, id)
					}
				}
			}
		}
	}
}

// TestSeedPlacementBankStructure: the calibrated placement bank must hold the
// full 20-item module structure (8 receptive / 8 grammar / 4 discourse) with
// every module present.
func TestSeedPlacementBankStructure(t *testing.T) {
	for _, pair := range seededPairs {
		files := seedFilesForPair(t, pair)
		var placement string
		for path, content := range files {
			if strings.Contains(path, "07_placement_items") {
				placement = content
			}
		}
		if placement == "" {
			t.Fatalf("%s: no placement item seed", pair)
		}
		counts := map[string]int{"receptive_vocab": 0, "grammar_production": 0, "discourse_reading": 0}
		for module := range counts {
			counts[module] = strings.Count(placement, "'"+module+"'")
		}
		if counts["receptive_vocab"] < 8 {
			t.Errorf("%s: placement bank has %d receptive_vocab items (want >= 8)", pair, counts["receptive_vocab"])
		}
		if counts["grammar_production"] < 8 {
			t.Errorf("%s: placement bank has %d grammar_production items (want >= 8)", pair, counts["grammar_production"])
		}
		if counts["discourse_reading"] < 4 {
			t.Errorf("%s: placement bank has %d discourse_reading items (want >= 4)", pair, counts["discourse_reading"])
		}
	}
}

// TestSeedScenarioPhasesReferenceSeededScripts: every scenario id used by the
// phase inserts must belong to a scripted scenario in the same file.
func TestSeedScenarioPhasesReferenceSeededScripts(t *testing.T) {
	for _, pair := range seededPairs {
		for path, content := range seedFilesForPair(t, pair) {
			if !strings.Contains(path, "06_scenarios") {
				continue
			}
			split := strings.Index(content, "-- Phases")
			if split < 0 {
				t.Errorf("%s: no phase section found", path)
				continue
			}
			scripted := uuidsByPrefix(content[:split], "d")
			used := uuidsByPrefix(content[split:], "d")
			for id := range used {
				if !scripted[id] {
					t.Errorf("%s: phases reference unscripted scenario %s", path, id)
				}
			}
			if len(scripted) == 0 {
				t.Errorf("%s: no scenarios scripted", path)
			}
		}
	}
}

// TestSeedMultiPairCatalogue asserts the V3 Phase 9 expansion: both language
// pairs are present in the embedded catalogue with a complete file set.
func TestSeedMultiPairCatalogue(t *testing.T) {
	required := []string{
		"01_course", "02_units", "03_lexical_items", "04_grammar_points",
		"05_grammar_items", "06_scenarios", "07_placement_items",
		"08_reading_passages", "09_real_talk_prompts",
	}
	for _, pair := range seededPairs {
		files := seedFilesForPair(t, pair)
		for _, base := range required {
			found := false
			for path := range files {
				if strings.Contains(path, base) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("pair %s is missing seed file %s.sql", pair, base)
			}
		}
	}
}

// TestSeedValuesTuplesSeparated walks every VALUES list in the seed files with
// a quote/paren/bracket-aware scanner and asserts adjacent tuples are comma
// separated — the exact failure mode of hand-appended rows (a missing comma
// between tuples only explodes at INSERT time on a real database). The 00_
// adoption files are PL/pgSQL DO blocks (no INSERT VALUES tuples) and are
// excluded from this check.
func TestSeedValuesTuplesSeparated(t *testing.T) {
	for _, pair := range seededPairs {
		for path, content := range seedFilesForPair(t, pair) {
			if strings.Contains(path, "00_adopt") {
				continue
			}
			for _, section := range valuesSections(content) {
				tuples, err := scanTopLevelTuples(section)
				if err != nil {
					t.Errorf("%s: VALUES section scan failed: %v", path, err)
					continue
				}
				if len(tuples) == 0 {
					continue
				}
				for i := 1; i < len(tuples); i++ {
					if !tuples[i-1].commaAfter && !tuples[i].commaBefore {
						t.Errorf("%s: tuple %d not comma-separated from tuple %d", path, i, i-1)
					}
				}
			}
		}
	}
}

type tupleSpan struct {
	commaBefore bool
	commaAfter  bool
}

// valuesSections returns the raw text between each "VALUES" keyword and the
// following "ON CONFLICT" or statement end.
func valuesSections(content string) []string {
	var out []string
	rest := content
	for {
		i := strings.Index(rest, "VALUES")
		if i < 0 {
			return out
		}
		rest = rest[i+len("VALUES"):]
		end := len(rest)
		if j := strings.Index(rest, "ON CONFLICT"); j >= 0 {
			end = j
		}
		out = append(out, rest[:end])
		rest = rest[end:]
	}
}

// scanTopLevelTuples walks a VALUES section tracking single-quoted strings
// (with ” escapes), parens, and brackets, splitting top-level (...) tuples
// and recording whether each is comma separated from its neighbours.
func scanTopLevelTuples(section string) ([]tupleSpan, error) {
	var tuples []tupleSpan
	depth := 0
	inString := false
	current := strings.Builder{}
	pendingComma := false
	flush := func(commaAfter bool) {
		t := tupleSpan{commaBefore: pendingComma, commaAfter: commaAfter}
		if len(strings.TrimSpace(current.String())) > 0 {
			tuples = append(tuples, t)
		}
		current.Reset()
	}
	i := 0
	for i < len(section) {
		c := section[i]
		if inString {
			if c == '\'' {
				if i+1 < len(section) && section[i+1] == '\'' {
					current.WriteByte(c)
					current.WriteByte(section[i+1])
					i += 2
					continue
				}
				inString = false
			}
			current.WriteByte(c)
			i++
			continue
		}
		switch c {
		case '\'':
			inString = true
			current.WriteByte(c)
		case '-':
			// Line comment (outside strings): skip to end of line so
			// illustrative parens in comments never create phantom tuples.
			if i+1 < len(section) && section[i+1] == '-' {
				for i < len(section) && section[i] != '\n' {
					i++
				}
				continue
			}
			if depth > 0 {
				current.WriteByte(c)
			}
		case '(', '[':
			depth++
			current.WriteByte(c)
		case ')', ']':
			depth--
			current.WriteByte(c)
			if depth == 0 && c == ')' {
				// Look ahead: comma (possibly after comments/whitespace)?
				j := i + 1
				comma := false
				for j < len(section) {
					if section[j] == ',' {
						comma = true
						break
					}
					if section[j] == '-' && j+1 < len(section) && section[j+1] == '-' {
						// skip a line comment
						for j < len(section) && section[j] != '\n' {
							j++
						}
						continue
					}
					if !strings.ContainsRune(" \t\r\n", rune(section[j])) {
						break
					}
					j++
				}
				flush(comma)
				pendingComma = comma
				i = j
				continue
			}
		default:
			if depth > 0 {
				current.WriteByte(c)
			}
		}
		i++
	}
	if depth != 0 {
		return tuples, fmt.Errorf("unbalanced brackets (depth=%d)", depth)
	}
	return tuples, nil
}

var _ = fmt.Sprintf
