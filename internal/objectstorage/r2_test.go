package objectstorage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestNewR2ClientNormalizesConfig(t *testing.T) {
	client, err := NewR2Client(context.Background(), R2Config{
		AccountID:       " account ",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Bucket:          " bucket ",
		PublicURL:       " https://cdn.example.com/ ",
	})
	if err != nil {
		t.Fatalf("NewR2Client returned error: %v", err)
	}
	if client.bucket != "bucket" || client.publicURL != "https://cdn.example.com" {
		t.Fatalf("client = %#v", client)
	}
}

func TestR2ClientDeleteIgnoresEmptyKey(t *testing.T) {
	client := &R2Client{}
	if err := client.Delete(context.Background(), "  "); err != nil {
		t.Fatalf("Delete empty key returned error: %v", err)
	}
}

func TestR2ClientPutAndDeleteUseS3(t *testing.T) {
	httpClient := &fakeS3HTTPClient{statusCode: http.StatusOK}
	client := testR2Client(httpClient)

	url, err := client.Put(context.Background(), "/avatars/a.png", strings.NewReader("image"), 5, "image/png")
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if url != "https://cdn.example.com/avatars/a.png" {
		t.Fatalf("url = %q", url)
	}
	if httpClient.calls == 0 || httpClient.lastRequest.Method != http.MethodPut {
		t.Fatalf("put request = %#v", httpClient.lastRequest)
	}

	if err := client.Delete(context.Background(), "avatars/a.png"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if httpClient.lastRequest.Method != http.MethodDelete {
		t.Fatalf("delete method = %s", httpClient.lastRequest.Method)
	}
}

func TestR2ClientS3Errors(t *testing.T) {
	client := testR2Client(&fakeS3HTTPClient{err: errors.New("network")})
	if _, err := client.Put(context.Background(), "key", strings.NewReader("x"), 1, "text/plain"); err == nil {
		t.Fatal("Put returned nil error")
	}
	if err := client.Delete(context.Background(), "key"); err == nil {
		t.Fatal("Delete returned nil error")
	}
}

func testR2Client(httpClient *fakeS3HTTPClient) *R2Client {
	return &R2Client{
		client: s3.New(s3.Options{
			Region:       "auto",
			Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("key", "secret", "")),
			BaseEndpoint: aws.String("https://r2.example.com"),
			UsePathStyle: true,
			HTTPClient:   httpClient,
		}),
		bucket:    "bucket",
		publicURL: "https://cdn.example.com",
	}
}

type fakeS3HTTPClient struct {
	statusCode   int
	err          error
	calls        int
	lastRequest  *http.Request
	responseBody string
}

func (c *fakeS3HTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.calls++
	c.lastRequest = req
	if c.err != nil {
		return nil, c.err
	}
	statusCode := c.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(c.responseBody)),
	}, nil
}
