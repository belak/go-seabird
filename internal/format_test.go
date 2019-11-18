package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrettifySuffix(t *testing.T) {
	require.Equal(t, "1.2K", RawPrettifySuffix(1234, 1000, nil))
	require.Equal(t, "1.2M", RawPrettifySuffix(1234567, 1000, nil))
	require.Equal(t, "1.2B", RawPrettifySuffix(1234567890, 1000, nil))
	require.Equal(t, "999", RawPrettifySuffix(999, 1000, nil))
	require.Equal(t, "1,234.6B", RawPrettifySuffix(1234567890123, 1000, nil))

	require.Equal(t, "1.2K", RawPrettifySuffix(1234, 1000, []string{"K"}))
	require.Equal(t, "1,234.6K", RawPrettifySuffix(1234567, 1000, []string{"K"}))
	require.Equal(t, "1,234,567.9K", RawPrettifySuffix(1234567890, 1000, []string{"K"}))
	require.Equal(t, "999", RawPrettifySuffix(999, 1000, []string{"K"}))
	require.Equal(t, "1,234,567,890.1K", RawPrettifySuffix(1234567890123, 1000, []string{"K"}))

	require.Equal(t, "1,000", RawPrettifySuffix(1000, 2000, []string{"K"}))
}
