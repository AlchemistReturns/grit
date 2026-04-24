package prompt

type Question struct {
	Text       string
	TimeoutSec int
}

type InterviewResult struct {
	Answers []Result
}

// Run asks each question in sequence and returns all results.
func Run(questions []Question) InterviewResult {
	var ir InterviewResult
	for _, q := range questions {
		r := Ask(q.Text, q.TimeoutSec)
		ir.Answers = append(ir.Answers, r)
	}
	return ir
}
