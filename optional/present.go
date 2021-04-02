package optional

import "github.com/rhzx3519/stream/types"

/**
	present implements Optional interface which is not nil
 */
type present struct {
	value types.T
}

func (p *present) Get() types.T {
	return p.value
}

func (p *present) IsPresent() bool {
	return true
}

func (p *present) IfPresent(consumer types.Consumer) {
	consumer(p.value)
}

func (p *present) Filter(test types.Predicate) Optional {
	if test(p.value) {
		return p
	}
	return emtpy
}

func (p *present) Map(mapper types.Function) Optional {
	return OfNullable(mapper(p.value))
}

func (p *present) FlatMap(flatMapper func(t types.T) Optional) Optional {
	return flatMapper(p.value)
}

func (p *present) OrElse(t types.T) types.T {
	return p.value
}

func (p *present) OrElseGet(get types.Supplier) types.T {
	return p.value
}

func (p *present) OrPanic(panicArg interface{}) types.T {
	return p.value
}

func (p *present) OrPanicGet(getPanicArg types.Supplier) types.T {
	return p.value
}



