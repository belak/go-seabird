package internal

// Prepend is similar to the builtin "append" operation, but it puts the element
// at the start of the list.
func Prepend(v []interface{}, e interface{}) []interface{} {
	var vc []interface{}

	vc = append(vc, e)
	vc = append(vc, v...)

	return vc
}
