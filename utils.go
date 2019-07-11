package seabird

func minIntSlice(v []int) (m int) {
	for i, e := range v {
		if i == 0 || e < m {
			m = e
		}
	}
	return
}
