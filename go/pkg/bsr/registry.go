package bsr

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// cacheEntry wraps a cached FileDescriptorSet with an access timestamp for LRU eviction.
type cacheEntry struct {
	fds        *descriptorpb.FileDescriptorSet
	lastAccess time.Time
}

// Registry manages dynamic Protobuf message types.
type Registry struct {
	client *Client
	mu     sync.RWMutex
	files  *protoregistry.Files
	cache  map[string]*cacheEntry
}

func NewRegistry(client *Client) *Registry {
	return &Registry{
		client: client,
		files:  new(protoregistry.Files),
		cache:  make(map[string]*cacheEntry),
	}
}

const MaxCacheSize = 100

// Resolve fetches and registers the schema for a BSR reference.
func (r *Registry) Resolve(ctx context.Context, refStr string) (protoreflect.MessageType, error) {
	ref, err := ParseRef(refStr)
	if err != nil {
		return nil, err
	}

	// 1. Check if we already have the message type (read lock only).
	r.mu.RLock()
	if mt, findErr := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(ref.Message)); findErr == nil {
		r.mu.RUnlock()
		return mt, nil
	}

	// 2. Check cache under read lock.
	repoID := fmt.Sprintf("%s/%s@%s", ref.Owner, ref.Repository, ref.Version)
	entry, cached := r.cache[repoID]
	r.mu.RUnlock()

	var fds *descriptorpb.FileDescriptorSet
	if cached {
		fds = entry.fds
		// Update access time under write lock.
		r.mu.Lock()
		if e, ok := r.cache[repoID]; ok {
			e.lastAccess = time.Now()
		}
		r.mu.Unlock()
	} else {
		// 3. Fetch from BSR without holding the mutex.
		fds, err = r.client.FetchDescriptorSet(ctx, ref)
		if err != nil {
			return nil, err
		}

		// Store in cache under write lock.
		r.mu.Lock()
		// Evict oldest entries if cache is full.
		if len(r.cache) >= MaxCacheSize {
			r.evictOldest()
		}
		r.cache[repoID] = &cacheEntry{
			fds:        fds,
			lastAccess: time.Now(),
		}
		r.mu.Unlock()
	}

	// 4. Register the files under write lock.
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, fdProto := range fds.File {
		fd, fdErr := protodesc.NewFile(fdProto, r.files)
		if fdErr != nil {
			// File might already be registered
			continue
		}
		if regErr := r.files.RegisterFile(fd); regErr != nil {
			return nil, fmt.Errorf("failed to register file %s: %w", fd.Path(), regErr)
		}
	}

	// 5. Find the message descriptor and return a dynamic message type.
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

// evictOldest removes the least-recently-accessed cache entry.
// Must be called with r.mu held for writing.
func (r *Registry) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	for key, entry := range r.cache {
		if first || entry.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccess
			first = false
		}
	}
	if oldestKey != "" {
		delete(r.cache, oldestKey)
	}
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
