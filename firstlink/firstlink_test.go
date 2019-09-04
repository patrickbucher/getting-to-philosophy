package firstlink

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

type fromTo struct {
	articleName   string
	firstLinkName string
}

var findFirstLinkTest = map[string][]fromTo{
	"de": {
		fromTo{"Dilbert", "Comic"},
		fromTo{"Stupidedia", "Wiki"},
		fromTo{"Wüstenkohlrabi", "Art_(Biologie)"},
		fromTo{"Völkermord_in_Ruanda", "Ruanda"},
		fromTo{"Ruanda", "Binnenstaat"},
		fromTo{"Binnenstaat", "Staat"},
		fromTo{"Staat", "Sozialwissenschaften"},
		fromTo{"Sozialwissenschaften", "Wissenschaft"},
		fromTo{"Wissenschaft", "Wissen"},
		fromTo{"Wissen", "Tatsache"},
		fromTo{"Tatsache", "Sachverhalt"},
		fromTo{"Sachverhalt", "Interdisziplinäre_Wissenschaft"},
		fromTo{"Interdisziplinäre_Wissenschaft", "Einzelwissenschaft"},
		fromTo{"Einzelwissenschaft", "Fachgebiet"},
		fromTo{"Fachgebiet", "Wissensgebiet"},
		fromTo{"Wissensgebiet", "Begriff_(Philosophie)"},
		fromTo{"Begriff_(Philosophie)", "Philosophie"},
		fromTo{"St._Pölten", "Niederösterreich"},
		fromTo{"Niederösterreich", "Bundesland_(Österreich)"},
		fromTo{"Land_(Österreich)", "Gliedstaat"},
		fromTo{"Gliedstaat", "Staat"},
		fromTo{"Miserere_(Medizin)", "Symptom"},
	},
	"en": {
		fromTo{"Eka_Lagnachi_Teesri_Goshta", "Zee_Marathi"},
		fromTo{"Zee_Marathi", "Television_channel"},
		fromTo{"Television_channel", "Channel_(broadcasting)"},
		fromTo{"Channel_(broadcasting)", "Broadcasting"},
		fromTo{"Broadcasting", "Distribution_(business)"},
		fromTo{"Distribution_(business)", "Marketing_mix"},
		fromTo{"Kurunegala_Warriors", "Cricket"},
		fromTo{"Cricket", "Bat-and-ball_games"},
		fromTo{"Bat-and-ball_games", "Playing_field"},
		fromTo{"Pitch_(sports_field)", "Sport"},
		fromTo{"Sport", "Competition"},
		fromTo{"Competition", "Goal"},
		fromTo{"Rivalry", "Competitive"},
		fromTo{"People", "Sovereign_state"},
		fromTo{"Person", "Reason"},
		fromTo{"Reason", "Consciousness"},
		fromTo{"Consciousness", "Quality_(philosophy)"},
		fromTo{"Quality_(philosophy)", "Property_(philosophy)"},
		fromTo{"Zero-sum_game", "Game_theory"},
		fromTo{"Game_theory", "Mathematical_model"},
		fromTo{"Mathematical_model", "System"},
		fromTo{"System", "Interaction"},
	},
	"fr": {
		fromTo{"Clermont-Ferrand", "Liste_des_communes_de_France_les_plus_peuplées"},
		fromTo{"Association_amicale_des_amateurs_d'andouillette_authentique", "Métier_de_bouche"},
		fromTo{"Métier_de_bouche", "Alimentation"},
		fromTo{"Alimentation", "Nourriture"},
		fromTo{"Nourriture", "Produit_d'origine_animale"},
		fromTo{"Produit_d'origine_animale", "Co-produit"},
		fromTo{"Liste_des_États_transcontinentaux", "Continent"},
		fromTo{"Continent", "Latin"},
		fromTo{"Latin", "Langues_italiques"},
		fromTo{"Langues_italiques", "Famille_de_langues"},
		fromTo{"Famille_de_langues", "Langue"},
		fromTo{"Langue", "Système"},
		fromTo{"Système", "Ensemble"},
		fromTo{"Ensemble", "Mathématiques"},
		fromTo{"Mathématiques", "Connaissance"},
		fromTo{"Connaissance", "Notion"},
		fromTo{"Notion", "Connaissance_(philosophie)"},
		fromTo{"Connaissance_(philosophie)", "Philosophie"},
	},
	"ru": {
		fromTo{"Соколов,_Гавриил_Дмитриевич", "Засосна_(Белгородская_область)"},
		fromTo{"Засосна_(Белгородская_область)", "Красногвардейский_район_(Белгородская_область)"},
		fromTo{"Толстой,_Лев_Николаевич", "Граф_(титул)"},
		fromTo{"Граф_(титул)", "Король"},
		fromTo{"Король", "Титул"},
		fromTo{"Титул", "Архаизм"},
		fromTo{"Архаизм", "Слово"},
		fromTo{"Слово", "Язык"},
		fromTo{"Язык", "Знаковая_система"},
		fromTo{"Знаковая_система_(семиотика)", "Множество"},
		fromTo{"Множество", "Математика"},
		fromTo{"Математика", "Наука"},
		fromTo{"Наука", "Объективность"},
		fromTo{"Объективность", "Объект_(философия)"},
		fromTo{"Объект_(философия)", "Категория_(философия)"},
		fromTo{"Категория_(философия)", "Обобщение_понятий"},
		fromTo{"Обобщение_понятий", "Логическая_операция"},
		fromTo{"Логическая_операция", "Логика"},
		fromTo{"Логика", "Философия"},
	},
}

