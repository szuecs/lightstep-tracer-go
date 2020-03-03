package metrics

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/lightstep/lightstep-tracer-common/golang/gogo/metricspb"
)

const (
	// DefaultReporterAddress             = "https://metricingest.lightstep.com"
	DefaultReporterAddress             = "http://localhost:9876"
	DefaultReporterTimeout             = time.Second * 5
	DefaultReporterMeasurementDuration = time.Second * 30
)

var (
	acceptHeader      = http.CanonicalHeaderKey("Accept")
	contentTypeHeader = http.CanonicalHeaderKey("Content-Type")
	accessTokenHeader = http.CanonicalHeaderKey("Lightstep-Access-Token")
)

const (
	reporterPath = "/metrics"

	idempotencyKeyByteLength = 30
	protoContentType         = "application/octet-stream"
)

type Reporter struct {
	client      *http.Client
	tracerID    uint64
	attributes  map[string]string
	address     string
	timeout     time.Duration
	accessToken string
}

func NewReporter(opts ...ReporterOption) *Reporter {
	c := newConfig(opts...)

	return &Reporter{
		client:     &http.Client{},
		tracerID:   c.tracerID,
		attributes: c.attributes,
		address:    fmt.Sprintf("%s%s", c.address, reporterPath),
		timeout:    c.timeout,
		// duration:    c.duration,
		accessToken: c.accessToken,
	}
}

func (r *Reporter) prepareRequest(m Metrics) (*metricspb.IngestRequest, error) {
	start := time.Now()
	idempotencyKey, err := generateIdempotencyKey()
	if err != nil {
		return nil, err
	}
	return &metricspb.IngestRequest{
		IdempotencyKey: idempotencyKey,
		ReporterId:     r.tracerID,
		Labels:         r.attributes,
		Start: &types.Timestamp{
			Seconds: start.Unix(),
			Nanos:   int32(start.Nanosecond()),
		},
		// TODO: how do we want to handle duration?
		Duration: &types.Duration{
			Seconds: 0,
			Nanos:   0,
		},
	}, nil
}

func addFloat(labels map[string]string, key string, value float64) *metricspb.MetricPoint {
	return &metricspb.MetricPoint{
		Kind:          metricspb.MetricKind_GAUGE,
		TimeSeriesKey: key,
		Labels:        labels,
		Value: &metricspb.MetricPoint_Float{
			Float: value,
		},
	}
}

func addUint(labels map[string]string, key string, value uint64) *metricspb.MetricPoint {
	return &metricspb.MetricPoint{
		Kind:          metricspb.MetricKind_GAUGE,
		TimeSeriesKey: key,
		Labels:        labels,
		Value: &metricspb.MetricPoint_Uint{
			Uint: value,
		},
	}
}

// Measure takes a snapshot of system metrics and sends them
// to a LightStep endpoint.
func (r *Reporter) Measure(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	m, err := Measure(ctx)
	if err != nil {
		return err
	}

	pb, err := r.prepareRequest(m)
	if err != nil {
		return err
	}

	labels := map[string]string{
		"name": "process.cpu",
	}
	pb.Points = append(pb.Points, addFloat(labels, "cpu.user", m.ProcessCPU.User))
	pb.Points = append(pb.Points, addFloat(labels, "cpu.system", m.ProcessCPU.System))

	labels = map[string]string{
		"name": "mem",
	}
	pb.Points = append(pb.Points, addUint(labels, "mem.available", m.Memory.Available))
	pb.Points = append(pb.Points, addUint(labels, "mem.used", m.Memory.Used))

	for label, cpu := range m.CPU {
		labels := map[string]string{
			"name": label,
		}

		pb.Points = append(pb.Points, addFloat(labels, "cpu.system", cpu.System))
		pb.Points = append(pb.Points, addFloat(labels, "cpu.user", cpu.User))
		pb.Points = append(pb.Points, addFloat(labels, "cpu.idle", cpu.Idle))
		pb.Points = append(pb.Points, addFloat(labels, "cpu.steal", cpu.Steal))
		pb.Points = append(pb.Points, addFloat(labels, "cpu.nice", cpu.Nice))
	}
	for label, nic := range m.NIC {
		labels := map[string]string{
			"name": label,
		}
		pb.Points = append(pb.Points, addUint(labels, "net.recv", nic.BytesReceived))
		pb.Points = append(pb.Points, addUint(labels, "net.sent", nic.BytesSent))
	}

	b, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, r.address, bytes.NewReader(b))
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	req.Header.Set(contentTypeHeader, protoContentType)
	req.Header.Set(acceptHeader, protoContentType)
	req.Header.Set(accessTokenHeader, r.accessToken)

	res, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

type ReporterOption func(*config)

func WithReporterTracerID(tracerID uint64) ReporterOption {
	return func(c *config) {
		c.tracerID = tracerID
	}
}

func WithReporterAttributes(attributes map[string]string) ReporterOption {
	return func(c *config) {
		for k, v := range attributes {
			c.attributes[k] = v
		}
	}
}

// WithReporterAddress sets the address of the LightStep endpoint
func WithReporterAddress(address string) ReporterOption {
	return func(c *config) {
		c.address = address
	}
}

func WithReporterTimeout(timeout time.Duration) ReporterOption {
	return func(c *config) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

func WithReporterMeasurementDuration(measurementDuration time.Duration) ReporterOption {
	return func(c *config) {
		if measurementDuration > 0 {
			c.measurementDuration = measurementDuration
		}
	}
}

// WithReporterAccessToken sets an access token for communicating with LightStep
func WithReporterAccessToken(accessToken string) ReporterOption {
	return func(c *config) {
		c.accessToken = accessToken
	}
}

type config struct {
	tracerID            uint64
	attributes          map[string]string
	address             string
	timeout             time.Duration
	measurementDuration time.Duration
	accessToken         string
}

func newConfig(opts ...ReporterOption) config {
	var c config

	defaultOpts := []ReporterOption{
		WithReporterAttributes(make(map[string]string)),
		WithReporterAddress(DefaultReporterAddress),
		WithReporterTimeout(DefaultReporterTimeout),
		WithReporterMeasurementDuration(DefaultReporterMeasurementDuration),
	}

	for _, opt := range append(defaultOpts, opts...) {
		opt(&c)
	}

	return c
}

func generateIdempotencyKey() (string, error) {
	b := make([]byte, idempotencyKeyByteLength)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	return strings.ToLower(base32.StdEncoding.EncodeToString(b)), nil
}
