package main

import (
	"context"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Optional: signed upload policies for on-demand last-resort enrolment. Each device
// the panel lists gets a policy for its own request object, inside its user's
// encrypted file. A policy holds no storage key: it permits one exact object, up to
// a size limit, until it expires. Signing is local and contacts no one.

const (
	// maxRequestBytes matches the control service's own limit.
	maxRequestBytes = 8 << 10
	signAlgorithm   = "AWS4-HMAC-SHA256"
)

// enrolConfig holds the separate bucket and key for enrolment.
type enrolConfig struct {
	Endpoint, Bucket, Region string
	AccessKey, SecretKey     string
	Lifetime                 time.Duration
	// The client must use the same values, or its requests won't match any policy.
	Prefix, UserLabel, RequestLabel string
}

// uploadPolicy is what the device needs to upload its request.
type uploadPolicy struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

func labelled(parts ...string) string {
	var b []byte
	for _, p := range parts {
		b = binary.BigEndian.AppendUint16(b, uint16(len(p)))
		b = append(b, p...)
	}
	return string(b)
}

// requestName is the device's request object, computed as the client does. The name
// reveals neither the token nor the device.
func (e *enrolConfig) requestName(token, hwid string) (string, error) {
	user, err := hkdf.Key(sha256.New, []byte(token), nil, labelled(e.UserLabel), 16)
	if err != nil {
		return "", err
	}
	device, err := hkdf.Key(sha256.New, []byte(token), nil, labelled(e.RequestLabel, hwid), 16)
	if err != nil {
		return "", err
	}
	return e.Prefix + hex.EncodeToString(user) + "/" + hex.EncodeToString(device), nil
}

func signStep(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}

func (e *enrolConfig) policy(name string, now time.Time) (uploadPolicy, error) {
	now = now.UTC()
	date := now.Format("20060102")
	credential := e.AccessKey + "/" + date + "/" + e.Region + "/s3/aws4_request"
	stamp := now.Format("20060102T150405Z")
	doc, err := json.Marshal(map[string]any{
		"expiration": now.Add(e.Lifetime).Format("2006-01-02T15:04:05.000Z"),
		"conditions": []any{
			map[string]string{"bucket": e.Bucket},
			[]any{"eq", "$key", name},
			[]any{"content-length-range", 0, maxRequestBytes},
			map[string]string{"x-amz-algorithm": signAlgorithm},
			map[string]string{"x-amz-credential": credential},
			map[string]string{"x-amz-date": stamp},
		},
	})
	if err != nil {
		return uploadPolicy{}, err
	}
	encoded := base64.StdEncoding.EncodeToString(doc)
	key := signStep([]byte("AWS4"+e.SecretKey), date)
	for _, step := range []string{e.Region, "s3", "aws4_request"} {
		key = signStep(key, step)
	}
	return uploadPolicy{
		URL: "https://" + e.Bucket + "." + e.Endpoint + "/",
		Fields: map[string]string{
			"key": name, "policy": encoded, "x-amz-algorithm": signAlgorithm,
			"x-amz-credential": credential, "x-amz-date": stamp,
			"x-amz-signature": hex.EncodeToString(signStep(key, encoded)),
		},
	}, nil
}

// policiesFor signs one policy per device, keyed by object name so the file holds
// no raw HWIDs.
func (e *enrolConfig) policiesFor(token string, hwids []string, now time.Time) (map[string]uploadPolicy, error) {
	out := make(map[string]uploadPolicy, len(hwids))
	for _, hwid := range hwids {
		name, err := e.requestName(token, hwid)
		if err != nil {
			return nil, err
		}
		if out[name], err = e.policy(name, now); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// deviceKey is how a device names its user. The running panel links by numeric id,
// though its 2.7.4 schema documents userUuid; both are read, so an upgrade cannot
// silently leave every user with no policies.
func deviceKey(id int64, uuid string) string {
	if id != 0 {
		return "id:" + strconv.FormatInt(id, 10)
	}
	return "uuid:" + uuid
}

// devicesByUser reads every device from the panel, by device key.
func (s *syncer) devicesByUser(ctx context.Context) (map[string][]string, error) {
	out := map[string][]string{}
	for start := 0; ; {
		address := fmt.Sprintf("%s/api/hwid/devices?start=%d&size=%d", strings.TrimRight(s.cfg.APIURL, "/"), start, usersPageSize)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		if err != nil {
			return nil, errors.New("panel API address is invalid")
		}
		request.Header.Set("Authorization", "Bearer "+s.cfg.APIToken)
		setForwarded(request)
		response, err := s.client.Do(request)
		if err != nil {
			return nil, errors.New("panel device request failed")
		}
		var page struct {
			Response struct {
				Devices []struct {
					HWID     string `json:"hwid"`
					UserID   int64  `json:"userId"`
					UserUUID string `json:"userUuid"`
				} `json:"devices"`
				Total int `json:"total"`
			} `json:"response"`
		}
		status := response.StatusCode
		err = json.NewDecoder(io.LimitReader(response.Body, 64*1024*1024)).Decode(&page)
		response.Body.Close()
		if status != http.StatusOK {
			return nil, fmt.Errorf("panel device list answered %d", status)
		}
		if err != nil {
			return nil, errors.New("panel device list could not be read")
		}
		for _, d := range page.Response.Devices {
			key := deviceKey(d.UserID, d.UserUUID)
			out[key] = append(out[key], d.HWID)
		}
		start += len(page.Response.Devices)
		if len(page.Response.Devices) == 0 || start >= page.Response.Total {
			return out, nil
		}
	}
}
