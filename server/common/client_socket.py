import socket

from common.utils import Bet

AMOUNT_BATCHES_SIZE_BYTES = 4
BATCH_SIZE_BYTES = 4
SEPARATOR = ","
SEPARATOR_BETS = ";"

class ClientSocket:
    def __init__(self, client_socket: socket.socket):
        self._socket = client_socket

    def _recv_all(self, bytes_to_recv: int):
        bytes_received = bytearray()

        while len(bytes_received) < bytes_to_recv:
            bytes_read = self._socket.recv(bytes_to_recv - len(bytes_received), socket.MSG_WAITALL)

            if not bytes_read:
                if len(bytes_received) == 0:
                    # todavía no recibí nada, socket cerrado
                    return None
                raise OSError("failed to receive data")
            bytes_received.extend(bytes_read)

        return bytes(bytes_received)

    def recv(self):
        bets = []

        batch_size_bytes = self._recv_all(BATCH_SIZE_BYTES)
        if batch_size_bytes is None:
            # Socket cerrado antes de arrancar un nuevo batch
            # Significa que el cliente terminó de enviar batches
            return None

        batch_size = int.from_bytes(batch_size_bytes, "big")
        batch_bytes = self._recv_all(batch_size)
        for bet_bytes in batch_bytes.split(SEPARATOR_BETS.encode()):
            bet = Bet(*bet_bytes.decode().split(SEPARATOR))
            bets.append(bet)
        return bets

    def close(self):
        self._socket.close()