func TestFindFirstLink(t *testing.T) {
	total, good, bad := 0, 0, 0
	for lang, tests := range findFirstLinkTest {
		for _, test := range tests {
			total++
			link := fmt.Sprintf(urlPatternWiki, lang, test.articleName)
			got, err := FindFirstLink(link)
			if err != nil {
				t.Errorf("FindFirstLink(%s): %v", link, err)
				bad++
				continue
			}
			expectedURL := fmt.Sprintf(urlPatternWiki, lang, test.firstLinkName)
			if got != expectedURL {
				t.Errorf("FindFirstLink(%s): expected '%s', got '%s'", link, expectedURL, got)
				bad++
				continue
			}
			good++
		}
	}
	t.Logf("FindFirstLink\ntotal:\t%d\nbad:\t%d\ngood:\t%d", total, bad, good)
}

type fromToCount struct {
	source   string
	target   string
	expected int
	err      error
}

var articleHopCountTests = map[string][]fromToCount{
	"de": {
		fromToCount{"Dilbert", "Comic", 1, nil},
		fromToCount{"Ruanda", "Philosophie", 13, nil},
		fromToCount{"Edelmetall", "Philosophie", -1, LoopError},
		fromToCount{"Zürich", "Philosophie", -1, LimitError},
	},
	"en": {
		fromToCount{"System", "Interaction", 1, nil},
		fromToCount{"Kurunegala_Warriors", "Philosophy", 10, nil},
		fromToCount{"Cheese", "Philosophy", 15, nil},
		fromToCount{"Entity", "Philosophy", -1, FirstLinkError},
	},
	"fr": {
		fromToCount{"Système", "Ensemble", 1, nil},
		fromToCount{"Continent", "Philosophie", 11, nil},
	},
	"ru": {
		fromToCount{"Король", "Титул", 1, nil},
		fromToCount{"Титул", "Философия", 14, nil},
	},
}

func TestArticleHopCount(t *testing.T) {
	var limit uint8 = 20
	for lang, tests := range articleHopCountTests {
		for _, test := range tests {
			got, err := ArticleHopCount(lang, test.source, test.target, limit)
			wrongErrorCause := err != nil && err.cause != test.err
			unexpectedError := err != nil && test.err == nil
			if wrongErrorCause || unexpectedError {
				t.Errorf("ArticleHopCount(%s, %s, %s, %d): %v",
					lang, test.source, test.target, limit, err)
				continue
			}
			if got != test.expected {
				t.Errorf("ArticleHopCount(%s, %s, %s, %d) expected %d, got %d",
					lang, test.source, test.target, limit, test.expected, got)
			}
		}
	}
}

