generate-protobuf:
	python -m grpc_tools.protoc --python_out=. --grpc_python_out=. --pyi_out=. -I ../../protobuf/ ../../protobuf/addons.proto
freeze:
	python -m pip freeze > requirements.txt
