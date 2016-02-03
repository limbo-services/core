package tests

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func Test(t *testing.T) {
	a := Person{}
	b := Person{}
	a.Birth = time.Now().AddDate(-30, 0, 0)
	a.BirtdayParties = append(a.BirtdayParties, timePtr(a.Birth.AddDate(1, 0, 0)))
	a.BirtdayParties = append(a.BirtdayParties, timePtr(a.Birth.AddDate(2, 0, 0)))

	dataA, err := a.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	err = b.Unmarshal(dataA)
	if err != nil {
		t.Fatal(err)
	}

	dataB, err := b.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("A: %s", a)
	t.Logf("B: %s", b)

	if !bytes.Equal(dataA, dataB) {
		t.Fatalf("expected A(%v) and B(%v) to be equal", a, b)
	}
}

func Test_JSON(t *testing.T) {
	a := Person{}
	b := Person{}
	a.Birth = time.Now().AddDate(-30, 0, 0)
	a.BirtdayParties = append(a.BirtdayParties, timePtr(a.Birth.AddDate(1, 0, 0)))
	a.BirtdayParties = append(a.BirtdayParties, timePtr(a.Birth.AddDate(2, 0, 0)))

	dataA, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("JSON: %s", dataA)

	err = json.Unmarshal(dataA, &b)
	if err != nil {
		t.Fatal(err)
	}

	dataB, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("A: %s", a)
	t.Logf("B: %s", b)

	if !bytes.Equal(dataA, dataB) {
		t.Fatalf("expected A(%v) and B(%v) to be equal", a, b)
	}
}

func timePtr(t time.Time) time.Time {
	return t
}
