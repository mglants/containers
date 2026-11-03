package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxConfigBytes = 2 * 1024 * 1024
	usersPageSize  = 500
	syncHeader     = "x-sync-token"
	// USER_AGENT replaces this when panel rules expect a particular client.
	defaultUserAgent = "mihomo"
	accept           = "application/yaml, application/x-yaml, text/yaml, text/plain"
)

// keptHeaders are the headers clients display. Device-limit answers are left out:
// a copied refusal would block someone reading the file later.
var keptHeaders = []string{
	"profile-title", "profile-update-interval", "subscription-userinfo", "support-url",
	"profile-web-page-url", "announce", "premium-url", "expiry-banner-text", "expiry-banner-button-url",
}

var fileName = regexp.MustCompile(`^[0-9a-f]{32}$`)

// store is the storage this job writes to; every name is relative to the prefix.
type store interface {
	Put(ctx context.Context, name string, data []byte) error
	List(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, name string) error
}

type config struct {
	APIURL           string
	APIToken         string
	SubBaseURL       string
	SyncToken        string
	UserAgent        string
	Format           fileFormat
	MarkerKey        string
	MarkerSHA256     map[string]bool
	BackupHeader     string
	BackupAddresses  []string
	NewDomain        string
	MaxFailedPercent int
	MaxDeletePercent int
	Concurrency      int
}

type report struct {
	Users, Uploaded, Failed, Deleted int
}

// syncer logs and returns counts and fixed failure categories only: errors from
// HTTP clients can contain URLs, and a URL here contains a subscription token.
type syncer struct {
	cfg    config
	client *http.Client
	store  store
	now    func() time.Time
	logf   func(string, ...any)
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

type panelUser struct {
	ShortUUID string `json:"shortUuid"`
	Status    string `json:"status"`
}

// setForwarded lets requests reach the panel without its reverse proxy: it closes
// connections that don't carry these headers. A proxy in between replaces them.
func setForwarded(request *http.Request) {
	request.Header.Set("X-Forwarded-For", "127.0.0.1")
	request.Header.Set("X-Forwarded-Proto", "https")
}

func (s *syncer) activeTokens(ctx context.Context) ([]string, error) {
	var tokens []string
	for start := 0; ; start += usersPageSize {
		address := fmt.Sprintf("%s/api/users?start=%d&size=%d", strings.TrimRight(s.cfg.APIURL, "/"), start, usersPageSize)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		if err != nil {
			return nil, errors.New("panel API address is invalid")
		}
		request.Header.Set("Authorization", "Bearer "+s.cfg.APIToken)
		setForwarded(request)
		response, err := s.client.Do(request)
		if err != nil {
			return nil, errors.New("panel API request failed")
		}
		var page struct {
			Response struct {
				Users []panelUser `json:"users"`
				Total int         `json:"total"`
			} `json:"response"`
		}
		status := response.StatusCode
		err = json.NewDecoder(io.LimitReader(response.Body, 64*1024*1024)).Decode(&page)
		response.Body.Close()
		if status != http.StatusOK {
			return nil, fmt.Errorf("panel API answered %d", status)
		}
		if err != nil {
			return nil, errors.New("panel API answer could not be read")
		}
		for _, user := range page.Response.Users {
			if user.Status == "ACTIVE" {
				tokens = append(tokens, user.ShortUUID)
			}
		}
		if len(page.Response.Users) == 0 || start+len(page.Response.Users) >= page.Response.Total {
			return tokens, nil
		}
	}
}

func topLevelValues(body []byte, key string) []string {
	var out []string
	for rest := body; len(rest) > 0; {
		var line []byte
		line, rest, _ = bytes.Cut(rest, []byte("\n"))
		value, found := bytes.CutPrefix(bytes.TrimRight(line, "\r"), []byte(key))
		if found {
			out = append(out, string(bytes.Trim(bytes.TrimSpace(value), `"'`)))
		}
	}
	return out
}

func (s *syncer) hasMarker(body []byte) bool {
	for _, value := range topLevelValues(body, s.cfg.MarkerKey+":") {
		sum := sha256.Sum256([]byte(value))
		if s.cfg.MarkerSHA256[hex.EncodeToString(sum[:])] {
			return true
		}
	}
	return false
}

// fetchSubscription fails rather than store something a client would refuse.
func (s *syncer) fetchSubscription(ctx context.Context, token string) (backupContent, error) {
	var content backupContent
	address := strings.TrimRight(s.cfg.SubBaseURL, "/") + "/" + url.PathEscape(token)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return content, errors.New("invalid address")
	}
	request.Header.Set("User-Agent", s.cfg.UserAgent)
	request.Header.Set("Accept", accept)
	setForwarded(request)
	if s.cfg.SyncToken != "" {
		request.Header.Set(syncHeader, s.cfg.SyncToken)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return content, errors.New("request failed")
	}
	defer response.Body.Close()
	if strings.EqualFold(response.Header.Get("x-hwid-not-supported"), "true") ||
		strings.EqualFold(response.Header.Get("x-hwid-max-devices-reached"), "true") {
		return content, errors.New("device limit answered")
	}
	if response.StatusCode != http.StatusOK {
		return content, errors.New("status " + strconv.Itoa(response.StatusCode))
	}
	if media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type")); err == nil &&
		(media == "text/html" || media == "application/json") {
		return content, errors.New("not a config")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxConfigBytes+1))
	if err != nil || len(body) > maxConfigBytes || len(body) == 0 {
		return content, errors.New("body unusable")
	}
	if !s.hasMarker(body) {
		return content, errors.New("marker missing")
	}
	content = backupContent{CreatedAt: s.now().UnixMilli(), Config: string(body), Headers: map[string]string{}}
	for _, name := range keptHeaders {
		if values := response.Header.Values(name); len(values) == 1 && len(values[0]) <= 8192 {
			content.Headers[name] = values[0]
		}
	}
	// Moving is the operator's decision for this run, not a copy of the panel's.
	if s.cfg.NewDomain != "" {
		content.Headers["new-domain"] = s.cfg.NewDomain
	}
	addresses := s.cfg.BackupAddresses
	if len(addresses) == 0 {
		addresses = response.Header.Values(s.cfg.BackupHeader)
	}
	if len(addresses) > 0 {
		content.Headers[s.cfg.BackupHeader] = strings.Join(addresses, ",")
	}
	return content, nil
}

