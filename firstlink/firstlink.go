package firstlink

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

const (
	urlPattern     = "https://%s.wikipedia.org%s"
	urlPatternWiki = "https://%s.wikipedia.org/wiki/%s"
)

// FindFirstLink finds the first link from a given Wikipedia article to another article.
func FindFirstLink(link string) (string, error) {
	lang, err := ExtractLanguage(link)
	if err != nil {
		return "", fmt.Errorf("extract language from %s: %v", link, err)
	}
	resp, err := http.Get(link)
	if err != nil {
		return "", fmt.Errorf("get '%s': %v", link, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get '%s': %s", link, resp.Status)
	}
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parse HTML of %s: %v", link, err)
	}
	paragraphs := FilterChildrenTerminate(doc, isTablePredicate, isParagraphPredicate)
	var links []*html.Node
	for _, p := range paragraphs {
		paragraphLinks := FilterChildren(p, isWikiLinkPredicate)
		if len(paragraphLinks) == 0 {
			continue
		}
		paragraphText := RenderHTML(p)
		noParensText := RemoveParens(paragraphText)
		for _, paragraphLink := range paragraphLinks {
			linkName, err := ExtractLinkName(paragraphLink)
			if err != nil {
				continue
			}
			paragraphWords := strings.Fields(noParensText)
			linkWords := strings.Fields(linkName)
			if len(linkWords) == 0 {
				continue
			}
			contained := IsSubSliceOf(linkWords, paragraphWords)
			if contained {
				links = append(links, paragraphLink)
				break
			}
		}
		if len(links) == 0 {
			continue // next paragraph
		}
		hrefs := extractHrefs(links)
		hrefs = removeWithPrefixes(hrefs)
		if len(hrefs) == 0 {
			continue
		}
		// at least one link was found, finish here
		newLink, err := url.QueryUnescape(hrefs[0])
		if err != nil {
			return "", fmt.Errorf("error unescaping %s: %v", hrefs[0], err)
		}
		return fmt.Sprintf(urlPattern, lang, newLink), nil
	}
	return "", fmt.Errorf("unable to extract first link of %s", link)
}

type ArticleHopCountError struct {
	message string
	cause   error
}

func (err *ArticleHopCountError) Error() string {
	return fmt.Sprintf("%v: %s", err.cause, err.message)
}

var LoopError = errors.New("loop detected")
var LimitError = errors.New("limit reached")
var FirstLinkError = errors.New("first link not found")

// ArticleHopCount counts the number of first article link references needed to
// get from the source to the target article. If the limit is reached, an error
// and -1 is returned.
func ArticleHopCount(lang, source, target string, limit uint8) (int, *ArticleHopCountError) {
	visited := make(map[string]bool)
	path := make([]string, 0)
	sourceURL := fmt.Sprintf(urlPatternWiki, lang, source)
	targetURL := fmt.Sprintf(urlPatternWiki, lang, target)
	for hops := 0; hops < int(limit); hops++ {
		visited[sourceURL] = true
		path = append(path, sourceURL)
		firstLink, err := FindFirstLink(sourceURL)
		if err != nil {
			message := fmt.Sprintf("find first link: %v", err)
			return -1, &ArticleHopCountError{message, FirstLinkError}
		}
		if _, seen := visited[firstLink]; seen {
			// loop detected, reporting path
			path = append(path, firstLink) // about to visit
			loopPath := make([]string, 0)
			for _, link := range path {
				linkName, err := ExtractLinkNameURL(link)
				if err != nil {
					loopPath = append(loopPath, link)
				} else {
					loopPath = append(loopPath, linkName)
				}
			}
			return -1, &ArticleHopCountError{strings.Join(loopPath, " -> "), LoopError}
		}
		if targetURL == firstLink {
			// last link is not opened, hypothetical hop needed
			return hops + 1, nil
		}
		sourceURL = firstLink
	}
	message := fmt.Sprintf("stopped after %d hops without reaching target", limit)
	return -1, &ArticleHopCountError{message, LimitError}
}

