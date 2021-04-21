package saga

type Context struct {
	State         map[string]interface{}
	Type          string
	CorrelationId string
	IsCompleted   bool
	store         Store
	session       Session
}

func CreateContext(store Store, session Session) *Context {
	return &Context{
		store:   store,
		session: session,
	}
}

//Commit changes and unlock saga
func (ctx *Context) Done() {
	err := ctx.session.Commit()
	if err != nil {
		panic(err)
	}
	ctx.session.Close()
}

//Complete the saga
func (ctx *Context) Complete() {
	err := ctx.store.CompleteSaga(ctx.session, ctx.CorrelationId, ctx.Type)
	if err != nil {
		panic(err)
	}
}

//Save state to the saga
func (ctx *Context) Save() {
	err := ctx.store.UpdateState(ctx.session, ctx.CorrelationId, ctx.Type, ctx.State)
	if err != nil {
		panic(err)
	}
}
