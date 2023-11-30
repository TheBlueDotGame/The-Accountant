#!./venv/bin/python
from concurrent import futures
import grpc
import addons_pb2
import addons_pb2_grpc

err_msg_not_authorized = "not authorized"

def serve(port, workers, token):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=workers))
    addons_pb2_grpc.add_AddonsAPIServicer_to_server(AddonsServerServicer(token), server)
    server.add_insecure_port("[::]:"+str(port))
    server.start()
    server.wait_for_termination()


class AddonsServerServicer(addons_pb2_grpc.AddonsAPIServicer):
    """Provides methods that implement functionality of add-ons server."""

    def __init__(self, token):
        self.token = token


    def AnalyzeTransaction(self, request, context):
        if request.token != self.token:
            return addons_pb2.AddonsError(err_msg_not_authorized)
        # do staff with request.data
        return addons_pb2.AddonsError("")


if __name__ == "__main__":
    port = 50051
    workers = 10
    token = "5152557cd65516c983726acee119e4f6cd20e26fa473d08cd26c1685e492edd3"
    print("starting server on port ", port, " with workers pool ", workers, ".")
    serve(port, workers, token)


