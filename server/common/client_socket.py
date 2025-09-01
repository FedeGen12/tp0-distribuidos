import socket

from common.utils import Bet

BET_SIZE = 4
SEPARATOR = ","

class ClientSocket:
    def __init__(self, client_socket: socket.socket):
        self._socket = client_socket

    def _recv_all(self, bytes_to_recv: int):
        bytes_received = bytearray()

        while len(bytes_received) < bytes_to_recv:
            bytes_read = self._socket.recv(bytes_to_recv - len(bytes_received), socket.MSG_WAITALL)
            if not bytes_read:
                raise OSError("failed to receive data")
            bytes_received.extend(bytes_read)

        return bytes(bytes_received)

    def recv(self):
        bet_size = int.from_bytes(self._recv_all(BET_SIZE))
        bet_bytes = self._recv_all(bet_size)
        bet = Bet(*bet_bytes.decode().split(SEPARATOR))
        return bet

    def close(self):
        self._socket.close()