package agreement 

type Factory struct {
	publisher eventbus.Publisher
	keys user.Keys
	requestStepUpdate func()
}

func NewFactory(publisher eventbus.Publisher, keys user.Keys, requestStepUpdate func()) *Factory  {
	return &Factory{
		publisher,
		keys,
		requestStepUpdate,
	}
}

func (f *Factory) Instantiate() Component {
	return newComponent(f.publisher, keys)
}
