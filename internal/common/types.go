package common

// Regulation classifica a categoria de qualidade coberta por um sensor ou judge.
type Regulation string

const (
	RegulationMaintainability Regulation = "maintainability"
	RegulationFitness         Regulation = "fitness"
	RegulationBehaviour       Regulation = "behaviour"
)

// Severity classifica a gravidade de uma violacao.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Violation representa uma unica violacao encontrada por um sensor ou judge.
type Violation struct {
	Severity      Severity `json:"severity"`
	What          string   `json:"what"`
	Why           string   `json:"why"`
	Remediation   string   `json:"remediation"`
	FilesAffected []string `json:"filesAffected"`
	LinesAffected [][2]int `json:"linesAffected"`
	GuideURI      string   `json:"guideUri,omitempty"`
}

// SensorOutput e o envelope normalizado de saida de qualquer sensor ou judge (SPEC §9).
type SensorOutput struct {
	Tool               string      `json:"tool"`
	Regulation         Regulation  `json:"regulation"`
	Passed             bool        `json:"passed"`
	Summary            string      `json:"summary"`
	Inconclusive       bool        `json:"inconclusive"`
	InconclusiveReason string      `json:"inconclusiveReason,omitempty"`
	Violations         []Violation `json:"violations"`
}

// SensorKind classifica o tipo de sensor.
type SensorKind string

const (
	SensorKindLinter     SensorKind = "linter"
	SensorKindTypeCheck  SensorKind = "type-check"
	SensorKindStructural SensorKind = "structural"
	SensorKindCustom     SensorKind = "custom"
)
