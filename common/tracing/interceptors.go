// Spec: docs/specs/004-opentelemetry-tracing.md

package tracing

import (
	"google.golang.org/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// NewServerInterceptors returns gRPC interceptors with tracing enabled
// Spec: docs/specs/004-opentelemetry-tracing.md#2-grpc-interceptors
func NewServerInterceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return otelgrpc.UnaryServerInterceptor(), otelgrpc.StreamServerInterceptor()
}

// NewClientInterceptors returns gRPC client interceptors with tracing enabled
// Spec: docs/specs/004-opentelemetry-tracing.md#2-grpc-interceptors
func NewClientInterceptors() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return otelgrpc.UnaryClientInterceptor(), otelgrpc.StreamClientInterceptor()
}