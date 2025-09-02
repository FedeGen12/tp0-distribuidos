import socket
from common.client_socket import ClientSocket

class ServerSocket:
    def __init__(self, skt: socket.socket):
        self._socket = skt

    @classmethod
    def setup_listener(cls, host: str, port: int, backlog: int = 0):
        self = cls(socket.socket(socket.AF_INET, socket.SOCK_STREAM))
        self._socket.bind((host, port))
        self._socket.listen(backlog)
        return self

    def accept(self):
        skt, addr = self._socket.accept()
        client_socket = ClientSocket(skt)
        _, client_id_bytes = client_socket.recv()

        client_id = int.from_bytes(client_id_bytes, "big")
        return client_socket, client_id, addr

    def close(self):
        self._socket.close()