package main

import (
	"bytes"
	"testing"
)

func TestInjectFrontendLockPasswordSkipsEmptyPassword(t *testing.T) {
	originalIndexPage := indexPage
	defer func() {
		indexPage = originalIndexPage
	}()

	t.Setenv("FRONTEND_LOCK_PASSWORD", "")
	indexPage = []byte("<!doctype html><html><head></head><body></body></html>")

	InjectFrontendLockPassword()

	if bytes.Contains(indexPage, []byte("__FRONTEND_LOCK_PASSWORD__")) {
		t.Fatalf("expected empty password to skip frontend lock injection, got %s", string(indexPage))
	}
}

func TestInjectFrontendLockPasswordInjectsConfiguredPassword(t *testing.T) {
	originalIndexPage := indexPage
	defer func() {
		indexPage = originalIndexPage
	}()

	t.Setenv("FRONTEND_LOCK_PASSWORD", "open-sesame")
	indexPage = []byte("<!doctype html><html><head><title>New API</title></head><body></body></html>")

	InjectFrontendLockPassword()

	expected := []byte(`<script>window.__FRONTEND_LOCK_PASSWORD__="open-sesame";</script>`)
	if !bytes.Contains(indexPage, expected) {
		t.Fatalf("expected frontend lock password injection, got %s", string(indexPage))
	}
	if bytes.Index(indexPage, expected) > bytes.Index(indexPage, []byte("</head>")) {
		t.Fatalf("expected frontend lock script to be injected before </head>, got %s", string(indexPage))
	}
}

func TestInjectFrontendLockPasswordEscapesScriptBreakingCharacters(t *testing.T) {
	originalIndexPage := indexPage
	defer func() {
		indexPage = originalIndexPage
	}()

	t.Setenv("FRONTEND_LOCK_PASSWORD", `x"</script><script>alert(1)</script>`)
	indexPage = []byte("<!doctype html><html><head></head><body></body></html>")

	InjectFrontendLockPassword()

	if bytes.Contains(indexPage, []byte(`"</script><script>`)) {
		t.Fatalf("expected injected password to escape script-breaking content, got %s", string(indexPage))
	}
	if !bytes.Contains(indexPage, []byte(`\u003c/script\u003e\u003cscript\u003e`)) {
		t.Fatalf("expected HTML-sensitive characters to be JSON escaped, got %s", string(indexPage))
	}
}
