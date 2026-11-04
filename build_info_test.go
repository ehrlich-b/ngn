package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
)

func TestBuildInfoMetadataAndAbsence(t *testing.T) {
	info := makeBuildInfo(nil, nil, "hce")
	if info.Revision != nil || info.Dirty != nil || info.EmbeddedModelSHA256 != nil || info.DefaultEvaluator != "hce" || info.GOOS == "" || info.GOARCH == "" {
		t.Fatalf("missing metadata: %+v", info)
	}
	info = makeBuildInfo(&debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}, {Key: "vcs.modified", Value: "true"}}}, []byte("model"), "ngnn3")
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte("model")))
	if info.Revision == nil || *info.Revision != "abc" || info.Dirty == nil || !*info.Dirty || info.EmbeddedModelSHA256 == nil || *info.EmbeddedModelSHA256 != digest {
		t.Fatalf("metadata: %+v", info)
	}
}

func TestBuildInfoCommand(t *testing.T) {
	for _, profile := range []string{"nnue", "hce"} {
		useProfile(t, profile)
		var out, stderr bytes.Buffer
		if code := run([]string{"--build-info"}, strings.NewReader("uci\nquit\n"), &out, &stderr); code != 0 || stderr.Len() != 0 {
			t.Fatalf("exit %d stderr %s", code, &stderr)
		}
		var info buildInfo
		if err := json.Unmarshal(out.Bytes(), &info); err != nil {
			t.Fatal(err)
		}
		want := "ngnn3"
		if profile == "hce" {
			want = "hce"
		}
		if info.DefaultEvaluator != want || info.EmbeddedModelSHA256 == nil {
			t.Fatalf("info %+v", info)
		}
	}
}
