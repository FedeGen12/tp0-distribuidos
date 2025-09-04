import signal
import logging
import socket

from common.client_socket import ClientSocket, BATCH_MESSAGE, NOTIFY_MESSAGE
from common.utils import store_bets, has_won, load_bets
from common.server_socker import ServerSocket


def obtain_winners():
    agency_winners = {}
    for bet in load_bets():
        if bet.agency not in agency_winners:
            agency_winners[bet.agency] = []
        if has_won(bet):
            agency_winners[bet.agency].append(int(bet.document))
    return agency_winners

class Server:
    def __init__(self, port, listen_backlog, amount_clients):
        # Initialize server socket
        self._running = True
        self._amount_clients = amount_clients
        self._server_socket = ServerSocket.setup_listener('', port, listen_backlog)
        self._agencies = {}

        def sigterm_handler(_signum, _stacktrace):
            logging.info("action: shutdown | result: in_progress | msg: SIGTERM received")
            self._running = False
            try:
                self._server_socket.shutdown(socket.SHUT_RDWR)
                logging.info("action: close_server_socket | result: success")
                self.close_agency_sockets()
                logging.info("action: shutdown | result: success")
            except OSError as e:
                logging.error(f"action: close_server_socket | result: fail | error: {e}")
                logging.error(f"action: shutdown | result: fail")

        signal.signal(signal.SIGTERM, sigterm_handler)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        while self._running and len(self._agencies) < self._amount_clients:
            try:
                client_sock, client_id = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
                self._agencies[client_id] = client_sock
            except OSError:
                # si se cerró el socket desde sigterm_handler → salir del loop
                logging.info("action: shutdown del socket | result: success")
                return

        if not self._running:
            logging.info("action: shutdown_server | result: success")
            return

        self._server_socket.close()
        self._get_winners()
        self.close_agency_sockets()

    def close_agency_sockets(self):
        for agency_id, agency_socket in self._agencies.items():
            try:
                agency_socket.close()
                logging.info(f"action: close_client_socket | result: success | client_id: {agency_id}")
            except OSError as e:
                logging.error(f"action: close_client_socket | result: fail | error: {e}")

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        while True:
            type_message, message = client_sock.recv()

            if type_message == BATCH_MESSAGE:
                store_bets(message)
                logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(message)}")

            elif type_message == NOTIFY_MESSAGE:
                logging.info(f"action: notificacion_recibida | result: success")
                break

    def _get_winners(self):
        if self._running:
            logging.info("action: sorteo | result: success")

            agency_winners = obtain_winners()

            for agency, agency_socket in self._agencies.items():
                agency_socket.send_winners(agency_winners[agency])

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        client_socket, client_id, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return client_socket, client_id
