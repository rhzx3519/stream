package stream

import (
	"github.com/rhzx3519/stream/types"
	"reflect"
)

const unkonwnSize  = -1

type iterator interface {
	GetSizeIfKnown() int64
	HasNext() bool
	Next() types.T
}

// 创建切片迭代器
func it(elements ...types.T) iterator {
	return &sliceIterator{
		base: &base{
			current: 0,
			size: len(elements),
		},
		elements: elements,
	}
}

// 创建种子迭代器
func withSeed(seed types.T, f types.UnaryOperator) iterator {
	return &seedIt{
		element: seed,
		operator: f,
		first: true,
	}
}

// 创建迭代迭代器
func withSupplier(get types.Supplier) iterator {
	return &supplierIt{
		get: get,
	}
}

// 创建范围迭代器
func withRange(fromInclude, toExclude endpoint, step int) iterator {
	return &rangeIt{
		from: fromInclude,
		to: toExclude,
		step: step,
		next: fromInclude,
	}
}

// implementation of iterator
type base struct {
	current, size int
}

func (b *base) GetSizeIfKnown() int64 {
	return int64(b.size)
}

func (b *base) HasNext() bool {
	return b.current < b.size
}

// region sliceIterator

type sliceIterator struct {
	*base
	elements []types.T
}

func (s *sliceIterator) Next() types.T {
	e := s.elements[s.current]
	s.current++
	return e
}

// end region sliceIterator

// region intsIt

type intsIt struct {
	*base
	elements []int
}

func (i *intsIt) Next() types.T {
	e := i.elements[i.current]
	i.current++
	return e
}

// end region intsIt

// region int64sIt

type int64sIt struct {
	*base
	elements []int64
}

func (i *int64sIt) Next() types.T {
	e := i.elements[i.current]
	i.current++
	return e
}

// end region int64sIt

// region float32sIt

type float32sIt struct {
	*base
	elements []float32
}

func (f *float32sIt) Next() types.T {
	e := f.elements[f.current]
	f.current++
	return e
}

// end region float32sIt

// region float64sIt

type float64sIt struct {
	*base
	elements []float64
}

func (f *float64sIt) Next() types.T {
	e := f.elements[f.current]
	f.current++
	return e
}

// end region float64sIt

// region stringIt

type stringIt struct {
	*base
	elements []string
}

func (it *stringIt) Next() types.T {
	e := it.elements[it.current]
	it.current++
	return e
}

// end region stringIt

// region sliceIt
// sliceIt 切片迭代器 反射实现
type sliceIt struct {
	*base
	sliceValue reflect.Value
}

func (it *sliceIt) Next() types.T {
	e := it.sliceValue.Index(it.current).Interface()
	it.current++
	return e
}

// end region sliceIt

// region mapIt
// mapIt hash迭代器 反射实现
type mapIt struct {
	*base
	mapValue *reflect.MapIter
}

func (it *mapIt) Next() types.T {
	it.base.current++
	it.mapValue.Next()
	return types.Pair{
		First: it.mapValue.Key().Interface(),
		Second: it.mapValue.Value().Interface(),
	}
}

// end region mapIt

// region seedIt
// 种子迭代器, 通过传入的UnaryOperator生成下一个元素
type seedIt struct {
	element 	types.T				// 初始种子
	operator 	types.UnaryOperator	// 迭代函数
	first 		bool
}

func (s *seedIt) GetSizeIfKnown() int64 {
	return unkonwnSize
}

func (s *seedIt) HasNext() bool {
	return true
}

func (s *seedIt) Next() types.T {
	if s.first {
		s.first = false
		return s.element
	}
	s.element = s.operator(s.element)
	return s.element
}

// end region seedIt

// region supplierIt
// 每次返回相同的元素
type supplierIt struct {
	get types.Supplier // 元素生成器
}

func (s *supplierIt) GetSizeIfKnown() int64 {
	return unkonwnSize
}

func (s *supplierIt) HasNext() bool {
	return true
}

func (s *supplierIt) Next() types.T {
	return s.get()
}

// end region supplierIt

// region rangeIt
// 范围迭代器
type rangeIt struct {
	from endpoint
	to 	 endpoint
	step int
	next endpoint
}

func (r *rangeIt) GetSizeIfKnown() int64 {
	return unkonwnSize
}

func (r *rangeIt) HasNext() bool {
	if r.step >= 0 {
		return r.next.CompareTo(r.to) < 0
	}
	return r.next.CompareTo(r.to) > 0
}

func (r *rangeIt) Next() types.T {
	curr := r.next
	r.next = curr.Add(r.step)
	return curr
}

// end region rangeIt

// region Sortable
// Sortable use types.Comparator to sort []types.T 可以使用指定的 cmp 比较器对 list 进行排序
// see sort.Interface
type Sortable struct {
	List []types.T
	Cmp types.Comparator
}
// Len is the number of elements in the collection.
func (a *Sortable) Len() int {
	return len(a.List)
}
// Less reports whether the element with
// index i should sort before the element with index j.
func (a *Sortable) Less(i, j int) bool {
	return a.Cmp(a.List[i], a.List[j]) < 0
}

// Swap swaps the elements with indexes i and j.
func (a *Sortable) Swap(i, j int) {
	a.List[i], a.List[j] = a.List[j], a.List[i]
}

//end region Sortable

// region endpoint
// endpoint used in rangeIt.
type endpoint interface {
	CompareTo(other endpoint) int
	Add(step int) endpoint
}

type epInt int

func (m epInt) CompareTo(other endpoint) int {
	return int(m - other.(epInt))
}

func (m epInt) Add(step int) endpoint {
	return m + epInt(step)
}

type epInt64 int

func (m epInt64) CompareTo(other endpoint) int {
	return int(m - other.(epInt64))
}

func (m epInt64) Add(step int) endpoint {
	return m + epInt64(step)
}
// end region endpoint


