import signal
import socket
import logging


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._running = True
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

        def sigterm_handler(_signum, _stacktrace):
            logging.info("action: shutdown | result: in_progress | msg: SIGTERM received")
            self._running = False
            try:
                self._server_socket.close()
                logging.info("action: close_server_socket | result: success")
                logging.info("action: shutdown | result: success")
            except OSError as e:
                logging.error(f"action: close_server_socket | result: fail | error: {e}")
                logging.error(f"action: shutdown | result: fail")

        signal.signal(signal.SIGTERM, sigterm_handler)

        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

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
                break

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = client_sock.recv(1024).rstrip().decode('utf-8')
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {msg}')
            # TODO: Modify the send to avoid short-writes
            client_sock.send("{}\n".format(msg).encode('utf-8'))
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
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
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
