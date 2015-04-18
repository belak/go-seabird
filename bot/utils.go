package bot

func prepend(v []interface{}, e interface{}) []interface{} {
	var vc []interface{}

	vc = append(vc, e)
	vc = append(vc, v...)

	return vc
}
