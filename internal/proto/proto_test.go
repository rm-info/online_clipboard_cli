package proto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL), srv
}

func TestFetchChallenge_Success(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pow/challenge" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"challenge": "abc", "difficulty": 18})
	})
	ch, err := client.FetchChallenge(context.Background())
	if err != nil {
		t.Fatalf("FetchChallenge: %v", err)
	}
	if ch.Challenge != "abc" || ch.Difficulty != 18 {
		t.Errorf("unexpected response: %+v", ch)
	}
}

func TestCreateSession_RoundtripsCookie(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("salt"); got != "fakesalt" {
			t.Errorf("salt form field: got %q", got)
		}
		http.SetCookie(w, &http.Cookie{
			Name:   "clip_token_abc12",
			Value:  "TOKEN",
			MaxAge: 7200,
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":       true,
			"sid":      "abc12",
			"redirect": "/abc12",
		})
	})

	resp, err := client.CreateSession(context.Background(), &CreateSessionRequest{
		FirstItemCT:   "n:c",
		FirstItemSize: 5,
		Salt:          "fakesalt",
		VerifierBlob:  "n:c",
		AuthAnchor:    strings.Repeat("a", 64),
		POWChallenge:  "x",
		POWNonce:      "0",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if resp.SID != "abc12" {
		t.Errorf("SID: got %q want abc12", resp.SID)
	}
	if resp.Cookie != "TOKEN" {
		t.Errorf("Cookie: got %q want TOKEN", resp.Cookie)
	}
	if resp.ExpiresAt == 0 {
		t.Errorf("ExpiresAt should be non-zero (Set-Cookie max-age was 7200)")
	}
}

func TestGetVerifier_NotFoundReturnsSentinel(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Session not found."})
	})
	_, err := client.GetVerifier(context.Background(), "abc12")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestAuthenticate_WrongPasswordReturnsSentinel(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "wrong_password", "failures": 3})
	})
	_, err := client.Authenticate(context.Background(), "abc12", "proof", "ch", "0")
	if !errors.Is(err, ErrWrongPassword) {
		t.Errorf("expected ErrWrongPassword, got %v", err)
	}
}

func TestAuthenticate_SuccessReturnsCookie(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "clip_token_abc12", Value: "TOK", MaxAge: 3600})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "redirect": "/abc12"})
	})
	resp, err := client.Authenticate(context.Background(), "abc12", "proof", "ch", "0")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if resp.Cookie != "TOK" {
		t.Errorf("Cookie: got %q want TOK", resp.Cookie)
	}
}

func TestGetContents_ParsesMixedEntries(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the cookie was forwarded.
		cookie, err := r.Cookie("clip_token_abc12")
		if err != nil || cookie.Value != "TOK" {
			t.Errorf("cookie: %v / %v", cookie, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entries": []map[string]any{
				{"id": "i1", "type": "text", "ciphertext": "n:c", "size": 5, "created_at": 1700000000},
				{"id": "f1", "type": "file", "encrypted_name": "n:c", "size": 1024, "uploaded_at": 1700000100, "has_thumb": true},
			},
			"file_bytes": 1024,
		})
	})

	contents, err := client.GetContents(context.Background(), "abc12", "TOK")
	if err != nil {
		t.Fatalf("GetContents: %v", err)
	}
	if len(contents.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(contents.Entries))
	}
	if contents.Entries[0].Type != EntryText {
		t.Errorf("first entry type: got %q want text", contents.Entries[0].Type)
	}
	if contents.Entries[1].Type != EntryFile || !contents.Entries[1].HasThumb {
		t.Errorf("second entry: got %+v", contents.Entries[1])
	}
}

func TestAddItem_PostsForm(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("ciphertext") != "n:c" {
			t.Errorf("ciphertext: %q", r.Form.Get("ciphertext"))
		}
		if r.Form.Get("plain_size") != "42" {
			t.Errorf("plain_size: %q", r.Form.Get("plain_size"))
		}
		if r.Form.Get("secret") != "true" {
			t.Errorf("secret: %q", r.Form.Get("secret"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	if err := client.AddItem(context.Background(), "abc12", "TOK", "n:c", 42, true); err != nil {
		t.Errorf("AddItem: %v", err)
	}
}

func TestUploadFile_SendsMultipart(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Fatalf("Content-Type: %q (%v)", r.Header.Get("Content-Type"), err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		seen := map[string][]byte{}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("multipart: %v", err)
			}
			data, _ := io.ReadAll(part)
			seen[part.FormName()] = data
		}
		if string(seen["encrypted_name"]) != "n:c-name" {
			t.Errorf("encrypted_name: %q", seen["encrypted_name"])
		}
		if string(seen["plain_size"]) != "100" {
			t.Errorf("plain_size: %q", seen["plain_size"])
		}
		if string(seen["file"]) != "fakebytes" {
			t.Errorf("file: %q", seen["file"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"file": map[string]any{
				"id":             "fid",
				"encrypted_name": "n:c-name",
				"size":           100,
				"uploaded_at":    1700000000,
				"has_thumb":      false,
			},
		})
	})

	out, err := client.UploadFile(context.Background(), "abc12", "TOK", "n:c-name", 100, []byte("fakebytes"), nil)
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if out.ID != "fid" {
		t.Errorf("ID: got %q want fid", out.ID)
	}
}

func TestDownloadFile_DecodesEncryptedNameHeader(t *testing.T) {
	const rawName = "nonce:ciphertext"
	encodedName := base64.URLEncoding.EncodeToString([]byte(rawName))
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Clip-Encrypted-Name", encodedName)
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("BINARYDATA"))
	})
	name, body, err := client.DownloadFile(context.Background(), "abc12", "TOK", "fid")
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	if name != rawName {
		t.Errorf("name: got %q want %q", name, rawName)
	}
	if string(body) != "BINARYDATA" {
		t.Errorf("body: got %q want BINARYDATA", body)
	}
}

func TestWipe_AcceptsRedirect(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Mimic the real server: 303 See Other to /.
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	if err := client.Wipe(context.Background(), "abc12", "TOK"); err != nil {
		t.Errorf("Wipe: %v", err)
	}
}

func TestParseHTTPError_ExtractsDetail(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "something went wrong"})
	})
	_, err := client.FetchChallenge(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.Status != 400 || httpErr.Detail != "something went wrong" {
		t.Errorf("got %+v", httpErr)
	}
}