var processArticleHopTests = []struct {
	input    []TestInputRecord
	expected []TestOutputRecord
	limit    uint8
}{
	{
		[]TestInputRecord{
			TestInputRecord{"de", "Tatsache", "Philosophie", 7},
			TestInputRecord{"en", "Cheese", "Philosophy", 8},
			TestInputRecord{"fr", "Continent", "Philosophie", 11},
			TestInputRecord{"ru", "Титул", "Философия", 10},
		},
		[]TestOutputRecord{
			TestOutputRecord{TestInputRecord{"de", "Tatsache", "Philosophie", 7}, 7, "success"},
			TestOutputRecord{TestInputRecord{"en", "Cheese", "Philosophy", 8}, 15, "failure"},
			TestOutputRecord{TestInputRecord{"fr", "Continent", "Philosophie", 11}, 11, "success"},
			TestOutputRecord{TestInputRecord{"ru", "Титул", "Философия", 10}, 14, "failure"},
		},
		20,
	},
}

func TestProcessArticleHopTests(t *testing.T) {
	for _, test := range processArticleHopTests {
		got := ProcessArticleHopTests(test.input, test.limit)
		if len(got) != len(test.expected) {
			t.Errorf("got %d records, expected %d records",
				len(got), len(test.expected))
			continue
		}
		for i, expectedRecord := range test.expected {
			if !expectedRecord.Equals(got[i]) {
				t.Errorf("record mismatch: expected %s, got %s",
					expectedRecord, got[i])
			}
		}
	}
}

type recordResult struct {
	records []TestInputRecord
	err     error
}

var inputRecordsFromCSVTests = map[string]recordResult{
	"de,Haus,Baum,3\nen,Beer,City,4\nfr,Paris,Boulevard,7\nru,Знание,Философия,2\n": recordResult{
		[]TestInputRecord{
			TestInputRecord{"de", "Haus", "Baum", 3},
			TestInputRecord{"en", "Beer", "City", 4},
			TestInputRecord{"fr", "Paris", "Boulevard", 7},
			TestInputRecord{"ru", "Знание", "Философия", 2},
		}, nil,
	},
}

func TestInputRecordsFromCSV(t *testing.T) {
	for input, output := range inputRecordsFromCSVTests {
		r := strings.NewReader(input)
		gotRecords, err := InputRecordsFromCSV(r)
		if err != output.err {
			t.Errorf("expected error %v, got error %v", output.err, err)
			continue
		}
		if len(gotRecords) != len(output.records) {
			t.Errorf("got %d records, expected %d records",
				len(gotRecords), len(output.records))
			continue
		}
		for i, expectedRecord := range output.records {
			if !expectedRecord.Equals(gotRecords[i]) {
				t.Errorf("record mismatch: expected %s, got %s",
					expectedRecord, gotRecords[i])
			}
		}
	}
}

var outputRecordsToCSVTests = []struct {
	records []TestOutputRecord
	csv     string
}{
	{[]TestOutputRecord{
		TestOutputRecord{TestInputRecord{"de", "Haus", "Baum", 3}, 3, "success"},
		TestOutputRecord{TestInputRecord{"en", "Beer", "City", 4}, 5, "failure"},
		TestOutputRecord{TestInputRecord{"fr", "Paris", "Boulevard", 7}, 7, "success"},
		TestOutputRecord{TestInputRecord{"ru", "Знание", "Философия", 2}, 9, "failure"},
	},
		"de,Haus,Baum,3,3,success\n" +
			"en,Beer,City,4,5,failure\n" +
			"fr,Paris,Boulevard,7,7,success\n" +
			"ru,Знание,Философия,2,9,failure\n",
	},
}

func TestOutputRecordsToCSV(t *testing.T) {
	for _, test := range outputRecordsToCSVTests {
		w := bytes.NewBufferString("")
		err := OutputRecordsToCSV(test.records, w)
		if err != nil {
			t.Errorf("convert records %s to CSV: %v", test.records, err)
			continue
		}
		got := w.String()
		if got != test.csv {
			t.Errorf("CSV mismatch: expected\n%s\ngot\n%s", test.csv, got)
		}
	}
}

var extractLanguageTests = []struct {
	input    string
	expected string
}{
	{"https://de.wikipedia.org/wiki/Völkermord_in_Ruanda", "de"},
	{"https://en.wikipedia.org/wiki/Tourism_in_Hungary", "en"},
	{"https://fr.wikipedia.org/wiki/De_Groeve", "fr"},
	{"https://ru.wikipedia.org/wiki/Каролина_Бранденбург-Ансбахская", "ru"},
}

