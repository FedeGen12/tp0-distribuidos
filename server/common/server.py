import signal
import logging
import socket

from common.client_socket import ClientSocket
from common.utils import store_bets
from common.server_socker import ServerSocket


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._running = True
        self._server_socket = ServerSocket.setup_listener('', port, listen_backlog)

        def sigterm_handler(_signum, _stacktrace):
            logging.info("action: shutdown | result: in_progress | msg: SIGTERM received")
            self._running = False
            try:
                self._server_socket.shutdown(socket.SHUT_RDWR)
                logging.info("action: close_server_socket | result: success")
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

        while self._running:
            try:
                client_sock = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
            except OSError:
                # si se cerró el socket desde sigterm_handler → salir del loop
                logging.info("action: shutdown del accept | result: success")
                break

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        while True:
            try:
                bets, is_final = client_sock.recv()
                if bets is None:
                    break

                store_bets(bets)
                logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")

                if is_final:
                    logging.info("Último batch recibido de este cliente")
                    break
            except OSError as e:
                logging.error(f"action: apuesta_recibida | result: fail | error: {e}")
                break

        try:
            client_sock.close()
            logging.info("action: close_client_socket | result: success")
        except OSError as e:
            logging.error(f"action: close_client_socket | result: fail | error: {e}")

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        client_socket, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return client_socket
