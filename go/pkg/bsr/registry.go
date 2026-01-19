package bsr

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// Registry manages dynamic Protobuf message types.
type Registry struct {
	client *Client
	mu     sync.RWMutex
	files  *protoregistry.Files
	cache  map[string]*descriptorpb.FileDescriptorSet
}

func NewRegistry(client *Client) *Registry {
	return &Registry{
		client: client,
		files:  new(protoregistry.Files),
		cache:  make(map[string]*descriptorpb.FileDescriptorSet),
	}
}

const MaxCacheSize = 100

// Resolve fetches and registers the schema for a BSR reference.
func (r *Registry) Resolve(ctx context.Context, refStr string) (protoreflect.MessageType, error) {
	ref, err := ParseRef(refStr)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. Check if we already have the message type
	if mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(ref.Message)); err == nil {
		return mt, nil
	}

	// 2. Fetch from BSR if not in cache
	repoID := fmt.Sprintf("%s/%s@%s", ref.Owner, ref.Repository, ref.Version)
	fds, ok := r.cache[repoID]
	if !ok {
		// Security: Bounded cache to prevent memory exhaustion
		if len(r.cache) >= MaxCacheSize {
			// Basic eviction: clear cache if full
			// In production, use a proper LRU
			r.cache = make(map[string]*descriptorpb.FileDescriptorSet)
		}

		fds, err = r.client.FetchDescriptorSet(ctx, ref)
		if err != nil {
			return nil, err
		}
		r.cache[repoID] = fds
	}

	// 3. Register the files in the local registry
	for _, fdProto := range fds.File {
		fd, err := protodesc.NewFile(fdProto, r.files)
		if err != nil {
			// File might already be registered
			continue
		}
		if err := r.files.RegisterFile(fd); err != nil {
			return nil, fmt.Errorf("failed to register file %s: %w", fd.Path(), err)
		}
	}

	// 4. Find the message descriptor and return a dynamic message type
	md, err := r.files.FindDescriptorByName(protoreflect.FullName(ref.Message))
	if err != nil {
		return nil, fmt.Errorf("message %s not found in descriptors: %w", ref.Message, err)
	}

	messageDesc, ok := md.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a message", ref.Message)
	}

	return dynamicpb.NewMessageType(messageDesc), nil
}

// Unpack dynamically unpacks a google.protobuf.Any message.
func (r *Registry) Unpack(any *anypb.Any) (protoreflect.Message, error) {
	// Extract the full name from the TypeUrl
	// Format: type.googleapis.com/full.name
	parts := strings.Split(any.TypeUrl, "/")
	fullName := protoreflect.FullName(parts[len(parts)-1])

	md, err := r.files.FindDescriptorByName(fullName)
	if err != nil {
		return nil, fmt.Errorf("type %s not found in registry: %w", fullName, err)
	}

	messageDesc, ok := md.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a message", fullName)
	}

	msg := dynamicpb.NewMessage(messageDesc)
	if err := proto.Unmarshal(any.Value, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dynamic message: %w", err)
	}

	return msg, nil
}
