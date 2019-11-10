package internal

import (
	"fmt"
)

type ContextKey string

func (key ContextKey) String() string {
	return fmt.Sprintf("ContextKey(%s)", string(key))
}
