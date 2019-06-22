package firstlink

import "fmt"

// TestInputRecord describes the parameters to run a article hop count test.
type TestInputRecord struct {
	Lang     string `json:"lang"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Expected int    `json:"expected"`
}

func (r TestInputRecord) Equals(other TestInputRecord) bool {
	return r.Lang == other.Lang &&
		r.Source == other.Source &&
		r.Target == other.Target &&
		r.Expected == other.Expected
}

func (r TestInputRecord) String() string {
	return fmt.Sprintf(`Lang="%s", Source="%s", Target="%s", Expected="%d"`,
		r.Lang, r.Source, r.Target, r.Expected)
}

// TestOutputRecord describes the input parameters of a hop count test combined
// with its Result.
type TestOutputRecord struct {
	TestInputRecord
	Actual int    `json:"actual"`
	Result string `json:"result"`
}

func (r TestOutputRecord) Equals(other TestOutputRecord) bool {
	return r.Lang == other.Lang &&
		r.Source == other.Source &&
		r.Target == other.Target &&
		r.Expected == other.Expected &&
		r.Actual == other.Actual &&
		r.Result == other.Result
}

func (r TestOutputRecord) String() string {
	return fmt.Sprintf(`Lang="%s", Source="%s", Target="%s", Expected="%d, `+
		`"Actual="%d", Result="%s"`,
		r.Lang, r.Source, r.Target, r.Expected, r.Actual, r.Result)
}

func (r TestOutputRecord) EqualInput(input TestInputRecord) bool {
	return r.Lang == input.Lang &&
		r.Source == input.Source &&
		r.Target == input.Target &&
		r.Expected == input.Expected
}
