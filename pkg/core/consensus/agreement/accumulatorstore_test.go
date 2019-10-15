package agreement

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func mockAgreement(id string) Agreement {
	return Agreement{
		SignedVotes: []byte(id),
	}
}

func TestStore(t *testing.T) {
	sec := newStore()
	ev1 := mockAgreement("one")
	ev2 := mockAgreement("two")
	ev3 := mockAgreement("one")

	require.Equal(t, 1, sec.Insert(ev1, "1"))
	require.Equal(t, 1, sec.Insert(ev1, "1"))
	require.Equal(t, 1, sec.Insert(ev1, "2"))
	require.Equal(t, 2, sec.Insert(ev2, "2"))
	require.Equal(t, 2, sec.Insert(ev3, "2"))
}

func TestClear(t *testing.T) {
	sec := newStore()
	ev1 := mockAgreement("one")
	ev2 := mockAgreement("two")
	ev3 := mockAgreement("three")

	stepOne := "1"
	sec.Insert(ev1, stepOne)
	sec.Insert(ev2, stepOne)
	stepOneSize := sec.Insert(ev3, stepOne)
	require.Equal(t, 3, stepOneSize)

	stepTwo := "2"
	sec.Insert(ev1, stepTwo)
	sec.Insert(ev2, stepTwo)
	stepTwoSize := sec.Insert(ev3, stepTwo)
	require.Equal(t, 3, stepTwoSize)

	sec.Clear()
	require.Equal(t, 0, len(sec.Get(stepOne)))
	require.Equal(t, 0, len(sec.Get(stepTwo)))
}

func TestContains(t *testing.T) {
	sec := newStore()
	ev1 := mockAgreement("one")
	ev2 := mockAgreement("two")
	ev3 := mockAgreement("three")
	ev4 := mockAgreement("one")

	stepOne := "1"
	sec.Insert(ev1, stepOne)
	sec.Insert(ev2, stepOne)

	require.True(t, sec.Contains(ev1, stepOne))
	require.True(t, sec.Contains(ev2, stepOne))
	require.False(t, sec.Contains(ev3, stepOne))
	require.True(t, sec.Contains(ev4, stepOne))
}

func TestSECOperations(t *testing.T) {
	sec := newStore()
	ev1 := mockAgreement("one")
	ev2 := mockAgreement("two")
	ev3 := mockAgreement("one")

	// checking if the length of the array of step is consistent
	require.Equal(t, 1, sec.Insert(ev1, "1"))
	require.Equal(t, 1, sec.Insert(ev1, "1"))
	require.Equal(t, 1, sec.Insert(ev1, "2"))
	require.Equal(t, 2, sec.Insert(ev2, "2"))
	require.Equal(t, 2, sec.Insert(ev3, "2"))

	sec.Clear()
	require.Equal(t, 1, sec.Insert(ev1, "1"))
	require.Equal(t, 1, sec.Insert(ev1, "1"))
	require.Equal(t, 1, sec.Insert(ev1, "2"))
	require.Equal(t, 2, sec.Insert(ev2, "2"))
	require.Equal(t, 2, sec.Insert(ev3, "2"))
}
