package main

import (
	"bytes"
	"testing"
)

func TestInjectFrontendLockPasswordSkipsEmptyPassword(t *testing.T) {
	originalIndexPage := indexPage
	originalClassicIndexPage := classicIndexPage
	defer func() {
		indexPage = originalIndexPage
		classicIndexPage = originalClassicIndexPage
	}()

	t.Setenv("FRONTEND_LOCK_PASSWORD", "")
	indexPage = []byte("<!doctype html><html><head></head><body></body></html>")
	classicIndexPage = []byte("<!doctype html><html><head></head><body></body></html>")

	InjectFrontendLockPassword()

	if bytes.Contains(indexPage, []byte("__FRONTEND_LOCK_PASSWORD__")) {
		t.Fatalf("expected empty password to skip frontend lock injection, got %s", string(indexPage))
	}
	if bytes.Contains(classicIndexPage, []byte("__FRONTEND_LOCK_PASSWORD__")) {
		t.Fatalf("expected empty password to skip classic frontend lock injection, got %s", string(classicIndexPage))
	}
}

func TestInjectFrontendLockPasswordInjectsConfiguredPassword(t *testing.T) {
	originalIndexPage := indexPage
	originalClassicIndexPage := classicIndexPage
	defer func() {
		indexPage = originalIndexPage
		classicIndexPage = originalClassicIndexPage
	}()

	t.Setenv("FRONTEND_LOCK_PASSWORD", "open-sesame")
	indexPage = []byte("<!doctype html><html><head><title>New API</title></head><body></body></html>")
	classicIndexPage = []byte("<!doctype html><html><head><title>New API Classic</title></head><body></body></html>")

	InjectFrontendLockPassword()

	expected := []byte(`<script>window.__FRONTEND_LOCK_PASSWORD__="open-sesame";</script>`)
	for name, page := range map[string][]byte{
		"default": indexPage,
		"classic": classicIndexPage,
	} {
		if !bytes.Contains(page, expected) {
			t.Fatalf("expected %s frontend lock password injection, got %s", name, string(page))
		}
		if bytes.Index(page, expected) > bytes.Index(page, []byte("</head>")) {
			t.Fatalf("expected %s frontend lock script before </head>, got %s", name, string(page))
		}
	}
}

func TestInjectFrontendLockPasswordEscapesScriptBreakingCharacters(t *testing.T) {
	originalIndexPage := indexPage
	originalClassicIndexPage := classicIndexPage
	defer func() {
		indexPage = originalIndexPage
		classicIndexPage = originalClassicIndexPage
	}()

	t.Setenv("FRONTEND_LOCK_PASSWORD", `x"</script><script>alert(1)</script>`)
	indexPage = []byte("<!doctype html><html><head></head><body></body></html>")
	classicIndexPage = []byte("<!doctype html><html><head></head><body></body></html>")

	InjectFrontendLockPassword()

	for name, page := range map[string][]byte{
		"default": indexPage,
		"classic": classicIndexPage,
	} {
		if bytes.Contains(page, []byte(`"</script><script>`)) {
			t.Fatalf("expected %s injected password to escape script-breaking content, got %s", name, string(page))
		}
		if !bytes.Contains(page, []byte(`\u003c/script\u003e\u003cscript\u003e`)) {
			t.Fatalf("expected %s HTML-sensitive characters to be JSON escaped, got %s", name, string(page))
		}
	}
}
