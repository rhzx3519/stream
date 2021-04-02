package stream

import (
	"github.com/rhzx3519/stream/optional"
	"github.com/rhzx3519/stream/types"
	"reflect"
	"sort"
)

// stream is a node show as below. which source is a iterator. head stream has no prev node.
// terminal operate create a terminalStage,
// then this terminalStage will use a downStage of prev node and wrap a new stage,
// finally trigger source to iterate its data to the wrappedStage
//
// stream 是数据流中的一个节点，见下图。头节点没有前驱节点。每个操作会创建一个新的节点，连接到原有节点后。
// 终止操作会创建一个 terminalStage, 这个 terminalStage 会作为最后一个节点的 downStage, 依次往前调用 wrap 方法，生成最终的 wrappedStage
// 最后触发数据源迭代每个元素给 wrappedStage
//
//            head filter   map   for-each
//            +--+    +---+    +--+
//     nil <- |  | <- |   | <- |  | <- terminalStage
//            +--+    +---+    +--+
//
//                +-filter----------------+
//  source -->    |                       |
//                |       +-map-----------+
//                |       |               |
//                |       |    +-for-each-+
//                |       |    |          | terminalStage
//                +-------+----+----------+
//
//               <----- wrapped stage ----->
type stream struct {
	source iterator
	prev   *stream
	wrap   func(stage) stage
}

// region help methods

// 构造头节点
func newHead(source iterator) *stream {
	return &stream{source: source}
}

// 构造中间节点
func newNode(prev *stream, wrap func(stage) stage) *stream {
	return &stream{
		source: prev.source,
		prev: prev,
		wrap: wrap,
	}
}

// 终止节点通过调用terminal方法，
// 1. 生成并传入 terminalStage
// 2. 打包所有流操作
// 3. 依次遍历所有元素
func (s *stream) terminal(ts *terminalStage) {
	stage := s.wrapStage(ts) // 返回的stage是一个操作集合，即 stage1->stage2->...stage n
	source := s.source
	stage.Begin(source.GetSizeIfKnown())
	for source.HasNext() && !stage.CanFinish() {
		stage.Accept(source.Next())
	}
	stage.End()
}

// 从终止节点往回调用每一个节点(stream)的wrap方法，将所有操作都打包成一个操作(stage)
func (s *stream) wrapStage(terminalStage stage) stage {
	stage := terminalStage
	for i := s; i.prev != nil; i = i.prev {
		stage = i.wrap(stage)
	}
	return stage
}

// end region help methods

// region stateless operate 无状态操作

// 过滤操作, down stage 指代下一个操作
func (s *stream) Filter(test types.Predicate) Stream {
	return newNode(s, func(down stage) stage {
		return newChainedStage(down, begin(func(int64) {
			down.Begin(unkonwnSize)
		}), action(func(t types.T) {
			if test(t) {
				down.Accept(t)
			}
		}))
	})
}


// Map 转换操作
// apply is a Function, convert the element to another 转换元素
func (s *stream) Map(apply types.Function) Stream {
	return newNode(s, func(down stage) stage {
		return newChainedStage(down, action(func(t types.T) {
			down.Accept(apply(t))
		}))
	})
}


// FlatMap 打平集合为元素。[[1,2],[3,4]] -> [1,2,3,4]
func (s *stream) FlatMap(flatten func(types.T) Stream) Stream {
	return newNode(s, func(down stage) stage {
		return newChainedStage(down, begin(func(int64) {
				down.Begin(unkonwnSize)
			}), action(func(t types.T) {
				ss := flatten(t)		// 元素是集合, 转化为流
				ss.ForEach(down.Accept) // 依次消费流中的数据
		}))
	})
}

// Peek visit every element and leave them on stream so that they can be operated by next action  访问流中每个元素而不消费它，可用于 debug
func (s *stream) Peek(consumer types.Consumer) Stream {
	return newNode(s, func(down stage) stage {
		return newChainedStage(down, action(func(t types.T) {
			consumer(t)
			down.Accept(t)
		}))
	})
}

// end region stateless operate

// region stateful operate 有状态操作

// Distinct remove duplicate 去重操作
// distincter is a IntFunction, which return a int hashcode to identity each element 返回元素的唯一标识用于区分每个元素
func (s *stream) Distinct(distincter types.IntFunction) Stream {
	return newNode(s, func(down stage) stage {
		var set map[int]bool
		return newChainedStage(down, begin(func(int64) {
			set = make(map[int]bool)
			down.Begin(unkonwnSize)		// 去重后的个数不确定
		}), action(func(t types.T) {
			hash := distincter(t)
			if _, ok := set[hash]; !ok { // 唯一的元素才往下游发送
				set[hash] = true
				down.Accept(t)
			}
		}), end(func() {
			set = nil
			down.End()
		}))
	})
}

// Sorted sort by Comparator 排序
func (s *stream) Sorted(comparator types.Comparator) Stream {
	return newNode(s, func(down stage) stage {
		var list []types.T
		return newChainedStage(down, begin(func(size int64) {
			if size > 0 {
				list = make([]types.T, 0, size) // 返回一个length=0, cap=size的slice
			} else {
				list = make([]types.T, 0)
			}
			down.Begin(size)
		}), action(func(t types.T) {
			list = append(list, t)
		}), end(func() {
			a := &Sortable{
				List: list,
				Cmp: comparator,
			}
			sort.Sort(a)
			down.Begin(int64(len(a.List)))
			i := it(a.List...)
			for i.HasNext() && !down.CanFinish() {
				down.Accept(i.Next())
			}
			list = nil
			a = nil
			down.End()
		}))
	})
}

