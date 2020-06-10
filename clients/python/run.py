import os

import grpc

from echo_pb2_grpc import EchoServiceStub
from echo_pb2 import EchoRequest


def run():
    insecure = os.getenv('INSECURE', 'false').lower() == 'true'
    credentials = grpc.composite_channel_credentials(
        grpc.local_channel_credentials() if insecure else grpc.ssl_channel_credentials(),
        grpc.access_token_call_credentials(os.environ['TOKEN']),
    )
    channel = grpc.secure_channel('localhost:8000', credentials)
    client = EchoServiceStub(channel)

    response = client.Echo(EchoRequest(message='hello world'))
    print("got echo response: " + response.message)


if __name__ == '__main__':
    run()
