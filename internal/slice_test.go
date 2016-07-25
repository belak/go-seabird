package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepend(t *testing.T) {
	var data = []struct {
		base     []interface{}
		add      int
		expected []interface{}
	}{
		{
			nil,
			1,
			[]interface{}{1},
		},
		{
			[]interface{}{1},
			2,
			[]interface{}{2, 1},
		},
	}

	for _, testData := range data {
		out := Prepend(testData.base, testData.add)

		assert.EqualValues(t, testData.expected, out)
	}

}
