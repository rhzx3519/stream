package stream

import "github.com/rhzx3519/stream/types"

// stage 记录一个**操作**
// Begin 用于操作开始，参数是元素的个数，如果个数不确定，则是 unknownSize
// Accept 接收每个元素
// CanFinish 用于判断是否可以提前结束
// End 是收尾动作
type stage interface {
	Begin(size int64)
	Accept(t types.T)
	CanFinish() bool
	End()
}

// region baseStage

type baseStage struct {
	begin func(int64) 		// begin(size)
	action types.Consumer 	// aciton(t)
	canFinish func() bool 	// canFinish() bool
	end func()				// end()
}

func (b *baseStage) Begin(size int64) {
	b.begin(size)
}

func (b *baseStage) Accept(t types.T) {
	b.action(t)
}

func (b *baseStage) CanFinish() bool {
	return b.canFinish()
}

func (b *baseStage) End() {
	b.end()
}

// option is a function which input parameter is a baseStage pointer
type option func(b *baseStage)

func begin(onBegin func(int64)) option {
	return func(b *baseStage) {
		b.begin = onBegin
	}
}

func canFinish(judge func() bool) option {
	return func(b *baseStage) {
		b.canFinish = judge
	}
}

func action(onAction types.Consumer) option {
	return func(b *baseStage) {
		b.action = onAction
	}
}

func end(onEnd func()) option {
	return func(b *baseStage) {
		b.end = onEnd
	}
}

// end region baseStage


// region chainedStage
// chainedStage 串起下一个操作
type chainedStage struct {
	*baseStage
}

func defaultChainedStage(down stage) *chainedStage {
	return &chainedStage{
		&baseStage{
			begin: down.Begin,
			action: down.Accept,
			canFinish: down.CanFinish,
			end: down.End,
		},
	}
}

func newChainedStage(down stage, opt ...option) *chainedStage {
	s := defaultChainedStage(down)
	for _, o := range opt {
		o(s.baseStage)
	}
	return s
}

// end region chainedStage


// region terminalStage
// terminalStage 代表终结操作
type terminalStage struct {
	*baseStage
}

func defaultTerminalStage(action types.Consumer) *terminalStage {
	return &terminalStage{
		&baseStage{
			begin: func(int64) {},
			action: action,
			canFinish: func() bool { return false },
			end: func() {},
		},
	}
}

/**
parameter: option 用于绑定stage的begin, action, canFinish, end方法
 */
func newTerminalStage(action types.Consumer, opt ...option) *terminalStage {
	s := defaultTerminalStage(action)
	for _, o := range opt {
		o(s.baseStage)
	}
	return s
}

// end region terminalStage

