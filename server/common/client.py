from common.utils import Bet
from common.server_utils import recv_all_by_socket, send_all_by_socket

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

class Client:
    def __init__(self, client_socket: "ClientSocket", client_id: int):
        self.socket = client_socket
        self.id = client_id

class ClientSocket:
    def __init__(self, client_socket):
        self._socket = client_socket

    def _recv_batch(self):
        bets = []
        batch_size = int.from_bytes(recv_all_by_socket(self._socket, BATCH_SIZE_BYTES), "big")
        batch_bytes = recv_all_by_socket(self._socket, batch_size)
        for bet_bytes in batch_bytes.split(SEPARATOR_BETS.encode()):
            bet = Bet(*bet_bytes.decode().split(SEPARATOR))
            bets.append(bet)
        return bets

    def _recv_client_id(self):
        client_id = int.from_bytes(recv_all_by_socket(self._socket, CLIENT_ID_SIZE_BYTES), "big")
        return client_id

    def recv(self):
        type_message = int.from_bytes(recv_all_by_socket(self._socket, TYPE_MESSAGE_SIZE_BYTES), "big")

        if type_message == BATCH_MESSAGE:
            return type_message, self._recv_batch()
        elif type_message == NOTIFY_MESSAGE:
            return type_message, None
        elif type_message == ID_MESSAGE:
            return type_message, self._recv_client_id()

        raise ValueError(f"invalid type of message {type_message}")

    def send_winners(self, winners):
        amount_winners_bytes = len(winners).to_bytes(AMOUNT_WINNERS_SIZE_BYTES, "big")
        send_all_by_socket(self._socket, amount_winners_bytes)

        for document in winners:
            document_bytes = document.to_bytes(DOCUMENT_SIZE_BYTES, "big")
            send_all_by_socket(self._socket, document_bytes)

    def close(self):
        self._socket.close()