// Limit 限制元素个数
func (s *stream) Limit(maxSize int64) Stream {
	return newNode(s, func(down stage) stage {
		count := int64(0)
		return newChainedStage(down, begin(func(size int64) {
			if size > 0 && size > maxSize {
				size = maxSize
			}
			down.Begin(size)
		}), action(func(t types.T) {
			if count < maxSize {
				down.Accept(t)
			}
			count++
		}), canFinish(func() bool {
			return count >= maxSize		// 已经到了限制数量，就可以提前结束了
		}))
	})
}

// SKip 跳过指定个数的元素
func (s *stream) Skip(n int64) Stream {
	return newNode(s, func(down stage) stage {
		count := int64(0)
		return newChainedStage(down, begin(func(size int64) {
			if size > 0 {
				size -= n
				if size < 0 {
					size = 0
				}
			}
			down.Begin(size)
		}), action(func(t types.T) {
			if count >= n {
				down.Accept(t)
			}
			count++
		}))
	})
}

// end region stateful operate 有状态操作

// region terminate operate 终止操作
// ForEach消费流中的每个元素
func (s *stream) ForEach(consumer types.Consumer) {
	s.terminal(newTerminalStage(consumer))
}

func (s *stream) ToSlice() []types.T {
	return s.ReduceBy(func(count int64) types.R {
		if count >= 0 {
			return make([]types.T, 0, count)
		}
		return make([]types.T, 0)
	}, func(acc types.R, e types.T) types.R {
		slice := acc.([]types.T)
		slice = append(slice, e)
		return slice
	}).([]types.T)
}

// ToElementSlice needs a argument cause the stream may be empty
func (s *stream) ToElementSlice(some types.T) types.R {
	return s.ToSliceOf(reflect.TypeOf(some))
}

// ToRealSlice
func (s *stream) ToSliceOf(typ reflect.Type) types.R {
	sliceType := reflect.SliceOf(typ)	// 返回类型typ对应的切片类型
	return s.ReduceBy(func(size int64) types.R {
		if size >= 0 {
			return reflect.MakeSlice(sliceType, 0, int(size))
		}
		return reflect.MakeSlice(sliceType, 0, 16)
	}, func(acc types.R, e types.T) types.R {
		sliceValue := acc.(reflect.Value)
		sliceValue = reflect.Append(sliceValue, reflect.ValueOf(e))
		return sliceValue
	}).(reflect.Value).Interface()
}

func (s *stream) Reduce(accumulator types.BinaryOperator) optional.Optional {
	var result types.T = nil
	var hasElement = false
	s.terminal(newTerminalStage(func(t types.T) {
		if !hasElement {
			result = t
			hasElement = true
		} else {
			result = accumulator(result, t)
		}
	}))

	return optional.OfNullable(result)
}

// ReduceFrom 从给定的初始值 initValue(类型和元素类型相同) 开始迭代 使用 accumulator(2个入参类型和返回类型相同) 累计结果
func (s *stream) ReduceFrom(initValue types.T, accumulator types.BinaryOperator) types.T {
	var result = initValue
	s.terminal(newTerminalStage(func(t types.T) {
		result = accumulator(result, t)
	}))

	return result
}

// ReduceWith 使用给定的初始值 initValue(类型和元素类型不同) 开始迭代 使用 accumulator( R + T -> R) 累计结果
func (s *stream) ReduceWith(initValue types.R, accumulator func(acc types.R, e types.T) types.R) types.R {
	var result = initValue
	s.terminal(newTerminalStage(func(t types.T) {
		result = accumulator(result, t)
	}))

	return result
}

// ReduceBy 使用给定的初始化方法(参数是元素个数，或-1)生成 initValue, 然后使用 accumulator 累计结果
// ReduceBy use `buildInitValue` to build the initValue, which parameter is a int64 means element size, or -1 if unknown size.
// Then use `accumulator` to add each element to previous result
func (s *stream) ReduceBy(buildInitValue func(int64) types.R, accumulator func(acc types.R, e types.T) types.R) types.R {
	var result types.R
	s.terminal(newTerminalStage(func(t types.T) {
		result = accumulator(result, t)
	}, begin(func(count int64) {
		result = buildInitValue(count)
	})))

	return result
}

func (s *stream) FindFirst() optional.Optional {
	var result types.T = nil
	var find = false
	s.terminal(newTerminalStage(func(t types.T) {
		if !find {
			result = t
			find = true
		}
	}, canFinish(func() bool {
		return find
	})))
	return optional.OfNullable(result)
}

// Count 计算元素个数
func (s *stream) Count() int64 {
	return s.ReduceWith(int64(0), func(count types.R, t types.T) types.R {
		return count.(int64) + 1
	}).(int64)
}


// 测试是否所有元素满足条件
func (s *stream) AllMatch(test types.Predicate) bool {
	result := true
	s.terminal(newTerminalStage(func(t types.T) {
		if !test(t) {
			result = false
		}
	}, canFinish(func() bool { // canFinish返回一个option对象
		return !result
	})))
	return result
}

// 测试是否没有元素满足条件
func (s *stream) NoneMatch(test types.Predicate) bool {
	result := true
	s.terminal(newTerminalStage(func(t types.T) {
		if test(t) {
			result = false
		}
	}, canFinish(func() bool { // canFinish返回一个option对象
		return !result
	})))
	return result
}

// 测试有任意元素满足条件
func (s *stream) AnyMatch(test types.Predicate) bool {
	result := false
	s.terminal(newTerminalStage(func(t types.T) {
		if test(t) {
			result = true
		}
	}, canFinish(func() bool { // canFinish返回一个option对象
		return result
	})))
	return result
}

// end region terminate operate














