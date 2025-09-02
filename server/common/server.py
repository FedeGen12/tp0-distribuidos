import signal
import logging
from multiprocessing import Process, Lock
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
        self._lock_bets_file = Lock()

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
        agencies = []

        while self._running and len(agencies) < self._amount_clients:
            try:
                client_sock, client_id = self.__accept_new_connection()

                agency = Process(
                    name=str(client_id),
                    target=self.__handle_client_connection,
                    args=(client_sock, self._lock_bets_file),
                )

                agencies.append((agency, client_sock))
                agency.start()

            except OSError:
                # si se cerró el socket desde sigterm_handler → salir del loop
                break

        self._server_socket.close()
        self._get_winners(agencies)

        for agency, agency_socket in agencies:
            agency.join()
            agency_socket.close()

    def __handle_client_connection(self, client_sock, lock_bets_file):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        while True:
            type_message, message = client_sock.recv()

            if type_message == BATCH_MESSAGE:
                with lock_bets_file:
                    store_bets(message)
                    logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(message)}")

            elif type_message == NOTIFY_MESSAGE:
                logging.info(f"action: notificacion_recibida | result: success")
                break

    def _get_winners(self, agencies):
        if self._running:
            logging.info("action: sorteo | result: success")

            agency_winners = obtain_winners()

            for agency, agency_socket in agencies.items():
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
