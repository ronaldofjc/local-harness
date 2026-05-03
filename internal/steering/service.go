package steering

// Service orquestra operacoes de steering.
type Service struct {
	log       *Log
	suggester *Suggester
}

// NewService cria um novo service de steering.
func NewService(log *Log) *Service {
	return &Service{
		log:       log,
		suggester: NewSuggester(log),
	}
}

// Log expoe o log para append externo (ex: apos sensor.run ou judge.record).
func (s *Service) Log() *Log {
	return s.log
}

// Suggest delega para o suggester.
func (s *Service) Suggest(input SuggestInput) (*SuggestOutput, error) {
	return s.suggester.Suggest(input)
}