func (s *syncer) syncOne(ctx context.Context, token string) error {
	if len(token) < minToken {
		return errors.New("token too short")
	}
	content, err := s.fetchSubscription(ctx, token)
	if err != nil {
		return err
	}
	data, err := s.cfg.Format.seal(token, content)
	if err != nil {
		return errors.New("encryption failed")
	}
	name, err := s.cfg.Format.fileName(token)
	if err != nil {
		return errors.New("name failed")
	}
	if err := s.store.Put(ctx, name, data); err != nil {
		return errors.New("upload failed")
	}
	return nil
}

// run uploads as it goes, because an upload can't harm an existing copy. A user
// whose fetch fails keeps their previous file, which may still work.
func (s *syncer) run(ctx context.Context) (report, error) {
	var result report
	tokens, err := s.activeTokens(ctx)
	if err != nil {
		return result, err
	}
	result.Users = len(tokens)
	// A panel returning nothing must not look like every user being removed.
	if len(tokens) == 0 {
		return result, errors.New("panel listed no active users; nothing changed")
	}
	expected := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		if name, err := s.cfg.Format.fileName(token); err == nil {
			expected[name] = true
		}
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		reasons = map[string]int{}
		slots   = make(chan struct{}, max(1, s.cfg.Concurrency))
	)
	for _, token := range tokens {
		slots <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			err := s.syncOne(ctx, token)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Failed++
				reasons[err.Error()]++
				return
			}
			result.Uploaded++
		}()
	}
	wg.Wait()
	for reason, count := range reasons {
		s.logf("failed: %s: %d", reason, count)
	}
	if result.Failed*100 > result.Users*s.cfg.MaxFailedPercent {
		return result, fmt.Errorf("%d of %d subscriptions failed; nothing deleted", result.Failed, result.Users)
	}

	listed, err := s.store.List(ctx)
	if err != nil {
		return result, errors.New("storage listing failed; nothing deleted")
	}
	var stale []string
	files := 0
	for _, name := range listed {
		// Only this job's own files, whatever else shares the prefix.
		if !fileName.MatchString(name) {
			continue
		}
		files++
		if !expected[name] {
			stale = append(stale, name)
		}
	}
	if len(stale)*100 > files*s.cfg.MaxDeletePercent {
		return result, fmt.Errorf("would delete %d of %d files; nothing deleted", len(stale), files)
	}
	for _, name := range stale {
		if err := s.store.Delete(ctx, name); err != nil {
			return result, errors.New("deleting a file failed")
		}
		result.Deleted++
	}
	return result, nil
}