// ProcessArticleHopTests processes the test cases from the input using the
// ArticleHopCount function and returns the result report.
func ProcessArticleHopTests(input []TestInputRecord, limit uint8) []TestOutputRecord {
	var outputRecords []TestOutputRecord
	results := make(chan TestOutputRecord)
	var wg sync.WaitGroup
	consume := func(ch <-chan TestOutputRecord) {
		for out := range ch {
			outputRecords = append(outputRecords, out)
			wg.Done()
		}
	}
	produce := func(in TestInputRecord, ch chan<- TestOutputRecord) {
		hops, err := ArticleHopCount(in.Lang, in.Source, in.Target, limit)
		var result string
		if err != nil {
			result = fmt.Sprintf("error: %v", err)
		} else if hops == in.Expected {
			result = "success"
		} else {
			result = "failure"
		}
		out := TestOutputRecord{in, hops, result}
		ch <- out
	}

	go consume(results)
	for _, in := range input {
		wg.Add(1)
		go produce(in, results)
	}
	wg.Wait()
	close(results)

	// re-establish original order
	var outputOrdered []TestOutputRecord
	for _, in := range input {
		for _, out := range outputRecords {
			if out.EqualInput(in) {
				outputOrdered = append(outputOrdered, out)
				break
			}
		}
	}
	return outputOrdered
}

// InputRecordsFromCSV converts the CSV data behind the given reader with the
// record structure of lang,source,target,expected into a slice of test input
// records.
func InputRecordsFromCSV(r io.Reader) ([]TestInputRecord, error) {
	var records []TestInputRecord
	csvReader := csv.NewReader(r)
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading csv: %v", err)
		}
		if len(rec) != 4 {
			return nil, fmt.Errorf("missing fields in %q", rec)
		}
		expected, err := strconv.Atoi(rec[3])
		if err != nil {
			return nil, fmt.Errorf("fourth column must be numeric, was %s", rec[3])
		}
		records = append(records, TestInputRecord{rec[0], rec[1], rec[2], expected})
	}
	return records, nil
}

// OutputRecordsToCSV converts the records slice to the CSV format using the
// record structure lang,source,target,expected,actual,result
func OutputRecordsToCSV(records []TestOutputRecord, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	for _, record := range records {
		if err := csvWriter.Write([]string{record.Lang, record.Source,
			record.Target, strconv.Itoa(record.Expected),
			strconv.Itoa(record.Actual), record.Result}); err != nil {
			return fmt.Errorf("writing %s as CSV record: %v", record, err)
		}
	}
	csvWriter.Flush()
	return nil
}

// ExtractLanguage extracts the language code of a Wikipedia article URL.
// I.e. for https://de.wikipedia.org/wiki/Whatever, the language code is de.
func ExtractLanguage(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("unable to Parse URL %s: %v", link, err)
	}
	hostname := u.Hostname()
	dotAt := strings.Index(hostname, ".")
	if dotAt == -1 {
		return "", fmt.Errorf("URL format must be https://[lang].wikipedia.org/")
	}
	return hostname[:dotAt], nil
}

// FilterChildren filters the children of Node n recursively by applying the
// given predicate function.
func FilterChildren(n *html.Node, predicate func(n *html.Node) bool) []*html.Node {
	return FilterChildrenTerminate(n, func(n *html.Node) bool { return false }, predicate)
}

// FilterChildrenTerminate filters the children of Node n recursively by
// applying the given predicate function. The children of a node the terminate
// predicate applies to will not be entered.
func FilterChildrenTerminate(n *html.Node,
	terminate, predicate func(n *html.Node) bool) []*html.Node {
	var nodes []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if predicate(c) {
			nodes = append(nodes, c)
		}
		if !terminate(c) {
			nodes = append(nodes, FilterChildrenTerminate(c, terminate, predicate)...)
		}
	}
	return nodes
}

