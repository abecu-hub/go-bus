package saga

type Session interface {
	Commit() error
	Close()
}

type Store interface {
	SagaExists(correlationId string, sagaType string) (bool, error)
	RequestSaga(correlationId string, sagaType string) (*Context, error)
	CreateSaga(correlationId string, sagaType string) error
	UpdateState(session Session, correlationId string, sagaType string, state map[string]interface{}) error
	CompleteSaga(session Session, correlationId string, sagaType string) error
	DeleteSaga(correlationId string, sagaType string) error
}
