import socket

from common.utils import Bet

AMOUNT_BATCHES_SIZE_BYTES = 4
BATCH_SIZE_BYTES = 4
SEPARATOR = ","
SEPARATOR_BETS = ";"
TYPE_MESSAGE_SIZE_BYTES = 1
CLIENT_ID_SIZE_BYTES = 1
AMOUNT_WINNERS_SIZE_BYTES = 4
DOCUMENT_SIZE_BYTES = 4

BATCH_MESSAGE = 0
NOTIFY_MESSAGE = 1
ID_MESSAGE = 2

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

    def _send_all(self, bytes_to_send: bytes):
        total_bytes_sent = 0

        while total_bytes_sent < len(bytes_to_send):
            bytes_sent = self._socket.send(bytes_to_send[total_bytes_sent:], socket.MSG_WAITALL)
            if not bytes_sent:
                raise OSError("failed to sent data")
            total_bytes_sent += bytes_sent

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

    def _recv_client_id(self):
        client_id = int.from_bytes(self._recv_all(CLIENT_ID_SIZE_BYTES), "big")
        return client_id

    def recv(self):
        type_message = int.from_bytes(self._recv_all(TYPE_MESSAGE_SIZE_BYTES), "big")

        if type_message == BATCH_MESSAGE:
            return type_message, self._recv_batches()
        elif type_message == NOTIFY_MESSAGE:
            return type_message, None
        elif type_message == ID_MESSAGE:
            return type_message, self._recv_client_id()

        raise ValueError(f"invalid type of message {type_message}")

    def send_winners(self, winners):
        amount_winners_bytes = len(winners).to_bytes(AMOUNT_WINNERS_SIZE_BYTES, "big")
        self._send_all(amount_winners_bytes)

        for document in winners:
            document_bytes = document.to_bytes(DOCUMENT_SIZE_BYTES, "big")
            self._send_all(document_bytes)

    def close(self):
        self._socket.close()