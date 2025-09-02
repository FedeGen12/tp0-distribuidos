import signal
import logging

from common.client_socket import ClientSocket, BATCH_MESSAGE, NOTIFY_MESSAGE
from common.utils import store_bets
from common.server_socker import ServerSocket


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._running = True
        self._amount_clients = 5
        self._server_socket = ServerSocket.setup_listener('', port, listen_backlog)

        def sigterm_handler(_signum, _stacktrace):
            logging.info("action: shutdown | result: in_progress | msg: SIGTERM received")
            self._running = False
            self._server_socket.close()

        signal.signal(signal.SIGTERM, sigterm_handler)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        agencies = {}

        while self._running and len(agencies) < self._amount_clients:
            try:
                client_sock, client_id = self.__accept_new_connection()
                agencies[client_id] = client_sock
                self.__handle_client_connection(client_sock)
            except OSError:
                # si se cerró el socket desde sigterm_handler → salir del loop
                break

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
