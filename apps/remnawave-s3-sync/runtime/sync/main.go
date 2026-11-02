// Command remnawave-s3-sync keeps encrypted copies of Remnawave subscriptions in
// S3-compatible storage, for clients to use when the panel can't be reached. It
// runs once and exits, so a scheduler decides how often.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var version = "dev"

var markerHash = regexp.MustCompile(`^[0-9a-f]{64}$`)

func required(name string, missing *[]string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		*missing = append(*missing, name)
	}
	return value
}

func percent(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 || n > 100 {
		return 0, fmt.Errorf("%s must be 0 to 100", name)
	}
	return n, nil
}

func splitList(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func run(ctx context.Context) error {
	var missing []string
	cfg := config{
		APIURL:     required("RW_API_URL", &missing),
		APIToken:   required("RW_API_TOKEN", &missing),
		SubBaseURL: strings.TrimSpace(os.Getenv("RW_SUB_BASE_URL")),
		SyncToken:  strings.TrimSpace(os.Getenv("RW_SYNC_TOKEN")),
		UserAgent:  strings.TrimSpace(os.Getenv("USER_AGENT")),
		Format: fileFormat{
			Magic:    required("FORMAT_MAGIC", &missing),
			NameInfo: required("FORMAT_NAME_LABEL", &missing),
			KeyInfo:  required("FORMAT_KEY_LABEL", &missing),
		},
		MarkerKey:       strings.TrimSuffix(required("MARKER_KEY", &missing), ":"),
		BackupHeader:    required("BACKUP_HEADER", &missing),
		BackupAddresses: splitList(os.Getenv("BACKUP_ADDRESSES")),
		NewDomain:       strings.TrimSpace(os.Getenv("NEW_DOMAIN")),
		MarkerSHA256:    map[string]bool{},
		Concurrency:     4,
	}
	markers := splitList(required("MARKER_SHA256", &missing))
	endpoint := required("S3_ENDPOINT", &missing)
	bucket := required("S3_BUCKET", &missing)
	accessKey := required("S3_ACCESS_KEY", &missing)
	secretKey := required("S3_SECRET_KEY", &missing)
	if len(missing) > 0 {
		return fmt.Errorf("missing settings: %s", strings.Join(missing, ", "))
	}
	for _, marker := range markers {
		if !markerHash.MatchString(marker) {
			return errors.New("MARKER_SHA256 must be SHA-256 hashes in lower-case hex")
		}
		cfg.MarkerSHA256[marker] = true
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if cfg.SubBaseURL == "" {
		cfg.SubBaseURL = strings.TrimRight(cfg.APIURL, "/") + "/api/sub"
	}
	var err error
	if cfg.MaxFailedPercent, err = percent("MAX_FAILED_PERCENT", 10); err != nil {
		return err
	}
	if cfg.MaxDeletePercent, err = percent("MAX_DELETE_PERCENT", 50); err != nil {
		return err
	}
	if value := strings.TrimSpace(os.Getenv("CONCURRENCY")); value != "" {
		if cfg.Concurrency, err = strconv.Atoi(value); err != nil || cfg.Concurrency < 1 {
			return errors.New("CONCURRENCY must be a positive number")
		}
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: true,
		Region: strings.TrimSpace(os.Getenv("S3_REGION")),
	})
	if err != nil {
		return errors.New("storage settings are invalid")
	}
	s := &syncer{
		cfg:    cfg,
		client: newHTTPClient(),
		store:  &s3Store{client: client, bucket: bucket, prefix: prefixOf(os.Getenv("S3_PREFIX"))},
		now:    time.Now,
		logf:   log.Printf,
	}
	result, err := s.run(ctx)
	log.Printf("users=%d uploaded=%d failed=%d deleted=%d", result.Users, result.Uploaded, result.Failed, result.Deleted)
	return err
}

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	log.SetFlags(log.LstdFlags | log.LUTC)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Print(err)
		cancel()
		os.Exit(1)
	}
}
