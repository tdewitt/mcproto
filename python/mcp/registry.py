from typing import Dict, Any, Type
from google.protobuf.descriptor_pool import DescriptorPool
from google.protobuf.message_factory import GetMessageClass
from google.protobuf.descriptor_pb2 import FileDescriptorSet
from .bsr import BSRClient, BSRRef
from .config import MAX_REGISTRY_CACHE_SIZE

class Registry:
    def __init__(self, client: BSRClient):
        self.client = client
        self.pool = DescriptorPool()
        self.cache: Dict[str, FileDescriptorSet] = {}

    def resolve(self, ref_str: str) -> Type:
        ref = BSRRef.parse(ref_str)
        
        # 1. Check if descriptor is already in pool
        try:
            descriptor = self.pool.FindMessageTypeByName(ref.message)
            return GetMessageClass(descriptor)
        except KeyError:
            pass

        # 2. Fetch from BSR if not in cache
        repo_id = f"{ref.owner}/{ref.repository}@{ref.version}"
        if repo_id not in self.cache:
            # Security: Bounded cache to prevent memory exhaustion
            if len(self.cache) >= MAX_REGISTRY_CACHE_SIZE:
                self.cache.clear() # Basic eviction

            fds = self.client.fetch_descriptor_set(ref)
            self.cache[repo_id] = fds
        else:
            fds = self.cache[repo_id]

        # 3. Add to pool (including dependencies)
        # BSR returns a full Image which usually has all non-standard dependencies.
        # But for well-known types, we might need to be careful.
        for fd in fds.file:
            try:
                self.pool.AddSerializedFile(fd.SerializeToString())
            except (ValueError, TypeError) as e:
                # Skip files that fail to add (may already exist in pool)
                pass

        # 4. Return message class
        try:
            descriptor = self.pool.FindMessageTypeByName(ref.message)
            return GetMessageClass(descriptor)
        except KeyError:
            # If still not found, it might be because the name in BSR image 
            # doesn't match the ref.message exactly (e.g. leading dot).
            for fd in fds.file:
                for mt in fd.message_type:
                    full_name = f"{fd.package}.{mt.name}" if fd.package else mt.name
                    if full_name == ref.message:
                        descriptor = self.pool.FindMessageTypeByName(full_name)
                        return GetMessageClass(descriptor)
            raise

    def unpack(self, any_msg: Any) -> Any:
        # any_msg is a google.protobuf.any_pb2.Any
        type_url = any_msg.type_url
        full_name = type_url.split('/')[-1]
        
        descriptor = self.pool.FindMessageTypeByName(full_name)
        msg_class = GetMessageClass(descriptor)
        
        instance = msg_class()
        instance.ParseFromString(any_msg.value)
        return instance
