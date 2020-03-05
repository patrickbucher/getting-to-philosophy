package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/patrickbucher/firstlink"
)

type QueryParams struct {
	Language string `json:"language"`
	Article  string `json:"article"`
}

func (qp QueryParams) String() string {
	return fmt.Sprintf("language='%s', article='%s'", qp.Language, qp.Article)
}

type Response struct {
	FirstLink string `json:"firstLink"`
}

const requestLimit = 20

var assets = []string{"index.html", "style.css", "favicon.ico"}

func main() {
	http.HandleFunc("/csvForm", handleCSVForm)
	http.HandleFunc("/csv", handleCSV)
	http.HandleFunc("/hopcount", handleHopCount)
	http.HandleFunc("/firstlink", handleFirstlink)
	http.HandleFunc("/", handleFile("assets/index.html"))
	for _, asset := range assets {
		http.HandleFunc("/"+asset, handleFile("assets/"+asset))
	}
	port := os.Getenv("PORT")
	log.Println("listening on port", port)
	http.ListenAndServe("0.0.0.0:"+port, nil)
}

func handleCSVForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, http.StatusMethodNotAllowed, "%s not allowed", r.Method)
		return
	}
	defer r.Body.Close()
	csvFile, _, err := r.FormFile("csvFile")
	defer csvFile.Close()
	if err != nil {
		fail(w, http.StatusBadRequest, "field 'csvFile' missing: %v", err)
		return
	}
	limit := requestLimit
	if limitStr := r.FormValue("hopsLimit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			fail(w, http.StatusBadRequest, "hopsLimit='%s' as int: %v", limitStr, err)
			return
		}
	}
	if csvErr := processCSV(r, w, csvFile, uint8(limit)); csvErr != nil {
		fail(w, csvErr.httpStatus, "processing CSV from form: %v", csvErr)
	}
}

// TODO: consider accepting additional requestLimit field
func handleCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, http.StatusMethodNotAllowed, "%s not allowed", r.Method)
		return
	}
	defer r.Body.Close()
	csvReader := bufio.NewReader(r.Body)
	if csvErr := processCSV(r, w, csvReader, requestLimit); csvErr != nil {
		fail(w, csvErr.httpStatus, "processing CSV: %v", csvErr)
	}
}

// TODO: consider accepting additional requestLimit field
func handleHopCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, http.StatusMethodNotAllowed, "%s not allowed", r.Method)
		return
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(bufio.NewReader(r.Body))
	var inputRecords []firstlink.TestInputRecord
	if err := decoder.Decode(&inputRecords); err != nil {
		fail(w, http.StatusBadRequest, "decode JSON: %v", err)
		return
	}

	outputRecords := firstlink.ProcessArticleHopTests(inputRecords, requestLimit)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(&outputRecords); err != nil {
		fail(w, http.StatusInternalServerError, "encode JSON: %v", err)
		return
	}
}

func handleFirstlink(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, http.StatusMethodNotAllowed, "%s not allowed", r.Method)
		return
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(bufio.NewReader(r.Body))
	var params QueryParams
	if err := decoder.Decode(&params); err != nil {
		fail(w, http.StatusBadRequest, "decode JSON: %v", err)
		return
	}
	log.Print(params)
	link := fmt.Sprintf("https://%s.wikipedia.org/wiki/%s", params.Language, params.Article)
	firstLink, err := firstlink.FindFirstLink(link)
	if err != nil {
		fail(w, http.StatusInternalServerError, "find first link: %v", err)
		return
	}
	var resp Response
	resp.FirstLink = firstLink
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(&resp); err != nil {
		fail(w, http.StatusInternalServerError, "encode JSON: %v", err)
		return
	}
}

func handleFile(filename string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	}
}

type csvProcessingError struct {
	httpStatus int
	cause      error
}

func (err csvProcessingError) Error() string {
	if err.cause != nil {
		return err.cause.Error()
	}
	return fmt.Sprintf("http status: %d", err.httpStatus)
}

func processCSV(r *http.Request, w http.ResponseWriter, csvIn io.Reader,
	limit uint8) *csvProcessingError {
	sanitizedCSV, err := newCSVReader(bufio.NewReader(csvIn))
	if err != nil {
		return &csvProcessingError{http.StatusInternalServerError,
			fmt.Errorf("error sanitizing CSV input: %v", err)}
	}
	inputRecords, err := firstlink.InputRecordsFromCSV(sanitizedCSV)
	if err != nil {
		return &csvProcessingError{http.StatusBadRequest,
			fmt.Errorf("converting input to CSV: %v", err)}
	}
	outputRecords := firstlink.ProcessArticleHopTests(inputRecords, limit)
	report := bytes.NewBufferString("")
	if sanitizedCSV.wasBOMPolluted {
		// Bronze Rules: Abuse others as they abuse you.
		report.Write([]byte{bom1, bom2, bom3})
	}
	if err = firstlink.OutputRecordsToCSV(outputRecords, report); err != nil {
		return &csvProcessingError{http.StatusInternalServerError,
			fmt.Errorf("converting results to CSV: %v", err)}
	}
	if err = serveFile(report, w); err != nil {
		return &csvProcessingError{http.StatusInternalServerError,
			fmt.Errorf("serving file: %v", err)}
	}
	return nil
}

func serveFile(r io.Reader, w http.ResponseWriter) error {
	tempFile, err := ioutil.TempFile("", "*.csv")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	length, err := io.Copy(tempFile, r)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Disposition", "attachment; filename=report.csv")
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	tempFile.Seek(0, 0)
	_, err = io.Copy(w, tempFile)
	return err
}

// csvReader deals with Microsoft's erroneous interpretation of CSV, i.e.:
// - "comma separated values" means "semicolon separated values",
// - and UTF-8 text files should start with a byte order mark (BOM).
// csvReader.Read circumvents this misconception by discarding the leading BOM
// and replacing semicolons by commas on the fly.
type csvReader struct {
	reader         *bufio.Reader
	wasBOMPolluted bool
}

const (
	bom1                = 0xef
	bom2                = 0xbb
	bom3                = 0xbf
	bomSize             = 3
	illegalCSVSeperator = ';'
	legalCSVSeperator   = ','
)

func newCSVReader(r io.Reader) (*csvReader, error) {
	buf := bufio.NewReader(r)
	b, err := buf.Peek(bomSize)
	if err != nil {
		return nil, err
	}
	bomPolluted := false
	if b[0] == bom1 && b[1] == bom2 && b[2] == bom3 {
		buf.Discard(bomSize)
		bomPolluted = true
	}
	return &csvReader{buf, bomPolluted}, nil
}

func (r csvReader) Read(p []byte) (int, error) {
	char, n, err := r.reader.ReadRune()
	if err != nil {
		return n, err
	}
	if char == illegalCSVSeperator {
		char = legalCSVSeperator
	}
	copy(p, []byte(string(char)))
	return n, nil
}

func fail(w http.ResponseWriter, httpStatus int, format string, params ...interface{}) {
	message := fmt.Sprintf(format, params...)
	w.WriteHeader(httpStatus)
	w.Write([]byte(message))
	log.Println(message)
}
