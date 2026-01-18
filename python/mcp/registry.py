from typing import Dict, Any, Type
from google.protobuf.descriptor_pool import DescriptorPool
from google.protobuf.message_factory import GetMessageClass
from google.protobuf.descriptor_pb2 import FileDescriptorSet
from .bsr import BSRClient, BSRRef

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
            fds = self.client.fetch_descriptor_set(ref)
            self.cache[repo_id] = fds
        else:
            fds = self.cache[repo_id]

        # 3. Add to pool
        for fd in fds.file:
            try:
                self.pool.Add(fd)
            except Exception:
                # File might already be added
                pass

        # 4. Return message class
        descriptor = self.pool.FindMessageTypeByName(ref.message)
        return GetMessageClass(descriptor)

    def unpack(self, any_msg: Any) -> Any:
        # any_msg is a google.protobuf.any_pb2.Any
        type_url = any_msg.type_url
        full_name = type_url.split('/')[-1]
        
        descriptor = self.pool.FindMessageTypeByName(full_name)
        msg_class = GetMessageClass(descriptor)
        
        instance = msg_class()
        instance.ParseFromString(any_msg.value)
        return instance
