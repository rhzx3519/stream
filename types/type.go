package types

type (
	// T is a empty interface, that is `any` type.
	// since Go is not support generics now(but will coming soon),
	// so we use T to represent any type
	T interface {}
	// R is another `any` type used to distinguish T
	R interface {}
	// U is another `any` type used to distinguish T
	U interface {}
	// Function represents a conversion ability, which accepts one argument and produces a result
	Function func(t T) R
	// IntFunction is a Function, which result type is int
	IntFunction func(t T) int
	// Predicate is a Function, which produces a bool value. usually used to test a value whether satisfied condition
	Predicate func(t T) bool
	// UnaryOperator is a Function, which argument and result are the same type
	UnaryOperator func(t T) T
	// Consumer accepts one argument and not produces any result
	Consumer func(t T)
	// Supplier returns a result. each time invoked it can returns a new or distinct result
	Supplier func() R
	// BiFunction like Function, but is accepts two arguments and produces a result
	BiFunction func(t T, u U) R
	// BinaryOperator is a BiFunction which input and result are the same type
	BinaryOperator func(t1, t2 T) T
	// Comparator is a BiFunction, which two input arguments are the type, and returns a int.
	// if t1 is greater then t2, it returns a positive number;
	// if t1 is less then t2, it returns a negative number; if the two input are equal, it returns 0
	Comparator func(left, right T) int
	// pair is a struct with two elements
	Pair struct {
		First T
		Second R
	}
)


var (
	// IntComparator is a Comparator for int
	IntComparator Comparator = func(left, right T) int {
		if left.(int) > right.(int) {
			return 1
		} else if left.(int) < right.(int) {
			return -1
		}
		return 0
	}

	// IntComparator is a Comparator for int64
	Int64Comparator Comparator = func(left, right T) int {
		if left.(int64) > right.(int64) {
			return 1
		} else if left.(int64) < right.(int64) {
			return -1
		}
		return 0
	}
)

// return a reversed comparator
func ReverseOrder(cmp Comparator) Comparator {
	return func(left, right T) int {
		return cmp(right, left)
	}
}