import logging
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