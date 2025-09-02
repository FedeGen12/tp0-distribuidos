import logging
import socket
from common.utils import has_won, load_bets

def _obtain_agency_winners(agency_id, lock_bets_file):
    agency_winners = []

    with lock_bets_file:
        for bet in load_bets():
            if has_won(bet) and bet.agency == agency_id:
                agency_winners.append(int(bet.document))

    return agency_winners


def get_winners(agency, lock_bets_file, notify_barrier):
    if notify_barrier.wait() == 0:
        logging.info("action: sorteo | result: success")

    agency_winners = _obtain_agency_winners(agency.id, lock_bets_file)
    agency.socket.send_winners(agency_winners)


def send_all_by_socket(client_socket, bytes_to_send: bytes):
    total_bytes_sent = 0

    while total_bytes_sent < len(bytes_to_send):
        bytes_sent = client_socket.send(bytes_to_send[total_bytes_sent:], socket.MSG_WAITALL)
        if not bytes_sent:
            raise OSError("failed to sent data")
        total_bytes_sent += bytes_sent


def recv_all_by_socket(client_socket, bytes_to_recv: int):
    bytes_received = bytearray()

    while len(bytes_received) < bytes_to_recv:
        bytes_read = client_socket.recv(bytes_to_recv - len(bytes_received), socket.MSG_WAITALL)
        if not bytes_read:
            raise OSError("failed to receive data")
        bytes_received.extend(bytes_read)

    return bytes(bytes_received)