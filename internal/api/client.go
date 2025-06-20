package api

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"

	pb "EnigmaNetz/Enigma-Go-Sensor/internal/api/publish"
)

// grpcClient defines the interface for gRPC operations
type grpcClient interface {
	uploadExcelMethod(ctx context.Context, data []byte, employeeId string) (string, int32, string, error)
}

// LogUploader handles uploading logs to the gRPC server
type LogUploader struct {
	client       grpcClient
	apiKey       string
	retryCount   int
	retryDelay   time.Duration
	compressFunc func([]byte) ([]byte, error) // for DI/testing
}

// LogFiles contains paths to the log files to upload
type LogFiles struct {
	DNSPath  string
	ConnPath string
}

// CombinedLogs represents the compressed log data
type CombinedLogs struct {
	DNS  string `json:"dns"`  // base64 encoded compressed data
	Conn string `json:"conn"` // base64 encoded compressed data
}

// ErrAPIGone is returned when the API responds with HTTP 410 (Gone), indicating the sensor should stop.
var ErrAPIGone = errors.New("API returned 410 Gone: sensor should stop sending data and terminate")

// grpcClientImpl implements the grpcClient interface
type grpcClientImpl struct {
	client pb.PublishServiceClient
}

// NewLogUploader creates a new log uploader instance
func NewLogUploader(serverAddr string, apiKey string) (*LogUploader, error) {
	var opts []grpc.DialOption

	// Always use SSL credentials with system trust store
	host := serverAddr
	if idx := strings.LastIndex(serverAddr, ":"); idx >= 0 {
		host = serverAddr[:idx]
	}
	creds := credentials.NewClientTLSFromCert(nil, host)
	opts = append(opts, grpc.WithTransportCredentials(creds))

	// Add keepalive options
	opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`))

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	return &LogUploader{
		client:       &grpcClientImpl{client: pb.NewPublishServiceClient(conn)},
		apiKey:       apiKey,
		retryCount:   3,
		retryDelay:   5 * time.Second,
		compressFunc: compressData,
	}, nil
}

func (c *grpcClientImpl) uploadExcelMethod(ctx context.Context, data []byte, employeeId string) (string, int32, string, error) {
	req := &pb.UploadExcelRequest{
		Data:       data,
		EmployeeId: employeeId,
	}

	// Ensure the message implements proto.Message
	if _, ok := interface{}(req).(proto.Message); !ok {
		return "", 0, "", fmt.Errorf("request does not implement proto.Message")
	}

	resp, err := c.client.UploadExcelMethod(ctx, req)
	if err != nil {
		return "", 0, "", fmt.Errorf("gRPC call failed: %v", err)
	}

	return resp.Status, resp.StatusCode, resp.Message, nil
}

// UploadLogs uploads the DNS and connection logs to the server
func (u *LogUploader) UploadLogs(ctx context.Context, files LogFiles) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Read and compress log files
	combinedData, err := u.prepareLogData(files)
	if err != nil {
		return fmt.Errorf("failed to prepare log data: %v", err)
	}

	// Upload with retries
	var lastErr error
	for i := 0; i < u.retryCount; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := u.upload(ctx, combinedData); err != nil {
			lastErr = err
			time.Sleep(u.retryDelay)
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to upload after %d retries: %w", u.retryCount, lastErr)
}

// prepareLogData reads, compresses, and combines the log files
func (u *LogUploader) prepareLogData(files LogFiles) ([]byte, error) {
	// Read DNS log (allow missing)
	dnsData, err := os.ReadFile(files.DNSPath)
	if err != nil {
		if os.IsNotExist(err) {
			dnsData = []byte{} // treat missing DNS log as empty
		} else {
			return nil, fmt.Errorf("failed to read DNS log: %v", err)
		}
	}

	// Read connection log (required)
	connData, err := os.ReadFile(files.ConnPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read connection log: %v", err)
	}

	// Compress DNS data
	dnsCompressed, err := u.compressFunc(dnsData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress DNS data: %v", err)
	}

	// Compress connection data
	connCompressed, err := u.compressFunc(connData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress connection data: %v", err)
	}

	// Combine into JSON structure
	combined := CombinedLogs{
		DNS:  base64.StdEncoding.EncodeToString(dnsCompressed),
		Conn: base64.StdEncoding.EncodeToString(connCompressed),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(combined)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal combined data: %v", err)
	}

	// Compress the combined JSON
	return u.compressFunc(jsonData)
}

// upload sends the compressed data to the server
func (u *LogUploader) upload(ctx context.Context, data []byte) error {
	_, statusCode, message, err := u.client.uploadExcelMethod(ctx, data, u.apiKey)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %v", err)
	}

	if statusCode == 410 {
		return fmt.Errorf("API returned 410 Gone: sensor should stop sending data and terminate: %w", ErrAPIGone)
	}

	if statusCode != 200 {
		return fmt.Errorf("upload failed: %s (code: %d)", message, statusCode)
	}

	return nil
}

// compressData compresses byte data using zlib
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write compressed data: %v", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close compressor: %v", err)
	}

	return buf.Bytes(), nil
}
