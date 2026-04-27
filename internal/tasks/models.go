package tasks

type Example struct {
	Title       string  `json:"title"`
	Input       string  `json:"input"`
	Output      string  `json:"output"`
	Explanation *string `json:"explanation,omitempty"`
}

type TestCase struct {
	Name           string `json:"name"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expectedOutput"`
	Locked         bool   `json:"locked,omitempty"`
}

type Challenge struct {
	ID                    string            `json:"id"`
	TaskID                string            `json:"taskId"`
	Slug                  string            `json:"slug"`
	Title                 string            `json:"title"`
	Difficulty            string            `json:"difficulty"`
	Summary               string            `json:"summary"`
	Description           []string          `json:"description"`
	Examples              []Example         `json:"examples"`
	Constraints           []string          `json:"constraints"`
	Notes                 []string          `json:"notes,omitempty"`
	Tags                  []string          `json:"tags"`
	Category              string            `json:"category"`
	Kind                  string            `json:"kind"`
	SupportedLanguages    []string          `json:"supportedLanguages"`
	LanguageVersions      map[string]string `json:"languageVersions"`
	StarterCodeByLanguage map[string]string `json:"starterCodeByLanguage"`
	AcceptanceRate        int               `json:"acceptanceRate"`
	EstimatedMinutes      int               `json:"estimatedMinutes"`
	Attempts              int               `json:"attempts"`
	Popularity            int               `json:"popularity"`
	CreatedAt             string            `json:"createdAt"`
	Progress              string            `json:"progress"`
	TestCases             []TestCase        `json:"testCases"`
}

type ChallengeListItem struct {
	ID                 string   `json:"id"`
	TaskID             string   `json:"taskId"`
	Slug               string   `json:"slug"`
	Title              string   `json:"title"`
	Difficulty         string   `json:"difficulty"`
	Summary            string   `json:"summary"`
	Tags               []string `json:"tags"`
	Category           string   `json:"category"`
	Kind               string   `json:"kind"`
	SupportedLanguages []string `json:"supportedLanguages"`
	AcceptanceRate     int      `json:"acceptanceRate"`
	EstimatedMinutes   int      `json:"estimatedMinutes"`
	Attempts           int      `json:"attempts"`
	Popularity         int      `json:"popularity"`
	CreatedAt          string   `json:"createdAt"`
	Progress           string   `json:"progress"`
}