func TestExtractLanguage(t *testing.T) {
	for _, test := range extractLanguageTests {
		got, err := ExtractLanguage(test.input)
		if err != nil {
			t.Errorf("ExtractLanguage(%s): %v", test.input, err)
			continue
		}
		if got != test.expected {
			t.Errorf("ExtractLanguage(%s): expected '%s', got '%s'",
				test.input, test.expected, got)
		}
	}
}

var filterChildrenTest = []struct {
	input         string
	predicate     func(n *html.Node) bool
	expectedCount int
}{
	{
		input: "<p>Paragraph <em>with</em> emphasized text.</p>",
		predicate: func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "em"
		},
		expectedCount: 1,
	},
	{
		input: `<p>Paragraph <em>with</em> emphasized <em>text</em>`,
		predicate: func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "em"
		},
		expectedCount: 2,
	},
	{
		input: `<p>Paragraph <em>with</em> emphasized <em class="foo">text</em>`,
		predicate: func(n *html.Node) bool {
			if n.Type != html.ElementNode || n.Data != "em" {
				return false
			}
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "foo" {
					return true
				}
			}
			return false
		},
		expectedCount: 1,
	},
}

func TestlFilterChildren(t *testing.T) {
	for _, test := range filterChildrenTest {
		r := strings.NewReader(test.input)
		node, err := html.Parse(r)
		if err != nil {
			t.Errorf("error parsing input '%s': %v", test.input, err)
			continue
		}
		children := FilterChildren(node, test.predicate)
		if got := len(children); got != test.expectedCount {
			t.Errorf("for input '%s', %d children expected, got %d",
				test.input, test.expectedCount, got)
		}
	}
}

var filterChildrenTerminateTest = []struct {
	input                string
	terminate, predicate func(n *html.Node) bool
	expectedCount        int
}{
	{
		input:         `<p>This <em>is</em> <span>some <em>interesting</em></span> test</p>.`,
		terminate:     func(n *html.Node) bool { return n.Data == "span" },
		predicate:     func(n *html.Node) bool { return n.Data == "em" },
		expectedCount: 1, // <em> within <span> must not be counted
	},
}

func TestFilterChildrenTerminate(t *testing.T) {
	for _, test := range filterChildrenTerminateTest {
		r := strings.NewReader(test.input)
		node, err := html.Parse(r)
		if err != nil {
			t.Errorf("error parsing input '%s': %v", test.input, err)
		}
		children := FilterChildrenTerminate(node, test.terminate, test.predicate)
		if got := len(children); got != test.expectedCount {
			t.Errorf("for input '%s', %d children expected, got %d",
				test.input, test.expectedCount, got)
		}
	}
}

var removeParensTests = []struct {
	input    string
	expected string
}{
	{"Das (ist) ein Test.", "Das  ein Test."},
	{"Das (ist) [ein] Test.", "Das   Test."},
	{"Das (ist ein Test.", "Das (ist ein Test."},
	{"Das ist) ein Test.", "Das ist) ein Test."},
	{"Das )ist( ein ]Test[.", "Das )ist( ein ]Test[."},
	{"Das (ist) ein (Test).", "Das  ein ."},
	{"(Das) hier [ist] noch (ein) weiterer [Test].", " hier  noch  weiterer ."},
	{"Das (ist (wirklich) ein) Test.", "Das  Test."},
	{"Das (ist [wirklich] ein) Test.", "Das  Test."},
	{"Das (ist [wirklich (ein schwieriger)]) Test.", "Das  Test."},
	{"Eine Grenze (Lehnwort aus dem Altpolnischen,[1] vgl. altslawisch, (alt-)polnisch granica „Grenze“, Abkürzungen: Gr. und Grz.) ist der Rand eines Raumes und damit ein Trennwert, eine Trennlinie oder eine Trennfläche.", "Eine Grenze  ist der Rand eines Raumes und damit ein Trennwert, eine Trennlinie oder eine Trennfläche."},
}

