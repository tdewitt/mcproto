import unittest
from unittest.mock import MagicMock, patch
from mcp.bsr import BSRRef, BSRClient
from google.protobuf import json_format
from google.protobuf.descriptor_pb2 import FileDescriptorProto

class TestBSR(unittest.TestCase):
    def test_parse_ref(self):
        ref_str = "buf.build/acme/tools/acme.tools.v1.WebSearchRequest:v1"
        ref = BSRRef.parse(ref_str)
        self.assertEqual(ref.owner, "acme")
        self.assertEqual(ref.repository, "tools")
        self.assertEqual(ref.message, "acme.tools.v1.WebSearchRequest")
        self.assertEqual(ref.version, "v1")

    @patch('requests.Session.post')
    def test_fetch_descriptor_set(self, mock_post):
        # Mock a FileDescriptorProto
        fd = FileDescriptorProto()
        fd.name = "test.proto"
        fd.package = "test.v1"
        fd_data = json_format.MessageToDict(fd)

        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "image": {
                "file": [fd_data]
            }
        }
        mock_post.return_value = mock_response

        client = BSRClient(token="test")
        ref = BSRRef("test", "test", "test")
        fds = client.fetch_descriptor_set(ref)

        self.assertEqual(len(fds.file), 1)
        self.assertEqual(fds.file[0].name, "test.proto")

if __name__ == '__main__':
    unittest.main()
