import signal
import logging
from multiprocessing import Process, Lock, Barrier
from common.client import BATCH_MESSAGE, NOTIFY_MESSAGE
from common.utils import store_bets
from common.server_socker import ServerSocket
from common.server_utils import get_winners

class Server:
    def __init__(self, port, listen_backlog, amount_clients):
        # Initialize server socket
        self._running = True
        self._amount_clients = amount_clients
        self._server_socket = ServerSocket.setup_listener('', port, listen_backlog)
        self._lock_bets_file = Lock()
        self._notify_barrier = Barrier(amount_clients)
        self.agencies = []

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

        while self._running and len(self.agencies) < self._amount_clients:
            try:
                client = self.__accept_new_connection()
                self.__start_agency_process(client)

            except OSError:
                # si se cerró el socket desde sigterm_handler → salir del loop
                break

        self._server_socket.close()

        for agency_process, agency in self.agencies:
            agency_process.join()
            agency.socket.close()

    def __handle_client_connection(self, client, lock_bets_file, notify_barrier):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        while True:
            type_message, message = client.socket.recv()

            if type_message == BATCH_MESSAGE:
                with lock_bets_file:
                    store_bets(message)
                    logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(message)}")

            elif type_message == NOTIFY_MESSAGE:
                logging.info(f"action: notificacion_recibida | result: success")
                get_winners(client, lock_bets_file, notify_barrier)
                break

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        client, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return client

    def __start_agency_process(self, client):
        agency = Process(
            name=str(client.id),
            target=self.__handle_client_connection,
            args=(client, self._lock_bets_file, self._notify_barrier),
        )

        self.agencies.append((agency, client))
        agency.start()
