import os
import requests
import json
from typing import Optional, List
from google.protobuf import json_format
from google.protobuf.descriptor_pb2 import FileDescriptorSet, FileDescriptorProto

class BSRRef:
    def __init__(self, owner: str, repository: str, message: str, version: str = "main"):
        self.owner = owner
        self.repository = repository
        self.message = message
        self.version = version

    @staticmethod
    def parse(ref: str) -> 'BSRRef':
        if not ref.startswith("buf.build/"):
            raise ValueError("Invalid BSR ref: must start with buf.build/")
        
        parts = ref[len("buf.build/"):].split("/")
        if len(parts) < 3:
            raise ValueError("Invalid BSR ref: too few parts")
        
        owner = parts[0]
        repository = parts[1]
        
        rest = "/".join(parts[2:])
        message_parts = rest.split(":")
        message = message_parts[0]
        version = message_parts[1] if len(message_parts) > 1 else "main"
        
        return BSRRef(owner, repository, message, version)

class BSRClient:
    def __init__(self, token: Optional[str] = None, base_url: str = "https://api.buf.build"):
        self.token = token or os.environ.get("BUF_TOKEN")
        self.base_url = base_url
        self.session = requests.Session()

    def fetch_descriptor_set(self, ref: BSRRef) -> FileDescriptorSet:
        url = f"{self.base_url}/buf.alpha.registry.v1alpha1.ImageService/GetImage"
        
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
            
        payload = {
            "owner": ref.owner,
            "repository": ref.repository,
            "reference": ref.version
        }
        
        resp = self.session.post(url, json=payload, headers=headers)
        if resp.status_code != 200:
            raise Exception(f"BSR API error ({resp.status_code}): {resp.text}")
            
        data = resp.json()
        image = data.get("image", {})
        files = image.get("file", [])
        
        fds = FileDescriptorSet()
        for file_data in files:
            fd = FileDescriptorProto()
            # json_format.ParseDict handles the JSON representation of FileDescriptorProto
            json_format.ParseDict(file_data, fd)
            fds.file.append(fd)
            
        return fds
