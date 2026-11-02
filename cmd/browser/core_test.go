package main

import (
	"encoding/json"
	"testing"
)

func check(t *testing.T, req string) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(execute([]byte(req)), &got); err != nil {
		t.Fatal(err)
	}
	return got
}
func TestGameRules(t *testing.T) {
	if got := check(t, `{"move":"e2e5"}`); got["error"] != "illegal move" {
		t.Fatal(got)
	}
	if got := check(t, `{"fen":"7k/P7/8/8/8/8/8/7K w - - 0 1","move":"a7a8n"}`); got["error"] != nil {
		t.Fatal(got)
	}
	if got := check(t, `{"fen":"7k/5Q2/6K1/8/8/8/8/8 b - - 0 1"}`); got["status"] != "stalemate" {
		t.Fatal(got)
	}
	if got := check(t, `{"fen":"7k/6Q1/6K1/8/8/8/8/8 b - - 0 1"}`); got["status"] != "checkmate" {
		t.Fatal(got)
	}
	if got := check(t, `{"fen":"7k/8/8/8/8/8/8/K7 w - - 100 1"}`); got["status"] != "50-move rule" {
		t.Fatal(got)
	}
	if got := check(t, `{"moves":["g1f3","g8f6","f3g1","f6g8","g1f3","g8f6","f3g1","f6g8"]}`); got["status"] != "repetition" {
		t.Fatal(got)
	}
	if got := check(t, `{}`); got["network"] != "812152293dad04184599b2f9ab82779168e6e134b221de136d8f8daae86d9f22" {
		t.Fatal(got)
	}
}