// RemoveParens removes all parentheses and square brackets with their content.
func RemoveParens(paragraph string) string {
	var parens = []struct {
		opening rune
		closing rune
	}{
		{'(', ')'},
		{'[', ']'},
	}
	for _, parenPair := range parens {
		for {
			from := strings.Index(paragraph, string(parenPair.opening))
			to := strings.Index(paragraph, string(parenPair.closing))
			if from == -1 || to == -1 || from > to {
				// no more enclosing pairs
				break
			}
			// nested parens: cut "foo (bar (baz) qux)" after "qux)", not "baz)"
			var open int
			var temp []rune
			runes := []rune(paragraph)
			firstOpening := -1
			for pos, char := range runes {
				if char == parenPair.opening {
					open++
					if firstOpening == -1 {
						firstOpening = pos
					}
				} else if char == parenPair.closing && open > 0 {
					open--
				}
				if open == 0 && firstOpening != -1 {
					temp = append(temp, runes[:firstOpening]...)
					temp = append(temp, runes[pos+1:]...)
					break
				}
			}
			paragraph = string(temp)
		}
	}
	return paragraph
}

// RenderHTML renders the given HTML node as plain text.
func RenderHTML(n *html.Node) string {
	var chunks = make([]string, 0)
	return renderHTML(n, &chunks)
}

func renderHTML(n *html.Node, chunks *[]string) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			text := strings.TrimSpace(c.Data)
			if text != "" {
				*chunks = append(*chunks, text)
			}
		case html.DocumentNode:
			fallthrough
		case html.ElementNode:
			if c.Data != "style" {
				renderHTML(c, chunks)
			}
		}
	}
	return strings.Join(*chunks, " ")
}

// ExtractLinkName extracts the link name of an <a> tag.
// Example: <a href="foo.com">Foo</a> yields "Foo"
func ExtractLinkName(a *html.Node) (string, error) {
	if a.Type != html.ElementNode || a.Data != "a" {
		return "", fmt.Errorf("the node <%s> is not a link tag", a.Data)
	}
	return RenderHTML(a), nil
}

// ExtractLinkNameURL extracts the article name from a Wikipedia article URL,
// i.e. "https://en.wikipedia.org/wiki/Computer" becomes "Computer"
func ExtractLinkNameURL(articleURL string) (string, error) {
	pattern := regexp.MustCompile(`https://[a-z]{2}.wikipedia.org/wiki/(.+)`)
	matches := pattern.FindStringSubmatch(articleURL)
	if len(matches) != 2 {
		return "", fmt.Errorf("extract link name of %s, got %d matches",
			articleURL, len(matches))
	}
	return matches[1], nil
}

// IsSubSliceOf tests whether or not subslice is contained in slice.
func IsSubSliceOf(subslice, slice []string) bool {
	if len(subslice) == 0 {
		// an empty subslice is contained in every slice
		return true
	}
	if len(slice) == 0 {
		// slice is empty, but subslice wasn't
		return false
	}
	for i := 0; i < len(slice); i++ {
		for j := 0; j < len(subslice); j++ {
			if slice[i+j] != subslice[j] {
				break
			}
			if j == len(subslice)-1 {
				return true
			}
		}
	}
	return false
}

func extractHrefs(links []*html.Node) []string {
	var extracted = make([]string, 0)
	for _, link := range links {
		for _, attr := range link.Attr {
			if attr.Key == "href" {
				extracted = append(extracted, attr.Val)
			}
		}
	}
	return extracted
}

func removeWithPrefixes(links []string) []string {
	var filtered = make([]string, 0)
	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil {
			continue
		}
		// filter out prefixed links, such as /File:flag.jpg
		if strings.Contains(u.Path, ":") {
			continue
		}
		filtered = append(filtered, link)
	}
	return filtered
}

func isParagraphPredicate(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "p"
}

func isTablePredicate(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "table"
}

func isWikiLinkPredicate(n *html.Node) bool {
	if n.Data != "a" {
		return false
	}
	for _, attr := range n.Attr {
		if attr.Key == "href" && strings.HasPrefix(attr.Val, "/wiki/") &&
			!strings.Contains(attr.Val, "#") {
			return true
		}
	}
	return false
}
