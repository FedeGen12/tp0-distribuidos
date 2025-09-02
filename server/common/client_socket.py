import socket

from common.utils import Bet

AMOUNT_BATCHES_SIZE_BYTES = 4
BATCH_SIZE_BYTES = 4
SEPARATOR = ","
SEPARATOR_BETS = ";"
TYPE_MESSAGE_SIZE_BYTES = 1

BATCH_MESSAGE = 0
NOTIFY_MESSAGE = 1

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

    def _recv_batches(self):
        amount_batches = int.from_bytes(self._recv_all(AMOUNT_BATCHES_SIZE_BYTES), "big")
        bets = []

        for _ in range(amount_batches):
            batch_size = int.from_bytes(self._recv_all(BATCH_SIZE_BYTES), "big")
            batch_bytes = self._recv_all(batch_size)
            for bet_bytes in batch_bytes.split(SEPARATOR_BETS.encode()):
                bet = Bet(*bet_bytes.decode().split(SEPARATOR))
                bets.append(bet)
        return bets

    def recv(self):
        type_message = int.from_bytes(self._recv_all(TYPE_MESSAGE_SIZE_BYTES), "big")

        if type_message == BATCH_MESSAGE:
            return self._recv_batches()
        elif type_message == NOTIFY_MESSAGE:
            return None

        raise ValueError(f"invalid type of message {type_message}")

    def close(self):
        self._socket.close()