func TestRemoveParens(t *testing.T) {
	for _, test := range removeParensTests {
		got := RemoveParens(test.input)
		if got != test.expected {
			t.Errorf("expected\n%s\ngot\n%s", test.expected, got)
		}
	}
}

var renderHTMLTests = []struct {
	input    string
	expected string
}{
	{"<p>This is a test.</p>", "This is a test."},
	{`<p>This <em>is</em> <span class="foo">a</a> test.</p>`, "This is a test."},
	{`<div>Test with</div><p>two blocks.</p>`, `Test with two blocks.`},
}

func TestRenderHTML(t *testing.T) {
	for _, test := range renderHTMLTests {
		r := strings.NewReader(test.input)
		node, err := html.Parse(r)
		if err != nil {
			t.Errorf("parse HTML '%s': %v", test.input, err)
			continue
		}
		got := RenderHTML(node)
		if got != test.expected {
			t.Errorf("render HTML: expected '%s', got '%s'", test.expected, got)
		}
	}
}

var extractLinkNameTests = []struct {
	input    string
	expected string
}{
	{"<a>Whatever</a>", "Whatever"},
	{`<a href="https://whatev.er">Whatever</a>`, "Whatever"},
	{`<a href="https://whatev.er">What <em>ever</em></a>`, "What ever"},
	{`<a href="https://whatev.er">What <em>do</em> you <em>mean?</em></a>`, "What do you mean?"},
	{`<a href="/wiki/Liste_des_communes_de_France_les_plus_peupl%C3%A9es" title="Liste des communes de France les plus peuplées">ville</a>`, "ville"},
}

func TestExtractLinkName(t *testing.T) {
	for _, test := range extractLinkNameTests {
		r := strings.NewReader(test.input)
		node, err := html.Parse(r)
		if err != nil {
			t.Errorf("parse HTML '%s': '%v'", test.input, err)
			continue
		}
		a := FilterChildren(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a"
		})
		got, err := ExtractLinkName(a[0])
		if err != nil {
			t.Errorf("extract link name '%s': '%v'", test.input, err)
			continue
		}
		if got != test.expected {
			t.Errorf("extract link name: expected '%s', got '%s'", test.expected, got)
		}
	}
}

var extractLinkNameURLTests = []struct {
	input    string
	expected string
}{
	{"https://de.wikipedia.org/wiki/Weck,_Worscht_un_Woi",
		"Weck,_Worscht_un_Woi"},
	{"https://en.wikipedia.org/wiki/Farallón_Negro",
		"Farallón_Negro"},
	{"https://fr.wikipedia.org/wiki/Augusta_de_Saxe-Weimar-Eisenach",
		"Augusta_de_Saxe-Weimar-Eisenach"},
	{"https://ru.wikipedia.org/wiki/Яффе,_Лев_Борисович",
		"Яффе,_Лев_Борисович"},
}

func TestExtractLinkNameURL(t *testing.T) {
	for _, test := range extractLinkNameURLTests {
		got, err := ExtractLinkNameURL(test.input)
		if err != nil {
			t.Errorf("extract link from URL '%s': %v", test.input, err)
		}
		if got != test.expected {
			t.Errorf("extract link from URL '%s', expected '%s', got '%s'",
				test.input, test.expected, got)
		}
	}
}

var isSubSliceOfTests = []struct {
	slice     []string
	subslice  []string
	contained bool
}{
	{[]string{}, []string{}, true},
	{[]string{"foo"}, []string{}, true},
	{[]string{}, []string{"foo"}, false},
	{[]string{"foo"}, []string{"foo"}, true},
	{[]string{"foo", "bar", "baz"}, []string{"foo", "bar"}, true},
	{[]string{"foo", "bar", "baz"}, []string{"bar", "baz"}, true},
	{[]string{"foo", "bar", "baz"}, []string{"foo", "bar", "baz"}, true},
	{[]string{"foo", "bar", "baz", "qux"}, []string{"bar", "baz"}, true},
}

func TestIsSubSliceOf(t *testing.T) {
	for _, test := range isSubSliceOfTests {
		got := IsSubSliceOf(test.subslice, test.slice)
		if got != test.contained {
			t.Errorf("IsSubSliceOf(%q, %q) expected %v, got %v",
				test.subslice, test.slice, test.contained, got)
		}
	}
}
