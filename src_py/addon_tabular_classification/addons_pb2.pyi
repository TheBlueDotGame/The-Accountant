from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class AddonsMessage(_message.Message):
    __slots__ = ["token", "data"]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    token: str
    data: bytes
    def __init__(self, token: _Optional[str] = ..., data: _Optional[bytes] = ...) -> None: ...

class AddonsError(_message.Message):
    __slots__ = ["error"]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    error: str
    def __init__(self, error: _Optional[str] = ...) -> None: ...
